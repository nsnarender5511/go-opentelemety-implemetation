package models

const (
	JSONFieldStock = "stock"
)
const (
	JSONDataField       = "data"
	JSONPaginationField = "pagination"
)

type Product struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Category    string  `json:"category"`
}
