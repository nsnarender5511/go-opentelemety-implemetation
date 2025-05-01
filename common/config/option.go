package config

// Option is a function that configures a Config
type Option func(*Config)

// WithServiceName sets the service name
func WithServiceName(name string) Option {
	return func(c *Config) {
		c.ServiceName = name
	}
}

// WithServiceVersion sets the service version
func WithServiceVersion(version string) Option {
	return func(c *Config) {
		c.ServiceVersion = version
	}
}

// WithOtelEndpoint sets the OpenTelemetry exporter endpoint
func WithOtelEndpoint(endpoint string) Option {
	return func(c *Config) {
		c.OtelEndpoint = endpoint
	}
}

// WithOtelInsecure sets whether the OpenTelemetry exporter uses an insecure connection
func WithOtelInsecure(insecure bool) Option {
	return func(c *Config) {
		c.OtelInsecure = insecure
	}
}

// WithOtelSampleRatio sets the OpenTelemetry sampling ratio
func WithOtelSampleRatio(ratio float64) Option {
	return func(c *Config) {
		c.OtelSampleRatio = ratio
	}
}

// WithLogLevel sets the log level
func WithLogLevel(level string) Option {
	return func(c *Config) {
		c.LogLevel = level
	}
}

// WithLogFormat sets the log format
func WithLogFormat(format string) Option {
	return func(c *Config) {
		c.LogFormat = format
	}
}

// WithProductServicePort sets the product service port
func WithProductServicePort(port string) Option {
	return func(c *Config) {
		c.ProductServicePort = port
	}
}

// WithDataFilePath sets the data file path
func WithDataFilePath(path string) Option {
	return func(c *Config) {
		c.DataFilePath = path
	}
}
