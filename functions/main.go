package firestore_to_bigquery

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/damsys/firestore_to_bigquery/functions/exporter"
)

func init() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#logseverity
			if a.Key == slog.LevelKey {
				return slog.Attr{
					Key:   "severity",
					Value: a.Value,
				}
			}
			return a
		},
		Level: slog.LevelDebug,
	})))
	exportConfigStr := os.Getenv("EXPORT_CONFIG")
	if exportConfigStr == "" {
		exportConfigStr = "{}"
	}
	exportConfig := exporter.ExportConfig{}
	if err := json.Unmarshal([]byte(exportConfigStr), &exportConfig); err != nil {
		panic(err)
	}
	exporter, err := exporter.New(context.Background(), &exportConfig)
	if err != nil {
		panic(err)
	}
	functions.CloudEvent("firestoreToBigQuery", exporter.ExportHandler)
}
