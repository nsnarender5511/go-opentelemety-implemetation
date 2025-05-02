package otel

import (
	"sync"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/metric"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
)


type TelemetryManager struct {
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	loggerProvider *sdklog.LoggerProvider
	tracer         oteltrace.Tracer 
	meter          metric.Meter     
	logger         *logrus.Logger

	serviceName    string
	serviceVersion string
}

var (
	globalManager *TelemetryManager
	once          sync.Once
	managerMutex  sync.RWMutex 
)






func initializeGlobalManager(tp *sdktrace.TracerProvider, mp *sdkmetric.MeterProvider, lp *sdklog.LoggerProvider, log *logrus.Logger, serviceName, serviceVersion string) {
	
	once.Do(func() {
		managerMutex.Lock() 
		defer managerMutex.Unlock()

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

		
		if globalManager.logger != nil {
			globalManager.logger.Info("Global TelemetryManager initialized.")
		}
	})
}




func GetTracer(instrumentationName string) oteltrace.Tracer {
	managerMutex.RLock() 
	defer managerMutex.RUnlock()

	if globalManager == nil || globalManager.tracerProvider == nil {
		
		logrus.Warn("GetTracer called before TelemetryManager initialization. Returning no-op tracer.")
		return tracenoop.NewTracerProvider().Tracer(instrumentationName)
	}
	
	return globalManager.tracerProvider.Tracer(instrumentationName, oteltrace.WithInstrumentationVersion(globalManager.serviceVersion))
}




func GetMeter(instrumentationName string) metric.Meter {
	managerMutex.RLock() 
	defer managerMutex.RUnlock()

	if globalManager == nil || globalManager.meterProvider == nil {
		
		logrus.Warn("GetMeter called before TelemetryManager initialization. Returning no-op meter.")
		return metricnoop.NewMeterProvider().Meter(instrumentationName)
	}
	
	return globalManager.meterProvider.Meter(instrumentationName, metric.WithInstrumentationVersion(globalManager.serviceVersion))
}




func GetLogger() *logrus.Logger {
	managerMutex.RLock() 
	defer managerMutex.RUnlock()

	if globalManager == nil || globalManager.logger == nil {
		
		logrus.Warn("GetLogger called before TelemetryManager initialization. Returning default logger.")
		
		return logrus.StandardLogger()
	}
	return globalManager.logger
}




func GetLoggerProvider() *sdklog.LoggerProvider {
	managerMutex.RLock()
	defer managerMutex.RUnlock()
	if globalManager == nil {
		logrus.Warn("GetLoggerProvider called before TelemetryManager initialization.")
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
