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
	// Start DB Span
	ctx, spanner := commontrace.StartSpan(ctx,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("READ"),
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	db.logger.DebugContext(ctx, "FileDB: Reading data from file", slog.String("file_path", db.filePath))

	fileContent, err := os.ReadFile(db.filePath)
	if err != nil {
		db.logger.ErrorContext(ctx, "FileDB: Failed to read data file", slog.String("file_path", db.filePath), slog.Any("error", err))
		opErr = err // Assign error to opErr
		return opErr
	}

	err = json.Unmarshal(fileContent, dest)
	if err != nil {
		db.logger.ErrorContext(ctx, "FileDB: Failed to unmarshal JSON data", slog.String("file_path", db.filePath), slog.Any("error", err))
		opErr = err // Assign error to opErr
		return opErr
	}

	db.logger.DebugContext(ctx, "FileDB: Data read and unmarshalled successfully", slog.String("file_path", db.filePath))
	return nil // Success
}

// Write marshals the data interface{} to JSON and writes it to the file, overwriting existing content.
func (db *FileDatabase) Write(ctx context.Context, data interface{}) (opErr error) {
	// Start DB Span
	ctx, spanner := commontrace.StartSpan(ctx,
		semconv.DBSystemKey.String("file"),
		semconv.DBOperationKey.String("WRITE"),
	)
	defer commontrace.EndSpan(spanner, &opErr, nil)

	db.logger.DebugContext(ctx, "FileDB: Writing data to file", slog.String("file_path", db.filePath))

	jsonData, err := json.MarshalIndent(data, "", "  ") // Use MarshalIndent for readability
	if err != nil {
		db.logger.ErrorContext(ctx, "FileDB: Failed to marshal data to JSON", slog.String("file_path", db.filePath), slog.Any("error", err))
		opErr = err // Assign error to opErr
		return opErr
	}

	err = os.WriteFile(db.filePath, jsonData, 0644) // 0644 provides read/write for owner, read for others
	if err != nil {
		db.logger.ErrorContext(ctx, "FileDB: Failed to write data file", slog.String("file_path", db.filePath), slog.Any("error", err))
		opErr = err // Assign error to opErr
		return opErr
	}

	db.logger.DebugContext(ctx, "FileDB: Data written successfully", slog.String("file_path", db.filePath))
	return nil // Success
}

// FilePath returns the path to the database file.
func (db *FileDatabase) FilePath() string {
	return db.filePath
}
