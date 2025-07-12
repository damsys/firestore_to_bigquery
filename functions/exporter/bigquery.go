package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/apiv1/storagepb"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InsertRow は指定テーブルに1行を挿入する
func (e *Exporter) InsertRow(ctx context.Context, table *bigquery.Table, row map[string]any) error {
	ins := table.Inserter()
	if err := ins.Put(ctx, []map[string]any{row}); err != nil {
		return fmt.Errorf("failed to insert row: %w", err)
	}
	return nil
}

// UpsertRowWithStorageAPI は指定テーブルに1行を Upsert する（BigQuery Storage API を使用）
func (e *Exporter) UpsertRowWithStorageAPI(ctx context.Context, table *bigquery.Table, row map[string]any) error {
	metadata, err := table.Metadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to get table metadata: %w", err)
	}
	schema, err := adapt.BQSchemaToStorageTableSchema(metadata.Schema)
	if err != nil {
		return fmt.Errorf("failed to convert schema: %w", err)
	}
	descriptor, err := adapt.StorageSchemaToProto2Descriptor(schema, "root")
	if err != nil {
		return fmt.Errorf("failed to convert schema: %w", err)
	}
	messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		return fmt.Errorf("failed to convert schema: %w", err)
	}
	descriptorProto, err := adapt.NormalizeDescriptor(messageDescriptor)
	if err != nil {
		return fmt.Errorf("failed to normalize descriptor: %w", err)
	}
	// ManagedWriter を作成
	writer, err := e.managedWriterClient.NewManagedStream(
		ctx,
		managedwriter.WithType(managedwriter.DefaultStream),
		managedwriter.WithDestinationTable(managedwriter.TableParentFromParts(table.ProjectID, table.DatasetID, table.TableID)),
		managedwriter.WithSchemaDescriptor(descriptorProto),
	)
	if err != nil {
		return fmt.Errorf("failed to create managed writer: %w", err)
	}
	defer writer.Close()

	// 時刻を
	for k, v := range row {
		switch vv := v.(type) {
		case time.Time:
			row[k] = timestamppb.New(vv)
		case *time.Time:
			if vv != nil {
				row[k] = timestamppb.New(*vv)
			}
		}
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("failed to marshal json message: %w", err)
	}
	message := dynamicpb.NewMessage(messageDescriptor)
	err = protojson.Unmarshal(raw, message)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json message: %w", err)
	}
	b, err := proto.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal to proto byte message: %w", err)
	}

	rows := [][]byte{b}

	// データを書き込み
	res, err := writer.AppendRows(ctx, rows)
	if err != nil {
		return fmt.Errorf("failed to append rows: %w", err)
	}
	_, err = res.GetResult(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait append rows: %w", err)
	}
	_, err = writer.Finalize(ctx)
	if err != nil {
		return fmt.Errorf("failed to finalize writer: %w", err)
	}

	req := &storagepb.BatchCommitWriteStreamsRequest{
		Parent:       managedwriter.TableParentFromStreamName(writer.StreamName()),
		WriteStreams: []string{writer.StreamName()},
	}
	commitRes, err := e.managedWriterClient.BatchCommitWriteStreams(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to commit write streams: %w", err)
	}
	errs := commitRes.GetStreamErrors()
	if len(errs) > 0 {
		return fmt.Errorf("failed to commit write streams: %v", errs)
	}

	return nil
}
