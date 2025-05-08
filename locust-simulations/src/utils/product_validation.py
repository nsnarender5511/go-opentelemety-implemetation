"""
Unified validation utilities for product data across all services.
This module provides consolidated validation functions for product API responses.
"""
import logging
from typing import List, Dict, Any, Callable, Optional, Union

logger = logging.getLogger("product_validators")

def check_product_schema(response, expected_format="auto", required_fields=None):
    """
    Unified product schema validator that handles different response formats
    
    Args:
        response: HTTP response object
        expected_format: One of "auto", "array", "dict", or "wrapped"
        required_fields: List of required fields in each product (defaults to standard set)
        
    Returns:
        True if response has valid schema, False otherwise
    """
    if required_fields is None:
        required_fields = ["name", "description", "price", "stock", "category"]
        
    try:
        data = response.json()
        original_data = data  # Keep a copy of the original data structure
        
        # Handle standard API envelope
        if isinstance(data, dict) and "data" in data and "status" in data:
            data = data["data"]
            
        # Auto-detect format if not specified
        if expected_format == "auto":
            if isinstance(data, list):
                expected_format = "array"
            elif isinstance(data, dict) and not any(k in ["data", "status", "error"] for k in data):
                expected_format = "dict"
        
        # Empty response validation
        if (isinstance(data, list) and not data) or (isinstance(data, dict) and not data):
            return True  # Empty data is valid
            
        # Product validation based on format
        if expected_format == "array" or isinstance(data, list):
            # Array format validation
            if not isinstance(data, list):
                logger.warning(f"Expected an array of products, got {type(data)}")
                return False
                
            if not data:
                return True  # Empty array is valid
                
            # Check first product
            first_product = data[0]
            if not isinstance(first_product, dict):
                logger.warning(f"Expected product to be a dictionary, got {type(first_product)}")
                return False
                
            # Check required product fields
            for field in required_fields:
                if field not in first_product:
                    logger.warning(f"Product missing required field: {field}")
                    return False
                    
            return True
            
        elif expected_format == "dict" or isinstance(data, dict):
            # Dictionary format validation
            if not isinstance(data, dict):
                logger.warning(f"Expected a dictionary of products, got {type(data)}")
                return False
                
            if not data:
                return True  # Empty dict is valid
                
            # Get first product to check schema
            try:
                first_product_name = next(iter(data))
                first_product = data[first_product_name]
            except StopIteration:
                return True  # Empty dict is valid
                
            if not isinstance(first_product, dict):
                logger.warning(f"Expected product to be a dictionary, got {type(first_product)}")
                return False
                
            # Check required product fields
            for field in required_fields:
                if field not in first_product:
                    logger.warning(f"Product missing required field: {field}")
                    return False
                    
            return True
            
        else:
            logger.warning(f"Unsupported response format: {type(data)}")
            return False
            
    except Exception as e:
        logger.warning(f"Product schema validation error: {e}")
        return False 