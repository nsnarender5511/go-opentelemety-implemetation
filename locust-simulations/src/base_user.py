"""
Base User for Load Testing
This file defines the BaseAPIUser class with common functionality for all API users.
"""
import os
import logging
import time
from typing import Dict, Any, List, Optional, Callable

from locust import HttpUser, between, events
from locust.clients import ResponseContextManager

from src.utils.shared_data import SharedData
from src.utils.http_validation import validate_response, check_status_code, check_content_type

# Configure logging
logger = logging.getLogger("base_user")

class BaseAPIUser(HttpUser):
    """
    Base class with common functionality for all API users.
    """
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {}
        self.use_nginx_proxy = os.environ.get("USE_NGINX_PROXY", "false").lower() == "true"
        self.service_prefix = ""  # Default empty prefix, to be overridden by subclasses
        self.service_name = ""    # Default empty name, to be overridden by subclasses
        
    def on_start(self):
        """Initialize shared data on start"""
        self.shared_data = SharedData()  # Use global instance
    
    def _get_path(self, endpoint):
        """
        Convert endpoint path based on whether nginx proxy is used
        
        Args:
            endpoint: The endpoint path (e.g., "/health")
            
        Returns:
            The full path with prefix if using nginx proxy
        """
        if self.use_nginx_proxy and self.service_prefix:
            # For nginx proxy, prepend /prefix/
            return f"/{self.service_prefix}{endpoint}"
        # Direct access
        return endpoint
    
    def _can_execute_task(self, task_name):
        """
        Return True if the task can be executed, False otherwise
        
        Args:
            task_name: The name of the task to check
            
        Returns:
            True if task can be executed, False otherwise
        """
        max_executions_arg = f"max_{task_name}"
        max_executions = getattr(self.environment.parsed_options, max_executions_arg, -1) if hasattr(self.environment, "parsed_options") else -1
        
        if max_executions == 0:
            return False
        if max_executions < 0:
            return True
        current_count = self.task_counts.get(task_name, 0)
        return current_count < max_executions
    
    def _increment_task_count(self, task_name):
        """
        Increment the task count and return the new value
        
        Args:
            task_name: The name of the task to increment
            
        Returns:
            The new count after incrementing
        """
        self.task_counts[task_name] = self.task_counts.get(task_name, 0) + 1
        return self.task_counts[task_name]
    
    def _extract_products(self, response: ResponseContextManager) -> List[Dict[str, Any]]:
        """
        Extract products from a response in a consistent way regardless of format
        
        Args:
            response: HTTP response object
            
        Returns:
            List of product dictionaries
        """
        try:
            data = response.json()
            
            # Case 1: Standard API wrapper with data field
            if isinstance(data, dict) and "data" in data and "status" in data:
                data = data["data"]
                
            # Case 2: Response is a direct list of products
            if isinstance(data, list):
                return data
                
            # Case 3: Response is a dictionary with product names as keys
            if isinstance(data, dict):
                products_list = []
                for product_name, product_data in data.items():
                    if isinstance(product_data, dict) and "name" in product_data:
                        products_list.append(product_data)
                    elif not isinstance(product_data, dict) and product_name == "error":
                        logger.warning(f"API error response: {product_data}")
                        continue
                    else:
                        logger.warning(f"Unexpected product data format: {type(product_data)}")
                return products_list
                
            logger.warning(f"Unexpected response structure from API: {type(data)}")
            return []
        except Exception as e:
            logger.error(f"Error extracting products: {e}")
            return []
    
    def _retry_request(self, request_func, name, validators=None, max_retries=3):
        """
        Helper to retry API requests with exponential backoff
        
        Args:
            request_func: Function that makes the HTTP request
            name: Name of the request for logging
            validators: List of validation functions to apply
            max_retries: Maximum number of retry attempts
            
        Returns:
            Response object or None if all retries failed
        """
        retry_delay = 1
        
        for attempt in range(max_retries):
            try:
                response = request_func()
                
                if 500 <= response.status_code < 600:
                    if attempt < max_retries - 1:
                        logger.warning(f"{name} failed with status {response.status_code}, retrying in {retry_delay}s ({attempt+1}/{max_retries})")
                        time.sleep(retry_delay)
                        retry_delay *= 2
                        continue
                
                if validators:
                    valid = validate_response(response, validators)
                    if not valid and attempt < max_retries - 1:
                        logger.warning(f"{name} validation failed, retrying in {retry_delay}s ({attempt+1}/{max_retries})")
                        time.sleep(retry_delay)
                        retry_delay *= 2
                        continue
                
                return response
            
            except Exception as e:
                if attempt < max_retries - 1:
                    logger.error(f"Error during {name} (attempt {attempt+1}): {str(e)}")
                    time.sleep(retry_delay)
                    retry_delay *= 2
                else:
                    logger.error(f"Final error during {name} after {max_retries} attempts: {str(e)}")
        
        return None
    
    def process_products_response(self, response, update_shared_data=True):
        """
        Process product response and optionally update shared data
        
        Args:
            response: HTTP response object
            update_shared_data: Whether to update shared data with products
            
        Returns:
            List of extracted products or None
        """
        if response and response.status_code == 200:
            products = self._extract_products(response)
            if products and update_shared_data:
                self.shared_data.update_products(products)
            return products
        return None 