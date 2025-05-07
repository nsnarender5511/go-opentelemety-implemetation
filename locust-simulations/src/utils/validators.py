import logging
from typing import List, Dict, Any, Callable, Optional
import json

logger = logging.getLogger("validators")

def validate_response(response, checks: List[Callable]) -> bool:
    """
    Run a series of validation checks on a response.
    
    Args:
        response: The HTTP response object
        checks: List of validator functions to run
        
    Returns:
        True if all checks pass, False otherwise
    """
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
    """
    Create a validator for status code.
    
    Args:
        expected_code: Expected HTTP status code
        
    Returns:
        Validator function
    """
    def _check(response) -> bool:
        if response.status_code != expected_code:
            logger.warning(f"Expected status {expected_code}, got {response.status_code}")
            return False
        return True
    return _check

def check_content_type(expected_type: str = "application/json") -> Callable:
    """
    Create a validator for content type.
    
    Args:
        expected_type: Expected content type
        
    Returns:
        Validator function
    """
    def _check(response) -> bool:
        content_type = response.headers.get('Content-Type', '')
        if expected_type not in content_type:
            logger.warning(f"Expected content type {expected_type}, got {content_type}")
            return False
        return True
    return _check

def check_json_contains(expected_fields: List[str]) -> Callable:
    """
    Create a validator that checks if JSON response contains required fields.
    
    Args:
        expected_fields: List of field names expected in the response
        
    Returns:
        Validator function
    """
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
    """
    Check if response contains a valid product schema.
    
    Args:
        response: HTTP response object
        
    Returns:
        True if response has valid product schema, False otherwise
    """
    try:
        product = response.json()
        required_fields = ["productID", "name", "price", "stock"]
        for field in required_fields:
            if field not in product:
                logger.warning(f"Product missing required field: {field}")
                return False
        return True
    except Exception as e:
        logger.warning(f"Product schema validation error: {e}")
        return False

def check_products_list_schema(response) -> bool:
    """
    Check if response contains a valid products list schema.
    
    Args:
        response: HTTP response object
        
    Returns:
        True if response has valid products list schema, False otherwise
    """
    try:
        data = response.json()
        
        # Check if response is a dict with "data" field (wrapper format)
        if isinstance(data, dict) and "data" in data:
            products = data["data"]
        # Or if response is a direct list
        elif isinstance(data, list):
            products = data
        else:
            logger.warning(f"Expected products list, got: {type(data)}")
            return False
        
        # Check if it's a list
        if not isinstance(products, list):
            logger.warning(f"Expected list of products, got: {type(products)}")
            return False
        
        # If list is empty, that's valid (could be filtering result)
        if not products:
            return True
        
        # Check first item for schema
        if not isinstance(products[0], dict):
            logger.warning(f"Expected product to be a dictionary")
            return False
        
        # Check minimal product fields in first item
        required_fields = ["productID", "name"]
        for field in required_fields:
            if field not in products[0]:
                logger.warning(f"Product list item missing required field: {field}")
                return False
                
        return True
        
    except Exception as e:
        logger.warning(f"Products list schema validation error: {e}")
        return False 