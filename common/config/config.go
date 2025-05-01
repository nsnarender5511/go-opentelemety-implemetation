package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	PRODUCT_SERVICE_PORT string
	LOG_LEVEL            string
	LOG_FORMAT           string
	OTEL_SERVICE_NAME    string
	SERVICE_NAME         string
	SERVICE_VERSION      string
	// OTEL_EXPORTER_OTLP_ENDPOINT: Target for OTLP traces, metrics, logs.
	// Default value in .env.default assumes Docker network 'signoz-net' and a collector service named 'otel-collector'.
	OTEL_EXPORTER_OTLP_ENDPOINT string
	OTEL_EXPORTER_INSECURE      string
	OTEL_SAMPLE_RATIO           string
	DATA_FILE_PATH              string // Path to the product data JSON file
)

func init() {
	_ = godotenv.Load("../.env.default")
	_ = godotenv.Load("../.env")

	PRODUCT_SERVICE_PORT = os.Getenv("PRODUCT_SERVICE_PORT")
	LOG_LEVEL = os.Getenv("LOG_LEVEL")
	LOG_FORMAT = os.Getenv("LOG_FORMAT")
	OTEL_SERVICE_NAME = os.Getenv("OTEL_SERVICE_NAME")
	SERVICE_VERSION = os.Getenv("SERVICE_VERSION")
	OTEL_EXPORTER_OTLP_ENDPOINT = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	OTEL_EXPORTER_INSECURE = os.Getenv("OTEL_EXPORTER_INSECURE")
	OTEL_SAMPLE_RATIO = os.Getenv("OTEL_SAMPLE_RATIO")
	DATA_FILE_PATH = os.Getenv("DATA_FILE_PATH") // Load data file path

	SERVICE_NAME = OTEL_SERVICE_NAME

	log.Println("Configuration initialized.")
	log.Printf("Using data file path: %s", DATA_FILE_PATH) // Log the path being used
}
