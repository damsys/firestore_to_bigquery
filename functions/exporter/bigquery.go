package exporter

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/apiv1/storagepb"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
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
func (e *Exporter) UpsertRowWithStorageAPI(ctx context.Context, row map[string]any, cache *exportCache) error {
	// https://github.com/googleapis/google-cloud-go/blob/7a46b5428f239871993d66be2c7c667121f60a6f/bigquery/storage/managedwriter/integration_test.go#L397
	if err := e.setupDynamicDescriptors(ctx, cache); err != nil {
		return fmt.Errorf("failed to setup dynamic descriptors: %w", err)
	}

	// ManagedWriter を作成
	writer, err := e.managedWriterClient.NewManagedStream(
		ctx,
		managedwriter.WithDestinationTable(managedwriter.TableParentFromParts(
			cache.table.ProjectID,
			cache.table.DatasetID,
			cache.table.TableID,
		)),
		managedwriter.WithSchemaDescriptor(cache.descriptorProto),
		managedwriter.WithType(managedwriter.DefaultStream),
	)
	if err != nil {
		return fmt.Errorf("failed to create managed writer: %w", err)
	}
	defer writer.Close()

	// 時刻は microseconds に変換
	for k, v := range row {
		switch vv := v.(type) {
		case time.Time:
			row[k] = vv.UnixMicro()
		case *time.Time:
			if vv != nil {
				row[k] = vv.UnixMicro()
			}
		}
	}
	raw, err := json.Marshal(row)
	if err != nil {
		return fmt.Errorf("failed to marshal json message: %w", err)
	}
	message := dynamicpb.NewMessage(cache.messageDescriptor)
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
