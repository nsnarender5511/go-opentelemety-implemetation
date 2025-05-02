package manager

import (
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

type TelemetryManager struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
	tracer         oteltrace.Tracer
	meter          metric.Meter
	logger         *zap.Logger

	serviceName    string
	serviceVersion string
}

var (
	globalManager *TelemetryManager
	once          sync.Once
	managerMutex  sync.RWMutex
)

func InitializeGlobalManager(tp *sdktrace.TracerProvider, mp *sdkmetric.MeterProvider, lp *sdklog.LoggerProvider, log *zap.Logger, serviceName, serviceVersion string) {
	once.Do(func() {
		managerMutex.Lock()
		defer managerMutex.Unlock()

		if log == nil {
			log = zap.NewNop()
		}

		globalManager = &TelemetryManager{
			tracerProvider: tp,
			meterProvider:  mp,
			loggerProvider: lp,
			logger:         log,
			serviceName:    serviceName,
			serviceVersion: serviceVersion,

			tracer: func() oteltrace.Tracer {
				if tp != nil {
					return tp.Tracer(serviceName, oteltrace.WithInstrumentationVersion(serviceVersion))
				}
				return tracenoop.Tracer{}
			}(),
			meter: func() metric.Meter {
				if mp != nil {
					return mp.Meter(serviceName, metric.WithInstrumentationVersion(serviceVersion))
				}
				return metricnoop.Meter{}
			}(),
		}

		if tp != nil {
			otel.SetTracerProvider(tp)
		} else {
			otel.SetTracerProvider(tracenoop.NewTracerProvider())
		}
		if mp != nil {
			otel.SetMeterProvider(mp)
		} else {
			otel.SetMeterProvider(metricnoop.NewMeterProvider())
		}
		if lp != nil {
			global.SetLoggerProvider(lp)
		} else {

		}

		globalManager.logger.Info("Global TelemetryManager initialized.")
	})
}

func GetTracer(instrumentationName string) oteltrace.Tracer {
	managerMutex.RLock()
	defer managerMutex.RUnlock()

	if globalManager == nil || globalManager.tracer == nil {
		tempLogger := zap.NewNop()
		if globalManager != nil && globalManager.logger != nil {
			tempLogger = globalManager.logger
		}
		tempLogger.Warn("GetTracer called before TelemetryManager initialization or tracer is nil. Returning no-op tracer.")
		return tracenoop.Tracer{}
	}

	if instrumentationName != "" && instrumentationName != globalManager.serviceName {
		if globalManager.tracerProvider != nil {
			return globalManager.tracerProvider.Tracer(instrumentationName, oteltrace.WithInstrumentationVersion(globalManager.serviceVersion))
		}
	}
	return globalManager.tracer
}

func GetMeter(instrumentationName string) metric.Meter {
	managerMutex.RLock()
	defer managerMutex.RUnlock()

	if globalManager == nil || globalManager.meter == nil {
		tempLogger := zap.NewNop()
		if globalManager != nil && globalManager.logger != nil {
			tempLogger = globalManager.logger
		}
		tempLogger.Warn("GetMeter called before TelemetryManager initialization or meter is nil. Returning no-op meter.")
		return metricnoop.Meter{}
	}

	if instrumentationName != "" && instrumentationName != globalManager.serviceName {
		if globalManager.meterProvider != nil {
			return globalManager.meterProvider.Meter(instrumentationName, metric.WithInstrumentationVersion(globalManager.serviceVersion))
		}
	}
	return globalManager.meter
}

func GetLogger() *zap.Logger {
	managerMutex.RLock()
	defer managerMutex.RUnlock()

	if globalManager == nil || globalManager.logger == nil {
		return zap.NewNop()
	}
	return globalManager.logger
}

func GetLoggerProvider() *sdklog.LoggerProvider {
	managerMutex.RLock()
	defer managerMutex.RUnlock()
	if globalManager == nil {
		tempLogger := zap.NewNop()
		if globalManager != nil && globalManager.logger != nil {
			tempLogger = globalManager.logger
		}
		tempLogger.Warn("GetLoggerProvider called before TelemetryManager initialization.")
		return nil
	}
	return globalManager.loggerProvider
}

func GetTracerProvider() *sdktrace.TracerProvider {
	managerMutex.RLock()
	defer managerMutex.RUnlock()
	if globalManager == nil {
		return nil
	}
	return globalManager.tracerProvider
}

func GetMeterProvider() metric.MeterProvider {
	managerMutex.RLock()
	defer managerMutex.RUnlock()
	if globalManager == nil {
		return nil
	}
	return globalManager.meterProvider
}
