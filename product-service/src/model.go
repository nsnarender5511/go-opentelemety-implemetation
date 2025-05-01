package main

import (
	"time"
)

// Constants for JSON field names
const (
	JSONFieldProductID = "productId"
	JSONFieldStock     = "stock"
	JSONFieldError     = "error"
)

// Product defines the structure for product data, simplified for in-memory
type Product struct {
	ID          uint      `json:"-"`
	ProductID   string    `json:"productId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
