import logging
import random
from locust import task, tag
from src.utils.config_loader import load_config
from src.utils.validators import validate_response, check_status_code, check_content_type, check_product_schema
from src.users.base_user import BaseUser
from src.data.test_data import USER_PROFILES

# Get configuration
config = load_config()
weights = USER_PROFILES.get("browser", {})

logger = logging.getLogger("browser_user")

class BrowserUser(BaseUser):
    """
    User that primarily browses products with minimal buying activity.
    Represents a casual visitor who is mostly just looking around.
    """
    
    # Set weight relative to other user types
    weight = config.get("user_type_weights", {}).get("browser_user", 3)
    
    # Override task weights from base user to focus on browsing
    browse_all_products = task(weights.get("browse_products", 10))(BaseUser.browse_all_products)
    get_products_by_category = task(weights.get("search_by_category", 7))(BaseUser.get_products_by_category)
    health_check = task(2)(BaseUser.health_check)  # Keep health check at same priority
    
    @task(weights.get("view_product_details", 8))
    @tag("details")
    def view_product_details(self):
        """View detailed information about a product."""
        product = self.shared_data.get_random_product()
        if not product:
            logger.debug("No products available for details lookup")
            return
        
        with self.client.post(
            "/products/details",
            json={"name": product["name"]},
            name="Get_Product_Details",
            catch_response=True
        ) as response:
            valid = validate_response(response, [
                check_status_code(200),
                check_content_type(),
                check_product_schema
            ])
            
            if valid:
                logger.debug(f"Got details for product: {product['name']}")
    
    @task(weights.get("buy_product", 2))
    @tag("purchase")
    def buy_product(self):
        """
        Occasionally buy a product.
        Browser users buy much less frequently than shoppers.
        """
        product = self.shared_data.get_random_product()
        if not product:
            logger.debug("No products available for purchase")
            return
        
        # Browser users typically buy small quantities
        quantity = random.randint(1, 2)
        
        with self.client.post(
            "/products/buy",
            json={"name": product["name"], "quantity": quantity},
            name="Buy_Product",
            catch_response=True
        ) as response:
            # Check response but don't validate strictly - purchase may fail for valid reasons
            if response.status_code == 200:
                logger.debug(f"Successfully bought {quantity} of {product['name']}")
            elif response.status_code == 409:  # Out of stock
                logger.debug(f"Product {product['name']} is out of stock")
                response.success()  # Mark as success for this user type - expected behavior
            else:
                logger.warning(f"Failed to buy {quantity} of {product['name']}: {response.status_code}")
    
    @task(1)
    @tag("error")
    def hit_invalid_path(self):
        """
        Occasionally hit an invalid path - simulating user error or exploring.
        Browser users sometimes click unknown links or type incorrect URLs.
        """
        import uuid
        path = f"products/{uuid.uuid4()}"
        with self.client.get(path, name="Invalid_Path", catch_response=True) as response:
            if response.status_code == 404:
                logger.debug(f"Got expected 404 for invalid path")
                response.success()  # Mark as success since 404 is expected 