package exporter

import (
	"context"
	"fmt"
	"log/slog"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
	"google.golang.org/protobuf/proto"
)

type Exporter struct {
	config              *ExportConfig
	bigqueryClient      *bigquery.Client
	managedWriterClient *managedwriter.Client
}

type ExportConfig struct {
	Rules map[string]*ExportRule `json:"rules"`
}

type ExportRule struct {
	Table       string `json:"table"`
	cachedTable *bigquery.Table
	Fields      []string `json:"fields"`
}

func New(ctx context.Context, config *ExportConfig) (*Exporter, error) {
	bigqueryClient, err := bigquery.NewClient(ctx, getProjectID(bigquery.DetectProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create bigquery client: %w", err)
	}
	managedWriterClient, err := managedwriter.NewClient(
		ctx,
		getProjectID(managedwriter.DetectProjectID),
		managedwriter.WithMultiplexing(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create managed writer client: %w", err)
	}

	return &Exporter{
		config:              config,
		bigqueryClient:      bigqueryClient,
		managedWriterClient: managedWriterClient,
	}, nil
}

func (e *Exporter) Close() error {
	if e.managedWriterClient != nil {
		err := e.managedWriterClient.Close()
		if err != nil {
			slog.Error("failed to close managed writer client", slog.String("error", err.Error()))
		}
	}
	if e.bigqueryClient != nil {
		err := e.bigqueryClient.Close()
		if err != nil {
			slog.Error("failed to close bigquery client", slog.String("error", err.Error()))
		}
	}
	return nil
}

// ExportHandler は Firestore からの変更イベントを受け取って BigQuery に Upsert する
func (e *Exporter) ExportHandler(ctx context.Context, event event.Event) error {
	var data firestoredata.DocumentEventData

	slog.InfoContext(ctx, "event info", slog.String("event_type", event.Type()))

	// If you omit `DiscardUnknown`, protojson.Unmarshal returns an error
	// when encountering a new or unknown field.
	options := proto.UnmarshalOptions{
		DiscardUnknown: true,
	}

	err := options.Unmarshal(event.Data(), &data)
	if err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal event data", slog.String("error", err.Error()))
		return nil
	}

	err = e.exportToBigQuery(ctx, &data)
	if err != nil {
		slog.ErrorContext(ctx, "failed to export to bigquery", slog.String("error", err.Error()))
	}
	return nil
}

func (e *Exporter) exportToBigQuery(ctx context.Context, data *firestoredata.DocumentEventData) error {
	name, err := ParseDocumentName(data.GetValue().GetName())
	if err != nil {
		return err
	}

	kind := name.CollectionName
	// docID := name.DocumentID

	if e.config.Rules == nil {
		slog.WarnContext(ctx, "no rules found")
		return nil
	}
	rule := e.config.Rules[kind]
	if rule == nil {
		// Nothing to do
		return nil
	}

	// BigQuery へ送信するデータを整形
	id := data.GetValue().Fields["ID"].GetStringValue()
	if id == "" {
		slog.WarnContext(
			ctx,
			"ID is empty",
			slog.String("documentName", data.GetValue().GetName()),
		)
	}
	row := make(map[string]any, len(rule.Fields)+1)
	row["ID"] = id

	for _, field := range rule.Fields {
		row[field] = ExtractDocumentFieldValue(data.GetValue().Fields[field])
	}

	// BigQuery へ挿入
	table := rule.cachedTable
	if table == nil {
		tableName, err := ParseTableName(rule.Table)
		if err != nil {
			return fmt.Errorf("failed to parse table name: %w", err)
		}
		if tableName.ProjectID == "" {
			table = e.bigqueryClient.Dataset(tableName.DatasetID).Table(tableName.TableName)
		} else {
			table = e.bigqueryClient.DatasetInProject(tableName.ProjectID, tableName.DatasetID).Table(tableName.TableName)
		}
		rule.cachedTable = table
	}
	// if err := e.InsertRow(ctx, table, row); err != nil {
	if err := e.UpsertRowWithStorageAPI(ctx, table, row); err != nil {
		return fmt.Errorf("failed to upsert row: %w", err)
	}
	//}
	return nil
}
