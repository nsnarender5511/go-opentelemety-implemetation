package apiresponses

// Standard Success Response Envelope
type SuccessResponse struct {
	Status string      `json:"status"` // Always "success"
	Data   interface{} `json:"data"`   // Payload
}

// Standard Error Response Envelope (used by middleware)
type ErrorResponse struct {
	Status string      `json:"status"` // Always "error"
	Error  ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`    // Application-specific error code (e.g., "PRODUCT_NOT_FOUND")
	Message string `json:"message"` // User-friendly message
}

// Helper to create a success response
func NewSuccessResponse(data interface{}) SuccessResponse {
	return SuccessResponse{
		Status: "success",
		Data:   data,
	}
}

// Optional: Define common success data structures
type ActionConfirmation struct {
	Message string `json:"message"`
}
