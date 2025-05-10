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
from src.base_user import BaseAPIUser

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[logging.FileHandler("locust.log"), logging.StreamHandler()]
)
logger = logging.getLogger("locustfile")
shared_data = SharedData()


class SimulationUser(BaseAPIUser):
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {
            "health_check": 100 # Default, can be overridden by command line
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
    
    def _load_initial_products(self):
        max_retries = 3
        retry_delay = 2
        
        for attempt in range(max_retries):
            try:
                with self.client.get(self._get_path("/products"), name=f"Initial_Products_Load (Attempt {attempt+1})", catch_response=True) as response:
                    logger.info(f"Initial product load response: {response.status_code}")
                    logger.debug(f"Initial product load response text: {response.text}") # Use debug for potentially long text
                    if response.status_code == 200:
                        products = self._extract_products(response)
                        if products:
                            self.shared_data.update_products(products)
                            logger.info(f"Loaded initial product data: {len(products)} products")
                            return True
                        else:
                            logger.warning("No products found in initial data load despite 200 OK")
                    else:
                        logger.warning(f"Failed to load initial product data: {response.status_code} (Attempt {attempt+1}/{max_retries})")
                
                if attempt < max_retries - 1:
                    logger.info(f"Retrying in {retry_delay} seconds...")
                    time.sleep(retry_delay)
                    retry_delay *= 1.5
            
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
    
    @task(70)
    @tag("browse")
    def browse_all_products(self):
        task_name = "browse_all_products"
        if not self._can_execute_task(task_name):
            return
        
        current_count = self._increment_task_count(task_name)
        request_name = f"Get_All_Products (#{current_count})"
        
        def request_func():
            with self.client.get(self._get_path("/products"), name=request_name, catch_response=True) as response:
                return response
        
        response = self._retry_request(
            request_func, 
            request_name,
            validators=[check_status_code(200), check_content_type(), check_products_list_schema]
        )
        self.process_products_response(response, update_shared_data=True)
    
    @task(50)
    @tag("browse", "category")
    def get_products_by_category(self):
        task_name = "get_products_by_category"
        if not self._can_execute_task(task_name):
            return
        
        current_count = self._increment_task_count(task_name)
        possible_categories = self.possible_categories if self.possible_categories else ["Electronics"]
        category = random.choice(possible_categories)
        request_name = f"Get_Products_By_Category_{category} (#{current_count})"
        
        def request_func():
            with self.client.get(self._get_path("/products/category"), params={"category": category}, 
                               name=request_name, catch_response=True) as response:
                return response
        
        response = self._retry_request(
            request_func,
            request_name,
            validators=[check_status_code(200), check_content_type(), check_products_list_schema]
        )
        self.process_products_response(response, update_shared_data=False) # Don't necessarily update all products from category view

    @task(40)
    @tag("shopping", "details")
    def get_product_by_name(self):
        task_name = "get_product_by_name"
        if not self._can_execute_task(task_name):
            return
            
        product = self.shared_data.get_random_product()
        if not product or not product.get("name"):
            # logger.debug("No product found or product has no name for get_product_by_name")
            return
        product_name = product.get("name")
        current_count = self._increment_task_count(task_name)
        request_name = f"Get_Product_By_Name_{product_name[:20]} (#{current_count})" # Truncate for readability
        
        def request_func():
            with self.client.post(self._get_path("/products/details"), json={"name": product_name}, 
                                name=request_name, catch_response=True) as response:
                return response
        
        self._retry_request(
            request_func,
            request_name,
            validators=[check_status_code(200), check_content_type()]
        )

    @task(5)
    @tag("admin", "inventory")
    def update_product_stock(self):
        task_name = "update_product_stock"
        if not self._can_execute_task(task_name):
            return

        product = self.shared_data.get_random_product()
        if not product or not product.get("name"):
            # logger.debug("No product found or product has no name for update_product_stock")
            return
        product_name = product.get("name")
        new_stock = random.randint(0, 500) # Stock can be 0
        current_count = self._increment_task_count(task_name)
        request_name = f"Update_Product_Stock_{product_name[:20]} (#{current_count})"
        
        def request_func():
            with self.client.patch(self._get_path("/products/stock"), json={"name": product_name, "stock": new_stock}, 
                                 name=request_name, catch_response=True) as response:
                return response
        
        self._retry_request(
            request_func,
            request_name,
            validators=[check_status_code(200), check_content_type()]
        )

    @task(15)
    @tag("shopping", "purchase")
    def buy_product(self):
        task_name = "buy_product"
        if not self._can_execute_task(task_name):
            return

        product = self.shared_data.get_random_product()
        if not product or not product.get("name"):
            # logger.debug("No product found or product has no name for buy_product")
            return
        product_name = product.get("name")
        quantity = random.randint(1, 5)
        current_count = self._increment_task_count(task_name)
        request_name = f"Buy_Product_{product_name[:20]} (#{current_count})"
        
        def request_func():
            with self.client.post(self._get_path("/products/buy"), json={"name": product_name, "quantity": quantity}, 
                               name=request_name, catch_response=True) as response:
                return response
        
        # Allow 409 (out of stock) as a valid response for buy attempts
        response = self._retry_request(
            request_func,
            request_name,
            validators=[check_status_code([200, 409]), check_content_type()] 
        )
        # Custom logic for buy_product response can be added here if needed

    @task(2)
    @tag("health")
    def health_check(self):
        task_name = "health_check"
        if not self._can_execute_task(task_name):
            return
        
        current_count = self._increment_task_count(task_name)
        request_name = f"Health_Check (#{current_count})"
        
        def request_func():
            with self.client.get(self._get_path("/health"), name=request_name, catch_response=True) as response:
                return response
        
        response = self._retry_request(
            request_func,
            request_name,
            validators=[check_status_code(200)]
        )
        
        if response and response.status_code != 200:
            logger.error(f"Health check failed: {response.status_code} - {response.text}")

@events.init_command_line_parser.add_listener
def _(parser):
    # Task execution limits
    parser.add_argument("--max-health-check", type=int, env_var="LOCUST_MAX_HEALTH_CHECK", default=-1, help="Max executions for health_check (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-browse-all-products", type=int, env_var="LOCUST_MAX_BROWSE_ALL_PRODUCTS", default=-1, help="Max executions for browse_all_products (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-get-products-by-category", type=int, env_var="LOCUST_MAX_GET_PRODUCTS_BY_CATEGORY", default=-1, help="Max executions for get_products_by_category (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-get-product-by-name", type=int, env_var="LOCUST_MAX_GET_PRODUCT_BY_NAME", default=-1, help="Max executions for get_product_by_name (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-update-product-stock", type=int, env_var="LOCUST_MAX_UPDATE_PRODUCT_STOCK", default=-1, help="Max executions for update_product_stock (-1=unlimited, 0=disabled)")
    parser.add_argument("--max-buy-product", type=int, env_var="LOCUST_MAX_BUY_PRODUCT", default=-1, help="Max executions for buy_product (-1=unlimited, 0=disabled)")
    
    # Nginx proxy setting
    parser.add_argument("--use-nginx-proxy", type=str, env_var="USE_NGINX_PROXY", default="false", help="Set to 'true' to route requests via Nginx proxy (e.g., /product-service/endpoint)")

# Ensure telemetry is set up if enabled
from src.telemetry.monitoring import setup_opentelemetry
setup_opentelemetry()

# Ensure custom statistics event handlers are registered
from src.telemetry.event_hooks import register_stats_event_handlers
register_stats_event_handlers()
