import logging
from typing import List, Dict, Any, Callable, Optional, Union
# import json # Already commented/removed, ensure it stays that way if present

# Remove the problematic relative import of BaseAPIUser
# from ..base_user import BaseAPIUser # REMOVE THIS LINE

# Import the new utility function
from .product_parser import parse_products_from_data # Assuming product_parser.py is in the same 'utils' directory

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

def check_status_code(expected_codes: Union[int, List[int]]) -> Callable:
    def _check(response) -> bool:
        if isinstance(expected_codes, list):
            # Handle a list of expected codes
            if response.status_code not in expected_codes:
                logger.warning(f"Expected status to be one of {expected_codes}, got {response.status_code}")
                return False
        else:
            # Handle a single expected code (integer)
            if response.status_code != expected_codes:
                logger.warning(f"Expected status {expected_codes}, got {response.status_code}")
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
        json_payload = response.json()
        # Use the imported utility function to parse products
        products = parse_products_from_data(json_payload, logger) 
        
        if products is None: 
            response.failure("Failed to parse product data for schema check (parser returned None)")
            return False
            
        if not products:  
            return True
        
        first_product = products[0]
        if not isinstance(first_product, dict):
            logger.warning(f"Product in list is not a dictionary. Got: {type(first_product)}")
            response.failure("Product in list is not a dictionary.")
            return False
        
        required_fields = ["name", "description", "price"]
        for field in required_fields:
            if field not in first_product:
                logger.warning(f"Product list item missing required field: '{field}'. Product: {first_product}")
                response.failure(f"Product list item missing required field: '{field}'")
                return False
        return True
        
    except Exception as e:
        logger.warning(f"Products list schema validation error: {e} - Response text: {response.text[:500]}")
        response.failure(f"Products list schema validation error: {e}")
        return False 