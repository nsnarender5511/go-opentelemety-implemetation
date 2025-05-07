import logging
import random
from locust import task, tag
from src.utils.config_loader import load_config
from src.utils.validators import validate_response, check_status_code, check_content_type
from src.users.base_user import BaseUser
from src.data.test_data import USER_PROFILES, STOCK_UPDATES

# Get configuration
config = load_config()
weights = USER_PROFILES.get("admin", {})

logger = logging.getLogger("admin_user")

class AdminUser(BaseUser):
    """
    User that performs administrative tasks like stock updates.
    Represents staff or automated inventory management systems.
    """
    
    # Set weight relative to other user types - fewer admin users
    weight = config.get("user_type_weights", {}).get("admin_user", 1)
    
    # Override basic task weights from base user
    browse_all_products = task(weights.get("browse_products", 3))(BaseUser.browse_all_products)
    get_products_by_category = task(weights.get("search_by_category", 2))(BaseUser.get_products_by_category)
    health_check = task(5)(BaseUser.health_check)  # Higher health check frequency for monitoring
    
    @task(weights.get("view_product_details", 5))
    @tag("details", "admin")
    def view_product_details(self):
        """View detailed product information to check current state."""
        product = self.shared_data.get_random_product()
        if not product:
            logger.debug("No products available for details lookup")
            return
        
        with self.client.post(
            "/products/details",
            json={"name": product["name"]},
            name="Admin_Get_Product_Details",
            catch_response=True
        ) as response:
            valid = validate_response(response, [
                check_status_code(200),
                check_content_type()
            ])
            
            if valid:
                logger.debug(f"Admin got details for product: {product['name']}")
    
    @task(weights.get("update_stock", 10))
    @tag("admin", "stock")
    def update_stock(self):
        """Update stock level for a product."""
        # Sometimes use predefined stock update scenarios
        if random.random() < 0.3:  # 30% chance to use test data
            stock_update = random.choice(STOCK_UPDATES)
            product_name = stock_update["name"]
            new_stock = stock_update["stock"]
            expected_status = stock_update["expected_status"]
        else:
            # Get a random product from shared data
            product = self.shared_data.get_random_product()
            if not product:
                logger.debug("No products available for stock update")
                return
                
            product_name = product["name"]
            new_stock = random.randint(0, 100)
            expected_status = 200
        
        with self.client.patch(
            "/products/stock",
            json={"name": product_name, "stock": new_stock},
            name="Update_Product_Stock",
            catch_response=True
        ) as response:
            # Validate response based on expected status
            if response.status_code == expected_status:
                if expected_status == 200:
                    logger.debug(f"Updated stock for {product_name} to {new_stock}")
                else:
                    logger.debug(f"Got expected error {expected_status} for {product_name}")
                response.success()
            else:
                logger.warning(f"Unexpected status {response.status_code} when updating stock for {product_name}")
                response.failure(f"Expected {expected_status}, got {response.status_code}")
    
    @task(2)
    @tag("admin", "advanced")
    def check_all_products_inventory(self):
        """
        Advanced admin scenario: Get all products and check inventory levels.
        This simulates an inventory audit process.
        """
        # Step 1: Get all products
        with self.client.get("/products", name="Admin_Get_All_Products") as response:
            if response.status_code != 200:
                logger.warning(f"Failed to get products for inventory check: {response.status_code}")
                return
                
            products = self._extract_products(response)
            if not products:
                logger.debug("No products found for inventory check")
                return
            
            # Step 2: Check a sample of product details
            sample_size = min(5, len(products))
            sample_products = random.sample(products, sample_size)
            
            for product in sample_products:
                with self.client.post(
                    "/products/details",
                    json={"name": product["name"]},
                    name="Inventory_Check_Details",
                    catch_response=True
                ) as detail_response:
                    if detail_response.status_code == 200:
                        try:
                            # Check if stock value exists and is an integer
                            product_data = detail_response.json()
                            if "stock" in product_data and isinstance(product_data["stock"], int):
                                if product_data["stock"] < 10:
                                    logger.info(f"Low stock alert: {product['name']} has only {product_data['stock']} units")
                        except Exception as e:
                            logger.error(f"Error parsing product details: {e}")
                    else:
                        logger.warning(f"Failed to get details for {product['name']}: {detail_response.status_code}")
                        detail_response.failure(f"Failed with status {detail_response.status_code}") 