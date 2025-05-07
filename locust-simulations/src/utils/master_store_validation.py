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
        
        # First check if we have a standard API response format with data field
        if isinstance(data, dict) and "data" in data and "status" in data:
            # Extract the actual data
            data = data["data"]
        
        # Master-store returns an array of products
        if isinstance(data, list):
            # Empty response is valid
            if not data:
                return True
                
            # Check first product
            if len(data) > 0:
                first_product = data[0]
                if not isinstance(first_product, dict):
                    logger.warning(f"Expected product to be a dictionary, got {type(first_product)}")
                    return False
                    
                # Check required product fields
                required_fields = ["name", "description", "price", "stock", "category"]
                for field in required_fields:
                    if field not in first_product:
                        logger.warning(f"Master store product missing required field: {field}")
                        return False
                
                return True
        else:
            logger.warning(f"Expected an array of products, got {type(data)}")
            return False
        
    except Exception as e:
        logger.warning(f"Master store products schema validation error: {e}")
        return False 