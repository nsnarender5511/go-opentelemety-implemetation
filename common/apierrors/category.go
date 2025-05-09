package apierrors

// ErrorCategory distinguishes between different types of errors
type ErrorCategory string

const (
	// CategoryBusiness represents errors related to business rules violations
	CategoryBusiness ErrorCategory = "business"

	// CategoryApplication represents technical and infrastructure errors
	CategoryApplication ErrorCategory = "application"
)
