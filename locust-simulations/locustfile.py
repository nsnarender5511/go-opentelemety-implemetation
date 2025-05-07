"""
Main Locust entry point.
This file defines the SimulationUser class, load shapes, telemetry components,
and custom command-line arguments for task execution limits.
"""
import os
import logging
import random
from typing import Dict, Any, List, Optional

from locust import HttpUser, task, between, tag, events
from locust.clients import ResponseContextManager

# Assuming these are still relevant and in the correct path after consolidation
from src.utils.shared_data import SharedData
from src.utils.validators import validate_response, check_status_code, check_content_type, check_products_list_schema

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("locustfile") # Changed logger name for clarity

# Global shared data instance for all users
# This needs to be initialized before SimulationUser class definition if it's used by it at class level
# or ensure it's initialized in on_start if only used by instances.
# For simplicity, keeping it global as it was in simulation_user.py
shared_data = SharedData()

class SimulationUser(HttpUser):
    """
    Unified user class with all possible tasks and configurable execution limits via command-line arguments.
    
    Run with the --class-picker flag. Execution limits are set via --max-<task_name> arguments.
    Task selection is still random, but each task will only execute up to its configured limit.
    """
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {
            "browse_all_products": 0,
            "get_products_by_category": 0,
            "search_products": 0,
            "view_product_details": 0,
            "add_to_cart": 0,
            "checkout": 0,
            "update_inventory": 0,
            "view_analytics": 0,
            "health_check": 0
        }
    
    def on_start(self):
        self.shared_data = shared_data # Use the global instance
        if not self.shared_data.get_products():
            self._load_initial_products()
        self._initialize_test_data_params()
    
    def _initialize_test_data_params(self):
        self.possible_categories = getattr(self.environment.parsed_options, "possible_categories_list", 
                                           ["Electronics", "Books", "Clothing"])
        self.search_terms = getattr(self.environment.parsed_options, "search_terms_list", 
                                    ["phone", "laptop", "shirt", "book"])
        if isinstance(self.possible_categories, str):
            self.possible_categories = [cat.strip() for cat in self.possible_categories.split(',') if cat.strip()]
        if isinstance(self.search_terms, str):
            self.search_terms = [term.strip() for term in self.search_terms.split(',') if term.strip()]

    def _can_execute_task(self, task_name):
        if task_name not in self.task_counts:
            return True
        max_count_attr_name = f"max_{task_name}"
        max_count = getattr(self.environment.parsed_options, max_count_attr_name, -1)
        if max_count < 0:
            return True
        return self.task_counts[task_name] < max_count
    
    def _increment_task_count(self, task_name):
        if task_name in self.task_counts:
            self.task_counts[task_name] += 1
            return self.task_counts[task_name]
        return 0
    
    def _load_initial_products(self):
        with self.client.get("/products", name="Initial_Product_Load") as response:
            if response.status_code == 200:
                products = self._extract_products(response)
                if products:
                    self.shared_data.update_products(products)
                    logger.info(f"Loaded initial product data: {len(products)} products")
                else:
                    logger.warning("No products found in initial data load")
            else:
                logger.error(f"Failed to load initial product data: {response.status_code}")
    
    def _extract_products(self, response: ResponseContextManager) -> List[Dict[str, Any]]:
        try:
            data = response.json()
            if isinstance(data, dict) and "data" in data:
                products_list = data["data"]
            elif isinstance(data, list):
                products_list = data
            else:
                logger.warning(f"Unexpected response structure from API: {type(data)}")
                return []
            return [
                item for item in products_list 
                if isinstance(item, dict) and "productID" in item and "name" in item
            ]
        except Exception as e:
            logger.error(f"Error extracting products: {e}")
            return []
    
    @task(70)
    @tag("browse")
    def browse_all_products(self):
        task_name = "browse_all_products"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        with self.client.get("/products", name=f"Get_All_Products (#{count})", catch_response=True) as response:
            valid = validate_response(response, [check_status_code(200), check_content_type(), check_products_list_schema])
            if valid:
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
        with self.client.get(f"/products/category", params={"category": category}, name=f"Get_Products_By_Category (#{count})", catch_response=True) as response:
            valid = validate_response(response, [check_status_code(200), check_content_type()])
            if valid:
                self._extract_products(response)
                if len(category) <= 20: response.name = f"Get_Products_By_Category_{category} (#{count})"

    @task(30)
    @tag("browse", "search")
    def search_products(self):
        task_name = "search_products"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        search_terms = self.search_terms if self.search_terms else ["phone"]
        search_term = random.choice(search_terms)
        with self.client.get(f"/products/search", params={"q": search_term}, name=f"Search_Products (#{count})", catch_response=True) as response:
            valid = validate_response(response, [check_status_code(200), check_content_type()])
            if valid:
                self._extract_products(response)
                if len(search_term) <= 20: response.name = f"Search_Products_{search_term} (#{count})"

    @task(40)
    @tag("shopping", "details")
    def view_product_details(self):
        task_name = "view_product_details"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        product = self.shared_data.get_random_product()
        if not product or not product.get("productID"):
            return
        product_id = product.get("productID")
        with self.client.get(f"/products/{product_id}", name=f"Get_Product_Details (#{count})", catch_response=True) as response:
            validate_response(response, [check_status_code(200), check_content_type()])

    @task(25)
    @tag("shopping", "cart")
    def add_to_cart(self):
        task_name = "add_to_cart"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        product = self.shared_data.get_random_product()
        if not product or not product.get("productID"):
            return
        product_id = product.get("productID")
        quantity = random.randint(1, 5)
        with self.client.post(f"/cart/add", json={"product_id": product_id, "quantity": quantity}, name=f"Add_To_Cart (#{count})", catch_response=True) as response:
            validate_response(response, [check_status_code(200), check_content_type()])

    @task(15)
    @tag("shopping", "checkout")
    def checkout(self):
        task_name = "checkout"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        with self.client.post(f"/checkout", json={"payment_method": "credit_card", "shipping_address": "123 Test St, City, Country"}, name=f"Checkout (#{count})", catch_response=True) as response:
            validate_response(response, [check_status_code(200), check_content_type()])

    @task(10)
    @tag("admin", "inventory")
    def update_inventory(self):
        task_name = "update_inventory"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        product = self.shared_data.get_random_product()
        if not product or not product.get("productID"):
            return
        product_id = product.get("productID")
        new_stock = random.randint(10, 100)
        with self.client.put(f"/admin/products/{product_id}/stock", json={"stock": new_stock}, name=f"Update_Inventory (#{count})", catch_response=True) as response:
            validate_response(response, [check_status_code(200), check_content_type()])

    @task(5)
    @tag("admin", "analytics")
    def view_analytics(self):
        task_name = "view_analytics"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        with self.client.get(f"/admin/analytics", name=f"View_Analytics (#{count})", catch_response=True) as response:
            validate_response(response, [check_status_code(200), check_content_type()])

    @task(2)
    @tag("health")
    def health_check(self):
        task_name = "health_check"
        if not self._can_execute_task(task_name):
            return
        count = self._increment_task_count(task_name)
        with self.client.get("/health", name=f"Health_Check (#{count})") as response:
            pass

