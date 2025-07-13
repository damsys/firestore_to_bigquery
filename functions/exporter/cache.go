package exporter

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type exportCache struct {
	table             *bigquery.Table
	messageDescriptor protoreflect.MessageDescriptor
	descriptorProto   *descriptorpb.DescriptorProto
}

func (e *Exporter) newExportCache(tablename string) (*exportCache, error) {
	tableName, err := ParseTableName(tablename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse table name: %w", err)
	}
	table := e.bigqueryClient.Dataset(tableName.DatasetID).Table(tableName.TableName)
	return &exportCache{table: table}, nil
}

func (e *Exporter) setupDynamicDescriptors(ctx context.Context, cache *exportCache) error {
	// https://github.com/googleapis/google-cloud-go/blob/7a46b5428f239871993d66be2c7c667121f60a6f/bigquery/storage/managedwriter/integration_test.go#L100
	if cache.messageDescriptor != nil && cache.descriptorProto != nil {
		return nil
	}
	metadata, err := cache.table.Metadata(ctx)
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
		return fmt.Errorf("adapted descriptor is not a message descriptor")
	}
	descriptorProto, err := adapt.NormalizeDescriptor(messageDescriptor)
	if err != nil {
		return fmt.Errorf("failed to normalize descriptor: %w", err)
	}
	cache.messageDescriptor = messageDescriptor
	cache.descriptorProto = descriptorProto
	return nil
}
