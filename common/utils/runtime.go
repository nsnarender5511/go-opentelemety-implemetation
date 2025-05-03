package utils

import "runtime"

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
	return fn.Name()
}
