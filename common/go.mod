module github.com/narender/common

go 1.23.0

toolchain go1.24.1

require (
	github.com/joho/godotenv v1.5.1
	github.com/sirupsen/logrus v1.9.3
	go.opentelemetry.io/otel v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.35.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.35.0
	go.opentelemetry.io/otel/metric v1.35.0
	go.opentelemetry.io/otel/sdk v1.35.0
	go.opentelemetry.io/otel/sdk/metric v1.35.0
	go.opentelemetry.io/otel/trace v1.35.0
	google.golang.org/grpc v1.72.0
)

// Add replace directives for local sub-packages if used internally,
// although in this simple common module, it might not be strictly needed
// if sub-packages don't import each other heavily via the full path.
// Example if lifecycle used config:
// replace github.com/narender/common/config => ./config

require (
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	// github.com/go-viper/mapstructure/v2 v2.2.1 // indirect // Will be removed by go mod tidy
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.26.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.35.0 // indirect
	go.opentelemetry.io/proto/otlp v1.5.0 // indirect
	golang.org/x/net v0.35.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.5 // indirect
// gopkg.in/yaml.v3 v3.0.1 // indirect // Will be removed by go mod tidy
)

require (
	github.com/gofiber/contrib/otelfiber/v2 v2.2.2
	github.com/gofiber/fiber/v2 v2.52.6
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.11.0
	go.opentelemetry.io/otel/log v0.11.0
	go.opentelemetry.io/otel/sdk/log v0.11.0
)

require (
	github.com/andybalholm/brotli v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	go.opentelemetry.io/contrib v1.20.0 // indirect
)

// Add any *other* dependencies specific to the common module here if needed in the future

// Add any dependencies specific to the common module here if needed in the future
// require (
//    example.com/some/dependency v1.0.0
// )
