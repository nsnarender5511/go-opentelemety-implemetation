"""
Master Store User for Load Testing
This file defines the MasterStoreUser class for simulating load on master-store endpoints.
"""
import os
import logging
import random
import time
from typing import Dict, Any, List, Optional

from locust import HttpUser, task, between, tag, events
from locust.clients import ResponseContextManager
import json

from src.utils.shared_data import SharedData
from src.utils.http_validation import validate_response, check_status_code, check_content_type
from src.utils.master_store_validation import check_master_store_products_schema

# Configure logging
logger = logging.getLogger("master_store_user")

class MasterStoreUser(HttpUser):
    """
    User class for testing master-store endpoints.
    """
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {
            "master_health_check": 0,
            "master_browse_products": 0,
            "master_buy_product": 0
        }
        self.use_nginx_proxy = os.environ.get("USE_NGINX_PROXY", "false").lower() == "true"
        
    def on_start(self):
        self.shared_data = SharedData()  # Use global instance
        if not self.shared_data.get_products():
            self._load_initial_products()
    
    def _get_path(self, endpoint):
        """Convert endpoint path based on whether nginx proxy is used"""
        if self.use_nginx_proxy:
            # For nginx proxy, prepend /master/
            return f"/master{endpoint}"
        # Direct access
        return endpoint
    
    def _can_execute_task(self, task_name):
        """Return True if the task can be executed, False otherwise"""
        max_executions_arg = f"max_{task_name}"
        max_executions = getattr(self.environment.parsed_options, max_executions_arg, -1) if hasattr(self.environment, "parsed_options") else -1
        
        if max_executions == 0:
            return False
        if max_executions < 0:
            return True
        current_count = self.task_counts.get(task_name, 0)
        return current_count < max_executions
    
    def _increment_task_count(self, task_name):
        """Increment the task count and return the new value"""
        self.task_counts[task_name] = self.task_counts.get(task_name, 0) + 1
        return self.task_counts[task_name]
    
    def _load_initial_products(self):
        """Load initial product data if needed with retry logic"""
        max_retries = 3
        retry_delay = 2
        
        for attempt in range(max_retries):
            try:
                endpoint = self._get_path("/products")
                with self.client.get(endpoint, name=f"Master_Initial_Products_Load (Attempt {attempt+1})", catch_response=True) as response:
                    logger.info(f"Master initial product load response: {response.status_code}")
                    if response.status_code == 200:
                        products = self._extract_products(response)
                        if products:
                            self.shared_data.update_products(products)
                            logger.info(f"Loaded initial master product data: {len(products)} products")
                            return True
                        else:
                            logger.warning("No products found in initial master data load")
                    else:
                        logger.warning(f"Failed to load initial master product data: {response.status_code} (Attempt {attempt+1}/{max_retries})")
                
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                    retry_delay *= 1.5
            
            except Exception as e:
                logger.error(f"Error during initial master product load attempt {attempt+1}: {str(e)}")
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                    retry_delay *= 1.5
        
        logger.error(f"Failed to load initial master product data after {max_retries} attempts")
        return False
    
    def _extract_products(self, response: ResponseContextManager) -> List[Dict[str, Any]]:
        try:
            data = response.json()
            
            # Case 1: Object where each key is a product name (master-store format)
            if isinstance(data, dict) and not "data" in data:
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
                
            # Case 2: Response has a "data" field containing the products
            elif isinstance(data, dict) and "data" in data:
                products_list = data["data"]
                # If data is an object with product names as keys, convert to list
                if isinstance(products_list, dict):
                    return [product for _, product in products_list.items() if isinstance(product, dict)]
                return products_list
                
            # Case 3: Response is a direct list of products
            elif isinstance(data, list):
                return data
                
            else:
                logger.warning(f"Unexpected response structure from API: {type(data)}")
                return []
        except Exception as e:
            logger.error(f"Error extracting products: {e}")
            return []
    
    def _retry_request(self, request_func, name, validators=None, max_retries=3):
        """Helper to retry API requests with exponential backoff"""
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
    
    @task(10)
    @tag("master", "health")
    def master_health_check(self):
        task_name = "master_health_check"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        
        def request_func():
            endpoint = self._get_path("/health")
            with self.client.get(endpoint, name=f"Master_Health_Check (#{count})", catch_response=True) as response:
                return response
        
        response = self._retry_request(
            request_func,
            f"Master_Health_Check (#{count})",
            validators=[check_status_code(200), check_content_type()]
        )
    
    @task(70)
    @tag("master", "browse")
    def master_browse_products(self):
        task_name = "master_browse_products"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        
        def request_func():
            endpoint = self._get_path("/products")
            with self.client.get(endpoint, name=f"Master_Get_All_Products (#{count})", catch_response=True) as response:
                return response
        
        response = self._retry_request(
            request_func,
            f"Master_Get_All_Products (#{count})",
            validators=[check_status_code(200), check_content_type(), check_master_store_products_schema]
        )
        
        if response and response.status_code == 200:
            products = self._extract_products(response)
            if products:
                self.shared_data.update_products(products)
    
    @task(20)
    @tag("master", "purchase")
    def master_buy_product(self):
        task_name = "master_buy_product"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        
        # Get a random product
        product = self.shared_data.get_random_product()
        if not product:
            logger.warning("No products available for master buy operation")
            return
        
        product_name = product.get("name", "Unknown Product")
        # Random quantity between 1 and 5
        quantity = random.randint(1, 5)
        
        def request_func():
            endpoint = self._get_path("/products/update-stock")
            with self.client.post(
                endpoint,
                json={"name": product_name, "quantity": quantity},
                name=f"Master_Buy_Product (#{count})",
                catch_response=True
            ) as response:
                return response
        
        response = self._retry_request(
            request_func,
            f"Master_Buy_Product (#{count})",
            validators=[check_status_code(200), check_content_type()]
        )
        
        if response and response.status_code == 200:
            try:
                result = response.json()
                logger.info(f"Master purchase successful: {product_name}, quantity: {quantity}, remaining: {result.get('data', {}).get('remainingStock', 'unknown')}")
            except Exception as e:
                logger.error(f"Error parsing master purchase response: {e}") 