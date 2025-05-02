1. Schizophrenic Logging Setup: The Two-Headed Logger Beast
This isn't subtle. You have two distinct packages attempting to solve the same problem: integrating Zap with OpenTelemetry logs.

common/log/setup.go:
Uses github.com/uptrace/opentelemetry-go-extra/otelzap. This is a known, third-party bridge library.
Initializes a global logger L *otelzap.Logger.
This is the one actually used in product-service/src/main.go:
Go

// Signoz_assignment/product-service/src/main.go
import commonlog "github.com/narender/common/log"
// ...
if err := commonlog.Init(cfg); err != nil {
    log.Fatalf("Failed to initialize logger: %v", err)
}
defer commonlog.Cleanup()
// ... lots of usage like commonlog.L.Info(...)
common/logging/setup.go:
Uses github.com/agoda-com/opentelemetry-go/otelzap and github.com/agoda-com/opentelemetry-logs-go. This appears to be an alternative OTel Logs SDK implementation combined with a Zap bridge.
It tries to initialize an OTel Log Provider (agodalogprovider) and tee the Zap core with an otelzap.NewOtelCore.
Crucially, this package seems completely unused. Nowhere in product-service or the rest of common does common/logging appear to be imported or its functions called.
Code Analysis:

main.go clearly initializes and uses common/log.
Code throughout the service (repository.go, service.go, handler.go, middleware/error_handler.go, middleware/request_logger.go) uses commonlog.L, which comes from the common/log package.
The common/logging package is effectively dead code. It might be a remnant of a previous attempt or a misunderstanding.
Why It's Terrible:

Confusion: Anyone new to the codebase (or even you, six months from now) will waste time figuring out which setup is active and why the other exists.
Maintenance Hazard: Dependencies are still listed (presumably in go.mod, though not provided), bloating your dependency tree. If someone tries to "fix" or use common/logging, they could break the actual logging setup.
Lack of Clarity: It demonstrates indecision or sloppy cleanup, eroding confidence in the codebase's quality.
The Fix (Again):

DELETE the entire common/logging directory. Remove its dependencies. Eradicate it.
Ensure the common/log package clearly reflects why uptrace/opentelemetry-go-extra/otelzap was chosen and how it's configured.
2. Reliance on Non-Standard/Potentially Stale Log Bridges: Choosing Your Crutches
Okay, so you're actually using uptrace/opentelemetry-go-extra/otelzap. Let's dissect that choice and the ghost of agoda-com.

