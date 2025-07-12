package exporter

import (
	"fmt"
	"strings"

	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
)

type DocumentName struct {
	ProjectID      string
	DatabaseID     string
	CollectionName string
	DocumentID     string
}

func ParseDocumentName(name string) (*DocumentName, error) {
	// projects/{projectId}/databases/{databaseId}/documents/{collectionName}/{documentId}
	// ただし、 documentName に / が含まれる場合は、ネストした collectionName のため、サポートしないものとする。
	parts := strings.Split(name, "/")
	if len(parts) < 7 || parts[0] != "projects" || parts[2] != "databases" || parts[4] != "documents" {
		return nil, fmt.Errorf("document name must be in the format of projects/{projectId}/databases/{databaseId}/documents/{collectionName}/{documentId}: %v", name)
	}
	if len(parts) > 7 {
		return nil, fmt.Errorf("nested collection is not supported: %v", name)
	}
	return &DocumentName{
		ProjectID:      parts[1],
		DatabaseID:     parts[3],
		CollectionName: parts[5],
		DocumentID:     parts[6],
	}, nil
}

func ExtractDocumentFieldValue(value *firestoredata.Value) any {
	// https://cloud.google.com/firestore/docs/reference/rest/Shared.Types/ArrayValue#Value
	switch v := value.GetValueType().(type) {
	case *firestoredata.Value_NullValue:
		return nil
	case *firestoredata.Value_BooleanValue:
		return v.BooleanValue
	case *firestoredata.Value_IntegerValue:
		return v.IntegerValue
	case *firestoredata.Value_DoubleValue:
		return v.DoubleValue
	case *firestoredata.Value_TimestampValue:
		return v.TimestampValue
	case *firestoredata.Value_StringValue:
		return v.StringValue
	case *firestoredata.Value_BytesValue:
		return v.BytesValue
	case *firestoredata.Value_ReferenceValue:
		return v.ReferenceValue
	case *firestoredata.Value_GeoPointValue:
		return v.GeoPointValue
	case *firestoredata.Value_ArrayValue:
		values := make([]any, len(v.ArrayValue.Values))
		for i, value := range v.ArrayValue.Values {
			values[i] = ExtractDocumentFieldValue(value)
		}
		return values
	case *firestoredata.Value_MapValue:
		values := make(map[string]any, len(v.MapValue.Fields))
		for key, value := range v.MapValue.Fields {
			values[key] = ExtractDocumentFieldValue(value)
		}
		return values
	default:
		return nil
	}
}

type TableName struct {
	ProjectID string
	DatasetID string
	TableName string
}

func ParseTableName(name string) (*TableName, error) {
	parts := strings.Split(name, ".")
	switch len(parts) {
	case 2:
		return &TableName{
			DatasetID: parts[0],
			TableName: parts[1],
		}, nil
	case 3:
		return &TableName{
			ProjectID: parts[0],
			DatasetID: parts[1],
			TableName: parts[2],
		}, nil
	}
	return nil, fmt.Errorf("invalid table name: %s", name)
}
