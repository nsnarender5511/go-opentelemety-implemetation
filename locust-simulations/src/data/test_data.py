"""
Test data constants for load testing scenarios.
"""
from typing import Dict, List, Any

# Edge case products for testing specific scenarios
EDGE_CASE_PRODUCTS = [
    {"name": "Zero Stock Product", "quantity": 1, "expected_status": 409},  # Should fail due to stock
    {"name": "Laptop Pro", "quantity": 1000, "expected_status": 400},       # Very large order
    {"name": "Coffee Mug", "quantity": 0, "expected_status": 400},          # Zero quantity
    {"name": "NonExistentProduct", "quantity": 1, "expected_status": 404}   # Non-existent product
]

# Categories with expected results
CATEGORIES = [
    # Category name, expected min items, expected status
    {"name": "Electronics", "min_items": 1, "expected_status": 200},
    {"name": "NonExistentCategory", "min_items": 0, "expected_status": 200},  # Should return empty list
    {"name": "Furniture", "min_items": 1, "expected_status": 200}
]

# Invalid API requests for testing error handling
INVALID_REQUESTS = [
    # Path, method, expected status
    {"path": "/products/invalid", "method": "GET", "expected_status": 404},
    {"path": "/products/buy", "method": "GET", "expected_status": 405},  # Method not allowed
    {"path": "/invalid/endpoint", "method": "GET", "expected_status": 404}
]

# Stock update scenarios
STOCK_UPDATES = [
    {"name": "Laptop", "stock": 100, "expected_status": 200},
    {"name": "Phone", "stock": 50, "expected_status": 200},
    {"name": "NonExistentProduct", "stock": 10, "expected_status": 404},
    {"name": "Headphones", "stock": -10, "expected_status": 400}  # Negative stock should fail validation
]

# Test user profiles for weighted tasks
USER_PROFILES = {
    "browser": {
        "browse_products": 10,
        "search_by_category": 7,
        "view_product_details": 8,
        "buy_product": 2,
        "update_stock": 0
    },
    "shopper": {
        "browse_products": 5,
        "search_by_category": 5,
        "view_product_details": 8,
        "buy_product": 10,
        "update_stock": 0
    },
    "admin": {
        "browse_products": 3,
        "search_by_category": 2,
        "view_product_details": 5,
        "buy_product": 0,
        "update_stock": 10
    }
}

# Products to always include in initial data load
SEED_PRODUCTS = [
    {"name": "Laptop", "category": "Electronics", "price": 999.99, "stock": 50},
    {"name": "Smartphone", "category": "Electronics", "price": 599.99, "stock": 100},
    {"name": "Headphones", "category": "Electronics", "price": 149.99, "stock": 200},
    {"name": "Coffee Table", "category": "Furniture", "price": 199.99, "stock": 30},
    {"name": "Sofa", "category": "Furniture", "price": 899.99, "stock": 10},
    {"name": "Coffee Mug", "category": "Kitchenware", "price": 12.99, "stock": 0}  # Zero stock for testing
] 