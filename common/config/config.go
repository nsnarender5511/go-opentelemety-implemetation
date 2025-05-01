package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	PRODUCT_SERVICE_PORT        string
	LOG_LEVEL                   string
	LOG_FORMAT                  string
	OTEL_SERVICE_NAME           string
	SERVICE_NAME                string
	SERVICE_VERSION             string
	OTEL_EXPORTER_OTLP_ENDPOINT string
	OTEL_EXPORTER_INSECURE      string
	OTEL_SAMPLE_RATIO           string
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

	SERVICE_NAME = OTEL_SERVICE_NAME

	log.Println("Configuration initialized.")
}
