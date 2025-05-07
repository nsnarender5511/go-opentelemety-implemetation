import logging
import random
from typing import Dict, Any, List, Optional

from locust import HttpUser, task, between, tag
from locust.clients import ResponseContextManager

from src.utils.shared_data import SharedData
from src.utils.validators import validate_response, check_status_code, check_content_type, check_products_list_schema
from src.utils.config_loader import load_config

# Get global configuration
config = load_config()

# Set up logging
logger = logging.getLogger("base_user")

# Global shared data instance for all users
shared_data = SharedData()

class BaseUser(HttpUser):
    """Base user class with common functionality for all user types."""
    
    # Default wait time between tasks (can be overridden in subclasses)
    wait_time = between(1, 3)
    
    # Get scenario weights from config
    weights = config.get("scenario_weights", {})
    
    abstract = True  # Don't instantiate this class directly
    
    def on_start(self):
        """Initialize user session data."""
        # Use shared data instance across all users
        self.shared_data = shared_data
        
        # Ensure we have product data before starting tasks
        if not self.shared_data.get_products():
            self._load_initial_products()
    
    def _load_initial_products(self):
        """Load initial product data from the API."""
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
        """
        Extract products from API response.
        
        Args:
            response: HTTP response object
            
        Returns:
            List of product dictionaries
        """
        try:
            data = response.json()
            
            # Check if response is a dict with "data" field (wrapper format)
            if isinstance(data, dict) and "data" in data:
                products_list = data["data"]
            # Or if response is a direct list
            elif isinstance(data, list):
                products_list = data
            else:
                logger.warning(f"Unexpected response structure from API: {type(data)}")
                return []
            
            # Filter for valid product dictionaries
            return [
                item for item in products_list 
                if isinstance(item, dict) and "productID" in item and "name" in item
            ]
        except Exception as e:
            logger.error(f"Error extracting products: {e}")
            return []
    
    # Common tasks that all user types can perform
    
    @task(10)  # Default weight, can be overridden
    @tag("browse")
    def browse_all_products(self):
        """Browse all available products."""
        with self.client.get("/products", name="Get_All_Products", catch_response=True) as response:
            valid = validate_response(response, [
                check_status_code(200),
                check_content_type(),
                check_products_list_schema
            ])
            
            if valid:
                products = self._extract_products(response)
                if products:
                    self.shared_data.update_products(products)
                    logger.debug(f"Found {len(products)} products")
                
    @task(8)  # Default weight, can be overridden
    @tag("browse", "category")
    def get_products_by_category(self):
        """Get products filtered by category."""
        # Use either a category from our loaded data or one from test data
        possible_categories = (
            self.shared_data.get_categories() or 
            config.get("test_data", {}).get("possible_categories", ["Electronics"])
        )
        
        # Select a random category
        category = random.choice(possible_categories)
        
        with self.client.get(
            f"/products/category",
            params={"category": category},
            name=f"Get_Products_By_Category",
            catch_response=True
        ) as response:
            valid = validate_response(response, [
                check_status_code(200),
                check_content_type()
            ])
            
            if valid:
                # Check if we received a valid products list
                products = self._extract_products(response)
                logger.debug(f"Found {len(products)} products in category '{category}'")
                
                # Add category to request name for better metrics
                if len(category) <= 20:  # Avoid overly long names
                    response.name = f"Get_Products_By_Category_{category}"
    
    @task(2)  # Default weight, can be overridden
    @tag("health")
    def health_check(self):
        """Perform a health check."""
        with self.client.get("/health", name="Health_Check") as response:
            logger.debug(f"Health check returned {response.status_code}") 