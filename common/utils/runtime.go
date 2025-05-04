package utils

import (
	"runtime"
	"strings"
)

// GetCallerFunctionName retrieves the name of the calling function.
// skip determines how many stack frames to ascend.
func GetCallerFunctionName(skip int) string {
	pc := make([]uintptr, 1)
	n := runtime.Callers(skip, pc)
	if n == 0 {
		// Consider logging an error here as well
		return "<unknown>"
	}
	fn := runtime.FuncForPC(pc[0])
	if fn == nil {
		// Consider logging an error here as well
		return "<unknown>"
	}
	// Consider simplifying the name if needed (e.g., strip package path)
	fullFuncName := fn.Name()
	operationName := fullFuncName // Declare and initialize operationName
	if lastDotIndex := strings.LastIndexByte(fullFuncName, '.'); lastDotIndex != -1 {
		operationName = fullFuncName[lastDotIndex+1:] // Assign within if
	}
	return operationName
}
