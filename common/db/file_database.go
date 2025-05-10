package db

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	"github.com/narender/common/globals"
	commontrace "github.com/narender/common/telemetry/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// FileDatabase provides methods to interact with a JSON file database.
type FileDatabase struct {
	filePath string
	logger   *slog.Logger
}

// NewFileDatabase creates a new instance of FileDatabase.
func NewFileDatabase() *FileDatabase {
	return &FileDatabase{
		filePath: globals.Cfg().PRODUCT_DATA_FILE_PATH,
		logger:   globals.Logger(),
	}
}

// Read loads data from the JSON file into the dest interface{}.
func (db *FileDatabase) Read(ctx context.Context, dest interface{}) (opErr error) {
	// Get request ID from context if available
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	// Start DB Span
	ctx, spanner := commontrace.StartSpan(ctx,
		"file_database",
		"read",
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("READ"),
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	db.logger.DebugContext(ctx, "Database file access initiated",
		slog.String("file_path", db.filePath),
		slog.String("request_id", requestID),
		slog.String("operation", "read_database"))

	fileContent, err := os.ReadFile(db.filePath)
	if err != nil {
		db.logger.ErrorContext(ctx, "Database file read error",
			slog.String("file_path", db.filePath),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("operation", "read_database"))
		opErr = err // Assign error to opErr
		return opErr
	}

	err = json.Unmarshal(fileContent, dest)
	if err != nil {
		db.logger.ErrorContext(ctx, "JSON parsing error",
			slog.String("file_path", db.filePath),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("operation", "parse_json"))
		opErr = err // Assign error to opErr
		return opErr
	}

	db.logger.DebugContext(ctx, "Database data read successfully",
		slog.String("file_path", db.filePath),
		slog.String("request_id", requestID),
		slog.String("operation", "read_database"))
	return nil // Success
}

// Write marshals the data interface{} to JSON and writes it to the file, overwriting existing content.
func (db *FileDatabase) Write(ctx context.Context, data interface{}) (opErr error) {
	// Get request ID from context if available
	var requestID string
	if id, ok := ctx.Value("requestID").(string); ok {
		requestID = id
	}

	// Start DB Span
	ctx, spanner := commontrace.StartSpan(ctx,
		"file_database",
		"write",
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("WRITE"),
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	db.logger.DebugContext(ctx, "Database file write initiated",
		slog.String("file_path", db.filePath),
		slog.String("request_id", requestID),
		slog.String("operation", "write_database"))

	jsonData, err := json.MarshalIndent(data, "", "  ") // Use MarshalIndent for readability
	if err != nil {
		db.logger.ErrorContext(ctx, "JSON serialization error",
			slog.String("file_path", db.filePath),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("operation", "serialize_json"))
		opErr = err // Assign error to opErr
		return opErr
	}

	err = os.WriteFile(db.filePath, jsonData, 0644) // 0644 provides read/write for owner, read for others
	if err != nil {
		db.logger.ErrorContext(ctx, "Database file write error",
			slog.String("file_path", db.filePath),
			slog.String("error", err.Error()),
			slog.String("request_id", requestID),
			slog.String("operation", "write_database"))
		opErr = err // Assign error to opErr
		return opErr
	}

	db.logger.DebugContext(ctx, "Database data written successfully",
		slog.String("file_path", db.filePath),
		slog.String("request_id", requestID),
		slog.String("operation", "write_database"))
	return nil // Success
}

// FilePath returns the path to the database file.
func (db *FileDatabase) FilePath() string {
	return db.filePath
}
