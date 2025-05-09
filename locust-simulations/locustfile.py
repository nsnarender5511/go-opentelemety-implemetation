"""
Main Locust entry point.
This file defines a simplified SimulationUser class that only performs health checks.
"""
import os
import logging
import time
from typing import Dict, Any, Optional

from locust import HttpUser, task, between, tag, events
from locust.clients import ResponseContextManager
import json
import random
import time
from typing import List

# Assuming these are still relevant and in the correct path after consolidation
from src.utils.shared_data import SharedData
from src.utils.http_validation import validate_response, check_status_code, check_content_type, check_products_list_schema

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[logging.FileHandler("locust.log"), logging.StreamHandler()]
)
logger = logging.getLogger("locustfile")
shared_data = SharedData()


class SimulationUser(HttpUser):
    """
    Simplified user class that only performs health checks.
    """
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {
            "health_check": 100
        }
        self.possible_categories = []
    
    def on_start(self):
        self.shared_data = shared_data # Use the global instance
        if not self.shared_data.get_products():
            self._load_initial_products()
        self._initialize_test_data_params()
    
    def _initialize_test_data_params(self):
        # Extract categories from product data
        products = self.shared_data.get_products()
        if products:
            # Extract unique categories from product data
            self.possible_categories = list(set(product.get("category", "") for product in products if product.get("category")))
            logger.info(f"Extracted categories from product data: {self.possible_categories}")
        
        if not self.possible_categories:
            # Fallback to defaults if no categories were found
            self.possible_categories = ["Electronics", "Apparel", "Books", "Kitchenware", "Furniture"]
            logger.info(f"Using default categories: {self.possible_categories}")
    
    def _can_execute_task(self, task_name):
        """Return True if the task can be executed, False otherwise"""
        # Parse max executions from command-line args
        max_executions_arg = f"max_{task_name}"
        max_executions = getattr(self.environment.parsed_options, max_executions_arg, -1) if hasattr(self.environment, "parsed_options") else -1

        if task_name == "health_check":
            return True
        
        # Disabled if max executions is 0
        if max_executions == 0:
            return False
        
        # No limit if max executions is negative
        if max_executions < 0:
            return True
            
        # Check if we've reached the limit
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
                with self.client.get("/products", name=f"Initial_Products_Load (Attempt {attempt+1})", catch_response=True) as response:
                    logger.info(f"Initial product load response: {response.status_code}")
                    logger.info(f"Initial product load response: {response.text}")
                    if response.status_code == 200:
                        products = self._extract_products(response)
                        if products:
                            self.shared_data.update_products(products)
                            logger.info(f"Loaded initial product data: {len(products)} products")
                            return True
                        else:
                            logger.warning("No products found in initial data load")
                    else:
                        logger.warning(f"Failed to load initial product data: {response.status_code} (Attempt {attempt+1}/{max_retries})")
                
                if attempt < max_retries - 1:  # Don't sleep after the last attempt
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                    retry_delay *= 1.5  # Increase delay with each retry
            
            except Exception as e:
                logger.error(f"Error during initial product load attempt {attempt+1}: {str(e)}")
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                    retry_delay *= 1.5
        
        logger.error(f"Failed to load initial product data after {max_retries} attempts")
        return False
    
    def _extract_products(self, response: ResponseContextManager) -> List[Dict[str, Any]]:
        try:
            data = response.json()
            
            # Case 1: Object where each key is a product name
            if isinstance(data, dict) and not "data" in data:
                # Convert object with product names as keys to a list of products
                products_list = []
                for product_name, product_data in data.items():
                    if isinstance(product_data, dict) and "name" in product_data:
                        products_list.append(product_data)
                    elif not isinstance(product_data, dict) and product_name == "error":
                        # Skip error messages
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
                products_list = data
                return products_list
                
            else:
                logger.warning(f"Unexpected response structure from API: {type(data)}")
                return []
        except Exception as e:
            logger.error(f"Error extracting products: {e}")
            return []
    
    def _retry_request(self, request_func, name, validators=None, max_retries=3):
        """
        Helper to retry API requests with exponential backoff
        
        Args:
            request_func: Function that performs the actual request (should return response object)
            name: Name to use for the request (for reporting)
            validators: List of validator functions to run on the response
            max_retries: Maximum number of retry attempts
            
        Returns:
            Response object if successful, None otherwise
        """
        retry_delay = 1
        
        for attempt in range(max_retries):
            try:
                response = request_func()
                
                # If status code is 5xx (server error), retry
                if 500 <= response.status_code < 600:
                    if attempt < max_retries - 1:
                        logger.warning(f"{name} failed with status {response.status_code}, retrying in {retry_delay}s ({attempt+1}/{max_retries})")
                        time.sleep(retry_delay)
                        retry_delay *= 2  # Exponential backoff
                        continue
                
                # Validate response if validators are provided
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
    
    @task(70)
    @tag("browse")
    def browse_all_products(self):
        task_name = "browse_all_products"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        
        def request_func():
            with self.client.get("/products", name=f"Get_All_Products (#{count})", catch_response=True) as response:
                return response
        
        response = self._retry_request(
            request_func, 
            f"Get_All_Products (#{count})",
            validators=[check_status_code(200), check_content_type(), check_products_list_schema]
        )
        
        if response and response.status_code == 200:
            products = self._extract_products(response)
            if products:
                self.shared_data.update_products(products)
    
    @task(50)
    @tag("browse", "category")
    def get_products_by_category(self):
        task_name = "get_products_by_category"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        possible_categories = self.possible_categories if self.possible_categories else ["Electronics"]
        category = random.choice(possible_categories)
        
        def request_func():
            with self.client.get(f"/products/category", params={"category": category}, 
                               name=f"Get_Products_By_Category (#{count})", catch_response=True) as response:
                if response.status_code == 200 and len(category) <= 20:
                    response.name = f"Get_Products_By_Category_{category} (#{count})"
                return response
        
        response = self._retry_request(
            request_func,
            f"Get_Products_By_Category (#{count})",
            validators=[check_status_code(200), check_content_type()]
        )
        
        if response and response.status_code == 200:
            self._extract_products(response)

    @task(40)
    @tag("shopping", "details")
    def get_product_by_name(self):
        task_name = "get_product_by_name"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        product = self.shared_data.get_random_product()
        if not product or not product.get("name"):
            return
        product_name = product.get("name")
        
        def request_func():
            with self.client.post("/products/details", json={"name": product_name}, 
                                name=f"Get_Product_By_Name (#{count})", catch_response=True) as response:
                return response
        
        self._retry_request(
            request_func,
            f"Get_Product_By_Name (#{count})",
            validators=[check_status_code(200), check_content_type()]
        )

    @task(5)
    @tag("admin", "inventory")
    def update_product_stock(self):
        task_name = "update_product_stock"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        product = self.shared_data.get_random_product()
        if not product or not product.get("name"):
            return
        product_name = product.get("name")
        new_stock = random.randint(10, 500)
        
        def request_func():
            with self.client.patch("/products/stock", json={"name": product_name, "stock": new_stock}, 
                                 name=f"Update_Product_Stock (#{count})", catch_response=True) as response:
                return response
        
        self._retry_request(
            request_func,
            f"Update_Product_Stock (#{count})",
            validators=[check_status_code(200), check_content_type()]
        )

    @task(15)
    @tag("shopping", "purchase")
    def buy_product(self):
        task_name = "buy_product"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        product = self.shared_data.get_random_product()
        if not product or not product.get("name"):
            return
        product_name = product.get("name")
        quantity = random.randint(1, 5)
        
        def request_func():
            with self.client.post("/products/buy", json={"name": product_name, "quantity": quantity}, 
                               name=f"Buy_Product (#{count})", catch_response=True) as response:
                return response
        
        self._retry_request(
            request_func,
            f"Buy_Product (#{count})",
            validators=[check_status_code(200), check_content_type()]
        )

    @task(2)
    @tag("health")
    def health_check(self):
        task_name = "health_check"
        count = self._increment_task_count(task_name)
        
        def request_func():
            with self.client.get("/health", name=f"Health_Check (#{count})") as response:
                return response
        
        self._retry_request(
            request_func,
            f"Health_Check (#{count})"
        )

