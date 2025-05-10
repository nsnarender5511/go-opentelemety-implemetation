import os
import logging
import time
from typing import Dict, Any, Optional

from locust import HttpUser, task, between, tag, events
from locust.clients import ResponseContextManager
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
    
    DEFAULT_TASK_WEIGHTS = {
        "browse_all_products": 15,
        "get_products_by_category": 15,
        "get_product_by_name": 20,
        "update_product_stock": 5,
        "buy_product": 20,
        "health_check": 25,
    }
    
    @staticmethod
    def configure_parser(parser):
        # Add arguments for task weights and max_executions
        for task_name, default_weight in SimulationUser.DEFAULT_TASK_WEIGHTS.items():
            # Weight argument
            parser.add_argument(
                f"--weight-{task_name.replace('_', '-')}",
                type=int,
                env_var=f"LOCUST_WEIGHT_{task_name.upper()}",
                default=None,  # Default to None to detect if user set it
                help=f"Weight for {task_name} task (default: {default_weight})"
            )
            # Max execution argument
            parser.add_argument(
                f"--max-{task_name.replace('_', '-')}",
                type=int,
                env_var=f"LOCUST_MAX_{task_name.upper()}",
                default=-1,
                help=f"Max executions for {task_name} task (-1=unlimited, 0=disabled)"
            )

        # Nginx proxy setting
        parser.add_argument(
            "--use-nginx-proxy", 
            type=str, 
            env_var="USE_NGINX_PROXY", 
            default="false", 
            help="Set to 'true' to route requests via Nginx proxy (e.g., /product-service/endpoint)"
        )
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {
            "health_check": 100 # Default, can be overridden by command line
        }
        self.possible_categories = []

        # Dynamically build self.tasks
        weighted_tasks_list = []
        for task_name, default_weight in self.DEFAULT_TASK_WEIGHTS.items():
            arg_name = f"weight_{task_name}" 
            
            user_weight_input_str = None # Stores the raw string from env/arg
            if hasattr(self.environment, "parsed_options") and self.environment.parsed_options is not None:
                # getattr might return '' if env var is set but empty, or None if not set
                raw_user_weight = getattr(self.environment.parsed_options, arg_name, None)
                if raw_user_weight is not None: # Ensure it's not None before checking if it's an empty string
                    user_weight_input_str = str(raw_user_weight) # Ensure it's a string for consistent handling
            
            actual_weight = default_weight
            if user_weight_input_str is not None: # Check if user provided any input
                if user_weight_input_str == '': # Specifically handle empty string case
                    logger.warning(f"User-defined weight for task '{task_name}' is an empty string. Using default weight {default_weight}.")
                    # actual_weight remains default_weight
                else:
                    try:
                        user_weight_int = int(user_weight_input_str)
                        if user_weight_int >= 0: # Weights must be non-negative
                            actual_weight = user_weight_int
                        else:
                            logger.warning(f"User-defined weight '{user_weight_input_str}' for task '{task_name}' is negative. Using default weight {default_weight}.")
                            # actual_weight remains default_weight
                    except ValueError:
                        logger.warning(f"Could not convert user-defined weight '{user_weight_input_str}' for task '{task_name}' to int. Using default weight {default_weight}.")
                        # actual_weight remains default_weight
            
            if actual_weight > 0:
                task_method = getattr(type(self), task_name, None)
                if task_method:
                    for _ in range(actual_weight): # Append task_method actual_weight times
                        weighted_tasks_list.append(task_method)
                else:
                    logger.warning(f"Task method {task_name} not found in SimulationUser for dynamic weighting.")
        
        self.tasks = weighted_tasks_list # Assign the constructed list to self.tasks
        
        if not self.tasks:
            logger.error("No tasks were assigned weights > 0 or no task methods found. SimulationUser will have no tasks to run!")
    
    def on_start(self):
        super().on_start()
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
    SimulationUser.configure_parser(parser)
    # Any other truly global (non-SimulationUser specific) arguments 
    # could remain here or be added by other listeners.

# Ensure telemetry is set up if enabled
from src.telemetry.monitoring import setup_opentelemetry
setup_opentelemetry()

# Ensure custom statistics event handlers are registered
from src.telemetry.event_hooks import register_stats_event_handlers
register_stats_event_handlers()
