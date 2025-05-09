package apierrors

// Application error codes
const (
	// System Errors
	ErrCodeDatabaseAccess     = "DATABASE_ACCESS_ERROR"     // Database interaction failures
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"       // When a dependency is unavailable
	ErrCodeRequestValidation  = "REQUEST_VALIDATION_ERROR"  // Input validation failures
	ErrCodeInternalProcessing = "INTERNAL_PROCESSING_ERROR" // Logic execution failures
	ErrCodeResourceConstraint = "RESOURCE_CONSTRAINT_ERROR" // Resource limitations (rate limits, etc.)

	// Unexpected Errors
	ErrCodeSystemPanic    = "SYSTEM_PANIC"    // Recovered panics
	ErrCodeNetworkError   = "NETWORK_ERROR"   // Network-related failures
	ErrCodeMalformedData  = "MALFORMED_DATA"  // Invalid data formats (JSON parse errors, etc.)
	ErrCodeRequestTimeout = "REQUEST_TIMEOUT" // Operation timeouts
	ErrCodeUnknown        = "UNKNOWN_ERROR"   // Fallback for unclassified errors
)

// Deprecated error codes - for backward compatibility
const (
	ErrCodeNotFound   = ErrCodeProductNotFound
	ErrCodeValidation = ErrCodeRequestValidation
	ErrCodeDatabase   = ErrCodeDatabaseAccess
	ErrCodeInternal   = ErrCodeInternalProcessing
)
