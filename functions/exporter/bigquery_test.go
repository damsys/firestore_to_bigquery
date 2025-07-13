package exporter

import (
	"encoding/json"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestSetupDynamicDescriptors(t *testing.T) {
	bqSchema := bigquery.Schema{
		{
			Name:     "ID",
			Type:     bigquery.StringFieldType,
			Required: true,
		},
		{
			Name: "Name",
			Type: bigquery.StringFieldType,
		},
		{
			Name: "Age",
			Type: bigquery.IntegerFieldType,
		},
		{
			Name: "CreatedAt",
			Type: bigquery.TimestampFieldType,
		},
	}
	t.Logf("bqSchema: %#v\n", bqSchema)
	schema, err := adapt.BQSchemaToStorageTableSchema(bqSchema)
	if err != nil {
		t.Fatalf("failed to convert schema: %v", err)
	}
	t.Logf("schema: %#v\n", schema)
	descriptor, err := adapt.StorageSchemaToProto2Descriptor(schema, "root")
	if err != nil {
		t.Fatalf("failed to convert schema: %v", err)
	}
	t.Logf("descriptor: %v\n", descriptor)
	messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		t.Fatalf("adapted descriptor is not a message descriptor")
	}
	t.Logf("messageDescriptor: %v\n", messageDescriptor)
	descriptorProto, err := adapt.NormalizeDescriptor(messageDescriptor)
	if err != nil {
		t.Fatalf("failed to normalize descriptor: %v", err)
	}
	t.Logf("descriptorProto: %v\n", descriptorProto)

	row := map[string]any{
		"ID":        "1",
		"Name":      "John Doe",
		"Age":       30,
		"CreatedAt": time.Now().UTC(),
	}
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
		t.Fatalf("failed to marshal json message: %v", err)
	}
	t.Logf("raw: %v\n", string(raw))

	message := dynamicpb.NewMessage(messageDescriptor)
	err = protojson.Unmarshal(raw, message)
	if err != nil {
		t.Fatalf("failed to unmarshal json message: %v", err)
	}
	t.Logf("message: %#v\n", message)
	// _CHANGE_TYPE は unknown になっている
	message.SetUnknown(protoreflect.RawFields([]byte(`{"_CHANGE_TYPE":"INSERT"}`)))
	b, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("failed to marshal to proto byte message: %v", err)
	}
	t.Logf("b: %#v\n", b)
}
