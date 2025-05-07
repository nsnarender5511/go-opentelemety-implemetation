"""
Validation utilities specific to master-store endpoints.
This module provides validation functions for master-store API responses.
"""
import logging
from typing import List, Dict, Any, Callable, Optional
import json

logger = logging.getLogger("master_store_validators")

def check_master_store_products_schema(response) -> bool:
    """
    Check if response contains a valid products list schema from master-store.
    
    Args:
        response: HTTP response object
        
    Returns:
        True if response has valid products list schema, False otherwise
    """
    try:
        data = response.json()
        
        # The master-store returns an object with product names as keys
        if not isinstance(data, dict):
            if isinstance(data, dict) and "data" in data:
                # It might be wrapped in a data field
                data = data["data"]
                if not isinstance(data, dict):
                    logger.warning(f"Expected a dictionary in data field, got: {type(data)}")
                    return False
            else:
                logger.warning(f"Expected a dictionary response, got: {type(data)}")
                return False
        
        # Empty response is valid
        if not data:
            return True
            
        # Get first product to check schema
        try:
            first_product_name = next(iter(data))
            first_product = data[first_product_name]
        except StopIteration:
            return True  # Empty dict is valid
        
        if not isinstance(first_product, dict):
            logger.warning(f"Expected product to be a dictionary")
            return False
            
        # Check required product fields
        required_fields = ["name", "description", "price", "stock", "category"]
        for field in required_fields:
            if field not in first_product:
                logger.warning(f"Master store product missing required field: {field}")
                return False
        
        return True
        
    except Exception as e:
        logger.warning(f"Master store products schema validation error: {e}")
        return False 