"""
Master Store User for Load Testing
This file defines the MasterStoreUser class for simulating load on master-store endpoints.
"""
import os
import logging
import random
import time
from typing import Dict, Any, List, Optional

from locust import task, tag, events
from locust.clients import ResponseContextManager
import json

from src.base_user import BaseAPIUser
from src.utils.http_validation import validate_response, check_status_code, check_content_type
from src.utils.product_validation import check_product_schema

# Configure logging
logger = logging.getLogger("master_store_user")

class MasterStoreUser(BaseAPIUser):
    """
    User class for testing master-store endpoints.
    """
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {
            "master_health_check": 0,
            "master_browse_products": 0,
            "master_buy_product": 0
        }
        self.service_prefix = "master"  # Set service prefix for path construction
        self.service_name = "Master Store"
        
    def on_start(self):
        super().on_start()  # Call parent on_start to initialize shared data
        if not self.shared_data.get_products():
            self._load_initial_products()
    
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
            validators=[check_status_code(200), check_content_type(), 
                      lambda r: check_product_schema(r, expected_format="array")]
        )
        
        self.process_products_response(response)
    
    @task(20)
    @tag("master", "purchase")
    def master_buy_product(self):
        task_name = "master_buy_product"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        
        # Get a random product from shared data
        products = self.shared_data.get_products()
        if not products:
            logger.warning("No products available for purchase test")
            return
            
        product = random.choice(products)
        quantity = random.randint(1, 5)
        
        def request_func():
            endpoint = self._get_path("/products/update-stock")
            payload = {
                "name": product["name"],
                "quantity": quantity
            }
            
            with self.client.post(
                endpoint, 
                json=payload,
                name=f"Master_Buy_Product (#{count})",
                catch_response=True
            ) as response:
                return response
        
        response = self._retry_request(
            request_func,
            f"Master_Buy_Product (#{count})",
            validators=[check_status_code(200), check_content_type()]
        ) 