# Import load shape if specified (optional)
load_shape = None
load_shape_name = os.environ.get("LOAD_SHAPE", "").lower()

# Assuming these paths are correct relative to the new locustfile.py location or they are in PYTHONPATH
if load_shape_name == "stages":
    from src.load_shapes.stages_shape import StagesLoadShape
    load_shape = StagesLoadShape()
    logger.info("Using stages load shape")
elif load_shape_name == "spike":
    from src.load_shapes.spike_shape import SpikeLoadShape
    load_shape = SpikeLoadShape()
    logger.info("Using spike load shape")
elif load_shape_name == "multiple_spikes":
    from src.load_shapes.spike_shape import MultipleSpikeLoadShape
    load_shape = MultipleSpikeLoadShape()
    logger.info("Using multiple spikes load shape")
elif load_shape_name == "ramping":
    from src.load_shapes.stages_shape import RampingLoadShape
    load_shape = RampingLoadShape()
    logger.info("Using ramping load shape")
elif load_shape_name:
    logger.warning(f"Unknown load shape: {load_shape_name}")

# Set up telemetry
from src.telemetry.monitoring import setup_opentelemetry
from src.telemetry.event_hooks import register_stats_event_handlers

setup_opentelemetry()
register_stats_event_handlers()

@events.init_command_line_parser.add_listener
def _(parser):
    parser.add_argument("--max-browse-all-products", type=int, env_var="LOCUST_MAX_BROWSE_ALL_PRODUCTS", default=0, help="Max executions for browse_all_products (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-get-products-by-category", type=int, env_var="LOCUST_MAX_GET_PRODUCTS_BY_CATEGORY", default=0, help="Max executions for get_products_by_category (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-get-product-by-name", type=int, env_var="LOCUST_MAX_GET_PRODUCT_BY_NAME", default=0, help="Max executions for get_product_by_name (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-update-product-stock", type=int, env_var="LOCUST_MAX_UPDATE_PRODUCT_STOCK", default=0, help="Max executions for update_product_stock (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-buy-product", type=int, env_var="LOCUST_MAX_BUY_PRODUCT", default=0, help="Max executions for buy_product (-1=unlimited, 0=disabled)")

@events.init.add_listener
def on_locust_init(environment, **kwargs):
    logger.info("Locust initialization complete")
    logger.info("Using SimulationUser with health check at constant weight 100")
    if load_shape:
        logger.info(f"Using load shape: {type(load_shape).__name__}")
    else:
        logger.info("Using standard Locust load control (no custom shape)") 