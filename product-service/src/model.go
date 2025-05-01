package main

// Constants for JSON field names
const (
	JSONFieldProductID = "productId"
	JSONFieldStock     = "stock"
	JSONFieldError     = "error"
)

// Product defines the structure for product data, simplified for in-memory
type Product struct {
	ProductID   string   `json:"productId"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Stock       int      `json:"stock"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}
