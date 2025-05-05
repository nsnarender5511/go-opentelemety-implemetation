package apirequests

// Requires: go get github.com/go-playground/validator/v10 📦

// Used for GetProductByName
type GetByNameRequest struct {
	Name string `json:"name" validate:"required"` // Mark name as required
}

// Used for UpdateProductStock
type UpdateStockRequest struct {
	Name  string `json:"name" validate:"required"`
	Stock int    `json:"stock" validate:"required,gte=0"` // Stock must be provided and >= 0
}

// Used for BuyProduct
type ProductBuyRequest struct {
	Name     string `json:"name" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,gt=0"` // Quantity must be provided and > 0
}

// Note: GetProductsByCategory uses query param, validation handled separately (in handler)
