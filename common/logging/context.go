package logging

import (
	"context"

	"go.uber.org/zap"
)



type loggerKeyType struct{}


var loggerKey = loggerKeyType{}


func NewContextWithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if logger == nil {
		
		
		
		return ctx
	}
	return context.WithValue(ctx, loggerKey, logger)
}




func LoggerFromContext(ctx context.Context) *zap.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok && logger != nil {
			return logger
		}
	}

	
	
	base := GetBaseLogger()
	if base == nil {
		
		
		
		return zap.NewNop()
	}
	
	
	return base
}
