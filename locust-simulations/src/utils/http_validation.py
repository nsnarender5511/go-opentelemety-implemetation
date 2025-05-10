import logging
from typing import List, Dict, Any, Callable, Optional
import json

logger = logging.getLogger("validators")

def validate_response(response, checks: List[Callable]) -> bool:
    success = True
    for check in checks:
        try:
            check_result = check(response)
            if not check_result:
                success = False
        except Exception as e:
            logger.error(f"Validation error: {e}")
            success = False
    return success

def check_status_code(expected_code: int) -> Callable:
    def _check(response) -> bool:
        if response.status_code != expected_code:
            logger.warning(f"Expected status {expected_code}, got {response.status_code}")
            return False
        return True
    return _check

def check_content_type(expected_type: str = "application/json") -> Callable:
    def _check(response) -> bool:
        content_type = response.headers.get('Content-Type', '')
        if expected_type not in content_type:
            logger.warning(f"Expected content type {expected_type}, got {content_type}")
            return False
        return True
    return _check

def check_json_contains(expected_fields: List[str]) -> Callable:
    def _check(response) -> bool:
        try:
            data = response.json()
            for field in expected_fields:
                if field not in data:
                    logger.warning(f"Expected field '{field}' not found in response")
                    return False
            return True
        except Exception as e:
            logger.warning(f"JSON parsing error: {e}")
            return False
    return _check

def check_product_schema(response) -> bool:
    try:
        product = response.json()
        required_fields = ["name", "description", "price", "stock", "category"]
        for field in required_fields:
            if field not in product:
                logger.warning(f"Product missing required field: {field}")
                return False
        return True
    except Exception as e:
        logger.warning(f"Product schema validation error: {e}")
        return False

def check_products_list_schema(response) -> bool:
    try:
        data = response.json()
        products = None
        
        # Case 1: Object with product names as keys
        if isinstance(data, dict) and not "data" in data:
            # Check if at least one product exists and has required fields
            if not data:
                return True  # Empty response is valid
                
            # Get first product to check schema
            first_product_name = next(iter(data))
            first_product = data[first_product_name]
            
            if not isinstance(first_product, dict):
                logger.warning(f"Expected product to be a dictionary")
                return False
                
            # Check minimal product fields in first product
            required_fields = ["name", "description", "price"]
            for field in required_fields:
                if field not in first_product:
                    logger.warning(f"Product missing required field: {field}")
                    return False
            
            return True
        
        # Case 2: Response has a "data" field (wrapper format)
        elif isinstance(data, dict) and "data" in data:
            products = data["data"]
            
            # Handle case where data is an object with product names as keys
            if isinstance(products, dict):
                if not products:
                    return True  # Empty response is valid
                
                # Get first product to check schema
                first_product_name = next(iter(products))
                first_product = products[first_product_name]
                
                if not isinstance(first_product, dict):
                    logger.warning(f"Expected product to be a dictionary")
                    return False
                    
                required_fields = ["name", "description", "price"]
                for field in required_fields:
                    if field not in first_product:
                        logger.warning(f"Product missing required field: {field}")
                        return False
                
                return True
            
        # Case 3: Response data is a direct list
        elif isinstance(data, list):
            products = data
        else:
            logger.warning(f"Expected products list or object, got: {type(data)}")
            return False
        
        # For array responses, check if it's a list
        if products is not None and isinstance(products, list):
            # If list is empty, that's valid (could be filtering result)
            if not products:
                return True
            
            # Check first item for schema
            if not isinstance(products[0], dict):
                logger.warning(f"Expected product to be a dictionary")
                return False
            
            # Check minimal product fields in first item
            required_fields = ["name", "description", "price"]
            for field in required_fields:
                if field not in products[0]:
                    logger.warning(f"Product list item missing required field: {field}")
                    return False
        
        return True
        
    except Exception as e:
        logger.warning(f"Products list schema validation error: {e}")
        return False 