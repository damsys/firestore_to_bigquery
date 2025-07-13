package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"cloud.google.com/go/pubsub"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
	"google.golang.org/protobuf/proto"
)

type Exporter struct {
	config              *ExportConfig
	bigqueryClient      *bigquery.Client
	managedWriterClient *managedwriter.Client
	pubsubClient        *pubsub.Client
}

type ExportConfig struct {
	Rules map[string]*ExportRule `json:"rules"`
}

type ExportRule struct {
	Table  string   `json:"table"`
	Fields []string `json:"fields"`
	Topic  string   `json:"topic"`
	cache  *exportCache
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
	pubsubClient, err := pubsub.NewClient(ctx, getProjectID(pubsub.DetectProjectID))
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	return &Exporter{
		config:              config,
		bigqueryClient:      bigqueryClient,
		managedWriterClient: managedWriterClient,
		pubsubClient:        pubsubClient,
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

func hasIntersect(a, b []string) bool {
	for _, v := range a {
		for _, v2 := range b {
			if v == v2 {
				return true
			}
		}
	}
	return false
}

func (e *Exporter) exportToBigQuery(ctx context.Context, data *firestoredata.DocumentEventData) error {
	eventType := DetectEventType(data)

	var documentID string
	if eventType == EventTypeDelete {
		documentID = data.GetOldValue().GetName()
	} else {
		documentID = data.GetValue().GetName()
	}

	name, err := ParseDocumentName(documentID)
	if err != nil {
		return err
	}

	kind := name.CollectionName

	if e.config.Rules == nil {
		slog.WarnContext(ctx, "no rules found")
		return nil
	}
	rule := e.config.Rules[kind]
	if rule == nil {
		// Nothing to do
		return nil
	}
	// 更新時には同期対象が更新されたフィールドのみを送信する
	if data.GetUpdateMask() != nil {
		if !hasIntersect(rule.Fields, data.GetUpdateMask().GetFieldPaths()) {
			return nil
		}
	}

	// 同期用ドキュメントの構築
	row := make(map[string]any, len(rule.Fields))
	for _, field := range rule.Fields {
		value := data.GetValue().Fields[field]
		if value == nil {
			continue
		}
		row[field] = ExtractDocumentFieldValue(data.GetValue().Fields[field])
	}
	// 更新条件の設定
	if eventType == EventTypeDelete {
		row["_CHANGE_TYPE"] = "DELETE"
	} else {
		row["_CHANGE_TYPE"] = "UPSERT"
	}
	body, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("failed to marshal row: %w", err)
	}

	// データの送信
	topic := e.pubsubClient.Topic(rule.Topic)
	_, err = topic.Publish(ctx, &pubsub.Message{
		Data: body,
	}).Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	/*

		// BigQuery へ挿入
		if rule.cache == nil {
			rule.cache, err = e.newExportCache(rule.Table)
			if err != nil {
				return fmt.Errorf("failed to create export cache: %w", err)
			}
		}

		// if err := e.InsertRow(ctx, table, row); err != nil {
		if err := e.UpsertRowWithStorageAPI(ctx, row, rule.cache); err != nil {
			return fmt.Errorf("failed to upsert row: %w", err)
		}
		//}
	*/

	return nil
}
