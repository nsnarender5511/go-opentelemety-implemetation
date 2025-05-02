package main

import (
	"errors"
	"fmt"
	"log"
)

type ProductService interface {
	GetAll() ([]Product, error)
	GetByID(productID string) (Product, error)
}

type productService struct {
	repo ProductRepository
}

func NewProductService(repo ProductRepository) ProductService {
	return &productService{
		repo: repo,
	}
}

func (s *productService) GetAll() ([]Product, error) {
	products, repoErr := s.repo.GetAll()
	if repoErr != nil {
		log.Printf("ERROR: Service: Repository error during GetAllProducts: %v", repoErr)
		return nil, repoErr
	}

	fmt.Printf("INFO: Service: GetAll completed successfully, returning %d products\n", len(products))
	return products, nil
}

func (s *productService) GetByID(productID string) (Product, error) {
	product, repoErr := s.repo.GetByID(productID)
	if repoErr != nil {
		if errors.Is(repoErr, ErrNotFound) {
			fmt.Printf("WARN: Service: Product not found for ID %s\n", productID)
			return Product{}, ErrNotFound
		}
		log.Printf("ERROR: Service: Repository error during GetProductByID for ID %s: %v", productID, repoErr)
		return Product{}, repoErr
	}

	fmt.Printf("INFO: Service: GetByID completed successfully for productID: %s\n", productID)
	return product, nil
}
