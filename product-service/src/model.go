package main

const (
	JSONFieldProductID = "productId"
	JSONFieldStock     = "stock"
)
const (
	JSONDataField       = "data"
	JSONPaginationField = "pagination"
)

type Product struct {
	ProductID   string   `json:"productID"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	Stock       int      `json:"stock"`
	Category    string   `json:"category"`
	Tags        []string `json:"tags"`
}
