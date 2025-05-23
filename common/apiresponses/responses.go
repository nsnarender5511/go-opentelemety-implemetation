package apiresponses

import "time"

// Standard Success Response Envelope
type SuccessResponse struct {
	Status    string      `json:"status"` // Always "success"
	Data      interface{} `json:"data"`   // Payload
	Timestamp string      `json:"timestamp,omitempty"`
}

// Standard Error Response Envelope (used by middleware)
type ErrorResponse struct {
	Status string      `json:"status"` // Always "error"
	Error  ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code      string `json:"code"`    // Application-specific error code
	Message   string `json:"message"` // User-friendly message
	Timestamp string `json:"timestamp,omitempty"`
}

// Helper to create a success response
func NewSuccessResponse(data interface{}) SuccessResponse {
	return SuccessResponse{
		Status:    "success",
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// Optional: Define common success data structures
type ActionConfirmation struct {
	Message string `json:"message"`
}
