import os
import logging
from locust import events
import time

logger = logging.getLogger("monitoring")

# Flag to track if OpenTelemetry is initialized
otel_initialized = False

# Try to import OpenTelemetry packages
# This is in a try/except block to make it optional
try:
    from opentelemetry import trace
    from opentelemetry.sdk.trace import TracerProvider
    from opentelemetry.sdk.trace.export import BatchSpanProcessor
    from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
    from opentelemetry.sdk.resources import Resource, SERVICE_NAME
    
    def setup_opentelemetry():
        """Initialize OpenTelemetry based on environment variables."""
        global otel_initialized
        
        # Only initialize once
        if otel_initialized:
            return
            
        # Check environment variables for enablement and endpoint
        telemetry_enabled = os.environ.get("OTEL_ENABLED", "").lower() in ("true", "1", "yes")
        otel_endpoint = os.environ.get("OTEL_ENDPOINT")
            
        # If not enabled or no endpoint provided, don't initialize
        if not telemetry_enabled:
            logger.info("OpenTelemetry integration disabled (OTEL_ENABLED is not true)")
            return
            
        if not otel_endpoint:
            logger.info("OpenTelemetry endpoint not configured (OTEL_ENDPOINT missing), integration disabled")
            return
            
        try:
            # Create resource with service info
            service_name = os.environ.get("SERVICE_NAME", "locust-load-tester")
            resource = Resource(attributes={
                SERVICE_NAME: service_name
            })
            
            # Set up tracer provider
            tracer_provider = TracerProvider(resource=resource)
            trace.set_tracer_provider(tracer_provider)
            
            # Configure exporter
            otlp_exporter = OTLPSpanExporter(endpoint=otel_endpoint, insecure=True)
            span_processor = BatchSpanProcessor(otlp_exporter)
            tracer_provider.add_span_processor(span_processor)
            
            logger.info(f"OpenTelemetry initialized for service '{service_name}' with endpoint: {otel_endpoint}")
            otel_initialized = True
            
            # Register Locust event handlers for telemetry
            _register_telemetry_handlers()
            
        except Exception as e:
            logger.error(f"Failed to initialize OpenTelemetry: {e}")
    
    def _register_telemetry_handlers():
        """Register Locust event handlers for OpenTelemetry integration."""
        tracer = trace.get_tracer(__name__)
        
        @events.request.add_listener
        def on_request(request_type, name, response_time, response_length, exception, **kwargs):
            """Create spans for each request."""
            with tracer.start_as_current_span(f"{request_type} {name}") as span:
                span.set_attribute("http.method", request_type)
                span.set_attribute("http.url", name)
                span.set_attribute("http.response_time_ms", response_time)
                
                if exception:
                    span.set_attribute("error", True)
                    span.set_attribute("error.message", str(exception))
                else:
                    response = kwargs.get("response")
                    if response:
                        span.set_attribute("http.status_code", response.status_code)
        
        @events.test_start.add_listener
        def on_test_start(environment, **kwargs):
            """Create span for test start."""
            with tracer.start_as_current_span("locust_test_start") as span:
                span.set_attribute("test.type", "load_test")
                span.set_attribute("test.start_time", time.time())
                
                # Add info about the test configuration
                user_classes = getattr(environment, "user_classes", [])
                if user_classes:
                    class_names = [cls.__name__ for cls in user_classes]
                    span.set_attribute("test.user_classes", ",".join(class_names))
                
                target_host = getattr(environment, "host", "unknown")
                span.set_attribute("test.target_host", target_host)
        
        @events.test_stop.add_listener
        def on_test_stop(environment, **kwargs):
            """Create span for test completion."""
            with tracer.start_as_current_span("locust_test_stop") as span:
                span.set_attribute("test.end_time", time.time())
                
                # Add test results
                stats = environment.stats
                if stats:
                    total_requests = stats.total.num_requests
                    total_failures = stats.total.num_failures
                    
                    span.set_attribute("test.total_requests", total_requests)
                    span.set_attribute("test.total_failures", total_failures)
                    
                    if total_requests > 0:
                        success_rate = (total_requests - total_failures) / total_requests * 100
                        span.set_attribute("test.success_rate", success_rate)
    
except ImportError:
    # If OpenTelemetry packages are not available, provide stub functions
    logger.warning("OpenTelemetry packages not installed, telemetry features disabled")
    
    def setup_opentelemetry():
        """Stub function when OpenTelemetry is not available."""
        logger.warning("OpenTelemetry packages not installed, telemetry features disabled") 