Log Bridge Role: OpenTelemetry aims to decouple your application code from specific observability backends. For logging, this means:
Your app uses a logging library (like Zap).
An OTel "Log Bridge" or "Appender" intercepts logs from that library.
It converts them into the OTel Log Data Model.
It enriches them with OTel context (trace ID, span ID, resource attributes).
It passes them to the configured OTel Log SDK Exporter (usually OTLP).
uptrace/opentelemetry-go-extra/otelzap: This library acts as a wrapper around Zap, providing the bridge functionality. It's maintained by Uptrace (a commercial observability vendor) but is open-source. It's a viable option, but still third-party.
agoda-com/opentelemetry-logs-go: This looks like an attempt to implement the OTel Logs SDK itself, separate from the official go.opentelemetry.io/otel/sdk/log (which was unstable but is now Beta/Stable for basic use). Relying on a completely separate SDK implementation is far riskier than using a bridge library. It's highly likely to diverge from the official OTel specification. Its existence in your codebase, even unused, is concerning.
Modern Standard Approach:
Use a standard logging library (zap or Go's native slog).
Use the official OTel Go SDK (go.opentelemetry.io/otel/sdk and go.opentelemetry.io/otel/sdk/log).
Use a bridge from the official go.opentelemetry.io/contrib repository if available, or a well-maintained third-party bridge like the Uptrace one if necessary. For slog, there's go.opentelemetry.io/contrib/bridges/otelslog.
Code/Config Analysis:

Your common/log/setup.go uses the Uptrace bridge.
Your common/telemetry/setup.go initializes the standard OTel Trace and Meter providers (sdktrace and sdkmetric) but does not appear to initialize the standard OTel sdk/log provider. This implies the Uptrace bridge might be directly exporting or using its own embedded logic, bypassing the standard OTel log SDK pipeline. This needs verification by digging into the otelzap implementation or documentation.
Your otel-collector-config.yaml is configured to receive OTLP logs (receivers: [otlp] in the logs pipeline), process them (batch), and export them (otlp, logging). This part is fine and standard. It will accept logs in the OTLP format regardless of how the application constructed them, but using a standard SDK/bridge ensures they are constructed correctly according to the OTel Log Data Model.
Why It's Terrible (The agoda-com Ghost & Bridge Choice):

Risk: The unused agoda-com SDK represents a significant potential compatibility deviation from OTel standards. Using third-party bridges like Uptrace's is generally safer than a whole third-party SDK, but still less ideal than official contrib bridges if available and suitable.
Complexity: The Uptrace bridge might hide the standard OTel Log SDK configuration, making the overall OTel setup less transparent.
The Fix:

Confirm agoda-com is gone.
Recommended: Migrate from Zap+Uptrace bridge to Go's standard slog + go.opentelemetry.io/contrib/bridges/otelslog. This aligns with modern Go practices and OTel standards.
Alternative: If sticking with Zap, thoroughly understand how uptrace/opentelemetry-go-extra/otelzap integrates. Does it use the standard OTel Log SDK exporter pipeline, or does it export directly? Configure it explicitly to use the standard pipeline if possible for better consistency with trace/metric configuration.
3. Inconsistent Trace Context in Logs: The Unseen Thread
Observability's power comes from correlation. Logs are useful, traces are useful, but correlated logs and traces are observability gold. This means every log event ideally includes the trace_id and span_id of the operation it occurred within.

Code Analysis:

The Good: You are using the context-aware logger in many places:
Go

// Signoz_assignment/product-service/src/repository.go
logger := commonlog.L.Ctx(ctx) // Gets context-aware logger
logger.Info("Repository: GetAll called")
// ...
ctx, span := trace.StartSpan(ctx, ...) // Starts a span, adding it to ctx
logger.Info("Repository: GetAll returning products", ...) // Log call uses logger derived from ctx
The uptrace/opentelemetry-go-extra/otelzap bridge should automatically extract the trace_id and span_id from the ctx passed to .Ctx(ctx) and inject them into the log record it generates.
The Bad: There are instances where the base, non-contextual logger appears to be used, especially during initialization or in areas where context might not be readily available:
Go

// Signoz_assignment/product-service/src/main.go
commonlog.L.Info("Logger, Telemetry, and Common Metrics initialized.") // No ctx here
commonlog.L.Info("Initializing application dependencies...")        // No ctx here
commonlog.L.Info("Setting up Fiber application...")                // No ctx here
commonlog.L.Info("Middleware configured.")                           // No ctx here
commonlog.L.Info("API routes configured.")                           // No ctx here
go func() {
    commonlog.L.Info("Server starting", zap.String("address", addr)) // No ctx here (goroutine)
    // ...
}()
commonlog.L.Info("Shutdown signal received...")                    // No ctx here
commonlog.L.Info("Attempting to shut down Fiber server...")        // No ctx here
While some of these (like shutdown) occur outside active spans, initialization logs could potentially be associated with a startup span if you created one. More importantly, any logging within request handlers or service logic that accidentally uses commonlog.L.Info(...) instead of commonlog.L.Ctx(ctx).Info(...) will lose the trace correlation.
Middleware: Your RequestLoggerMiddleware uses commonlog.L.Ctx(c.UserContext()), which is correct for capturing context within the middleware itself. Your ErrorHandler also correctly uses otelLogger.Ctx(ctx), which is good.
Why It's Terrible:

Debugging Blind Spots: When a log entry lacks trace context, you can't easily find it when viewing a distributed trace. You see the trace, you see an error or latency, but you can't jump to the specific log messages generated during that exact request span that might explain why. This drastically increases debugging time.
Incomplete Picture: It breaks the unified view of observability that OTel promises.
The Fix:

Consistency is Mandatory: Make it a strict rule: Always use the context-aware logger (logger.Ctx(ctx).Info(...)) within any code block that operates within a request or trace context (ctx).
Static Analysis: Consider using static analysis tools (linters) to flag usages of the global logger (commonlog.L) where a context (ctx) is available in the scope.
Startup Span: Consider creating a dedicated OTel span during application startup (main) to associate initialization logs with it, using commonlog.L.Ctx(startupCtx).Info(...).
Review Goroutines: Be extra careful with logging in new goroutines. Ensure the context.Context (containing the span) is passed to the goroutine if you want logs within it to be correlated.
Your Docker and Collector configurations facilitate the transport of logs, but these fundamental issues lie within the application's instrumentation code – how the logs are generated and enriched (or not) before they even leave the service. Clean up the structure, standardize the tooling, and enforce context propagation religiously.