#!/usr/bin/env python3
"""
Simple test server that simulates the product-service API
for testing the Locust implementation locally.
"""
from flask import Flask, request, jsonify
import logging
import time
import random

app = Flask(__name__)

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger('test-server')

# Sample product data
PRODUCTS = [
    {"productID": "1", "name": "Laptop", "price": 999.99, "stock": 50, "category": "Electronics"},
    {"productID": "2", "name": "Smartphone", "price": 599.99, "stock": 100, "category": "Electronics"},
    {"productID": "3", "name": "Headphones", "price": 149.99, "stock": 200, "category": "Electronics"},
    {"productID": "4", "name": "Coffee Table", "price": 199.99, "stock": 30, "category": "Furniture"},
    {"productID": "5", "name": "Sofa", "price": 899.99, "stock": 10, "category": "Furniture"},
    {"productID": "6", "name": "Coffee Mug", "price": 12.99, "stock": 0, "category": "Kitchenware"},
    {"productID": "7", "name": "Zero Stock Product", "price": 29.99, "stock": 0, "category": "Electronics"},
    {"productID": "8", "name": "Laptop Pro", "price": 1999.99, "stock": 5, "category": "Electronics"}
]

# Health check endpoint
@app.route('/health', methods=['GET'])
def health_check():
    logger.info("Health check request")
    return jsonify({"status": "ok"})

# Get all products
@app.route('/products', methods=['GET'])
def get_products():
    # Simulate some processing time
    time.sleep(random.uniform(0.05, 0.2))
    logger.info("Request for all products")
    return jsonify({"data": PRODUCTS})

# Get products by category
@app.route('/products/category', methods=['GET'])
def get_products_by_category():
    category = request.args.get('category', '')
    time.sleep(random.uniform(0.05, 0.15))
    
    if not category:
        return jsonify({"error": "Category parameter is required"}), 400
    
    filtered_products = [p for p in PRODUCTS if p.get("category") == category]
    logger.info(f"Request for products in category: {category}, found {len(filtered_products)}")
    
    return jsonify({"data": filtered_products})

# Get product details
@app.route('/products/details', methods=['POST'])
def get_product_details():
    data = request.json
    time.sleep(random.uniform(0.05, 0.1))
    
    if not data or 'name' not in data:
        return jsonify({"error": "Product name is required"}), 400
    
    product_name = data.get('name')
    product = next((p for p in PRODUCTS if p.get("name") == product_name), None)
    
    if not product:
        logger.info(f"Product details requested for non-existent product: {product_name}")
        return jsonify({"error": "Product not found"}), 404
    
    logger.info(f"Product details requested for: {product_name}")
    return jsonify(product)

# Buy a product
@app.route('/products/buy', methods=['POST'])
def buy_product():
    data = request.json
    time.sleep(random.uniform(0.1, 0.3))  # Simulate processing time
    
    if not data:
        return jsonify({"error": "Request body is required"}), 400
    
    if 'name' not in data:
        return jsonify({"error": "Product name is required"}), 400
    
    if 'quantity' not in data:
        return jsonify({"error": "Quantity is required"}), 400
    
    product_name = data.get('name')
    quantity = data.get('quantity')
    
    # Validate quantity
    try:
        quantity = int(quantity)
        if quantity <= 0:
            return jsonify({"error": "Quantity must be positive"}), 400
    except (ValueError, TypeError):
        return jsonify({"error": "Invalid quantity"}), 400
    
    # Find product
    product = None
    for i, p in enumerate(PRODUCTS):
        if p.get("name") == product_name:
            product = p
            product_index = i
            break
    
    if not product:
        logger.info(f"Attempted to buy non-existent product: {product_name}")
        return jsonify({"error": "Product not found"}), 404
    
    # Check if enough stock
    if product["stock"] < quantity:
        logger.info(f"Out of stock: {product_name}, requested: {quantity}, available: {product['stock']}")
        return jsonify({"error": "Insufficient stock"}), 409
    
    # Update stock
    PRODUCTS[product_index]["stock"] -= quantity
    
    logger.info(f"Successfully purchased {quantity} of {product_name}")
    return jsonify({
        "success": True,
        "message": f"Successfully purchased {quantity} of {product_name}",
        "product": PRODUCTS[product_index]
    })

# Update stock
@app.route('/products/stock', methods=['PATCH'])
def update_stock():
    data = request.json
    time.sleep(random.uniform(0.05, 0.2))
    
    if not data:
        return jsonify({"error": "Request body is required"}), 400
    
    if 'name' not in data:
        return jsonify({"error": "Product name is required"}), 400
    
    if 'stock' not in data:
        return jsonify({"error": "Stock is required"}), 400
    
    product_name = data.get('name')
    new_stock = data.get('stock')
    
    # Validate stock
    try:
        new_stock = int(new_stock)
        if new_stock < 0:
            return jsonify({"error": "Stock cannot be negative"}), 400
    except (ValueError, TypeError):
        return jsonify({"error": "Invalid stock value"}), 400
    
    # Find product
    product = None
    for i, p in enumerate(PRODUCTS):
        if p.get("name") == product_name:
            product = p
            product_index = i
            break
    
    if not product:
        logger.info(f"Attempted to update stock for non-existent product: {product_name}")
        return jsonify({"error": "Product not found"}), 404
    
    # Update stock
    old_stock = PRODUCTS[product_index]["stock"]
    PRODUCTS[product_index]["stock"] = new_stock
    
    logger.info(f"Updated stock for {product_name} from {old_stock} to {new_stock}")
    return jsonify({
        "success": True,
        "message": f"Stock updated for {product_name}",
        "product": PRODUCTS[product_index]
    })

if __name__ == '__main__':
    logger.info("Starting test server on http://localhost:8082")
    app.run(host='0.0.0.0', port=8082, debug=True, threaded=True) 