# Import web UI extension (ensure this path is correct if locustfile moves)
from src.web_extension import init_web_ui_extension

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
    parser.add_argument("--max-browse-all-products", type=int, env_var="LOCUST_MAX_BROWSE_ALL_PRODUCTS", default=0, help="Max executions for browse_all_products")
    parser.add_argument("--max-get-products-by-category", type=int, env_var="LOCUST_MAX_GET_PRODUCTS_BY_CATEGORY", default=0, help="Max executions for get_products_by_category")
    parser.add_argument("--max-search-products", type=int, env_var="LOCUST_MAX_SEARCH_PRODUCTS", default=0, help="Max executions for search_products")
    parser.add_argument("--max-view-product-details", type=int, env_var="LOCUST_MAX_VIEW_PRODUCT_DETAILS", default=0, help="Max executions for view_product_details")
    parser.add_argument("--max-add-to-cart", type=int, env_var="LOCUST_MAX_ADD_TO_CART", default=0, help="Max executions for add_to_cart")
    parser.add_argument("--max-checkout", type=int, env_var="LOCUST_MAX_CHECKOUT", default=0, help="Max executions for checkout")
    parser.add_argument("--max-update-inventory", type=int, env_var="LOCUST_MAX_UPDATE_INVENTORY", default=0, help="Max executions for update_inventory")
    parser.add_argument("--max-view-analytics", type=int, env_var="LOCUST_MAX_VIEW_ANALYTICS", default=0, help="Max executions for view_analytics")
    parser.add_argument("--max-health-check", type=int, env_var="LOCUST_MAX_HEALTH_CHECK", default=0, help="Max executions for health_check")
    parser.add_argument("--possible-categories-list", type=str, env_var="LOCUST_POSSIBLE_CATEGORIES_LIST", default="Electronics,Books,Clothing", help="Comma-separated list of possible categories")
    parser.add_argument("--search-terms-list", type=str, env_var="LOCUST_SEARCH_TERMS_LIST", default="phone,laptop,shirt,book", help="Comma-separated list of search terms")

@events.init.add_listener
def on_locust_init(environment, **kwargs):
    init_web_ui_extension(environment)
    logger.info("Locust initialization complete")
    logger.info("Using SimulationUser. Task execution limits and test data are configurable via custom command-line arguments in the UI.")
    if load_shape:
        logger.info(f"Using load shape: {type(load_shape).__name__}")
    else:
        logger.info("Using standard Locust load control (no custom shape)") 