package apierrors

// Business error codes
const (
	// Product Domain Errors
	ErrCodeProductNotFound    = "PRODUCT_NOT_FOUND"    // When product doesn't exist
	ErrCodeInsufficientStock  = "INSUFFICIENT_STOCK"   // When purchase quantity exceeds stock
	ErrCodeInvalidProductData = "INVALID_PRODUCT_DATA" // When product information is invalid
	ErrCodeOrderLimitExceeded = "ORDER_LIMIT_EXCEEDED" // When purchase exceeds allowed quantity
	ErrCodePriceMismatch      = "PRICE_MISMATCH"       // When expected and actual prices don't match
)
