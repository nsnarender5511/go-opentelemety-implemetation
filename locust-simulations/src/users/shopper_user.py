import logging
import random
from locust import task, tag
from src.utils.config_loader import load_config
from src.utils.validators import validate_response, check_status_code, check_content_type, check_product_schema
from src.users.base_user import BaseUser
from src.data.test_data import USER_PROFILES, EDGE_CASE_PRODUCTS

# Get configuration
config = load_config()
weights = USER_PROFILES.get("shopper", {})

logger = logging.getLogger("shopper_user")

class ShopperUser(BaseUser):
    """
    User that primarily focuses on buying products.
    Represents customers with high purchase intent.
    """
    
    # Set weight relative to other user types
    weight = config.get("user_type_weights", {}).get("shopper_user", 2)
    
    # Override task weights from base user to focus on shopping
    browse_all_products = task(weights.get("browse_products", 5))(BaseUser.browse_all_products)
    get_products_by_category = task(weights.get("search_by_category", 5))(BaseUser.get_products_by_category)
    health_check = task(1)(BaseUser.health_check)  # Lower health check priority for this user
    
    @task(weights.get("view_product_details", 8))
    @tag("details", "shop")
    def view_product_details(self):
        """View detailed product information before purchase."""
        product = self.shared_data.get_random_product()
        if not product:
            logger.debug("No products available for details lookup")
            return
        
        with self.client.post(
            "/products/details",
            json={"name": product["name"]},
            name="Shopper_Product_Details",
            catch_response=True
        ) as response:
            valid = validate_response(response, [
                check_status_code(200),
                check_content_type(),
                check_product_schema
            ])
            
            if valid:
                logger.debug(f"Shopper viewed details for product: {product['name']}")
    
    @task(weights.get("buy_product", 10))
    @tag("purchase", "shop")
    def buy_product(self):
        """
        Purchase a product - main activity for shoppers.
        Shoppers buy more frequently and in larger quantities.
        """
        product = self.shared_data.get_random_product()
        if not product:
            logger.debug("No products available for purchase")
            return
        
        # Shoppers tend to buy larger quantities
        quantity = random.randint(1, 5)
        
        with self.client.post(
            "/products/buy",
            json={"name": product["name"], "quantity": quantity},
            name="Shopper_Buy_Product",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                logger.debug(f"Successfully bought {quantity} of {product['name']}")
            elif response.status_code == 409:  # Out of stock
                logger.warning(f"Product {product['name']} is out of stock")
                # For shoppers, out of stock is a failure (unlike browsers who expect it)
                response.failure("Product out of stock")
            else:
                logger.warning(f"Failed to buy {quantity} of {product['name']}: {response.status_code}")
                response.failure(f"Failed with status {response.status_code}")
    
    @task(3)
    @tag("purchase", "shop", "edge_case")
    def test_edge_case_purchase(self):
        """
        Test edge cases for purchases, like buying non-existent products
        or attempting to buy with invalid quantities.
        """
        edge_case = random.choice(EDGE_CASE_PRODUCTS)
        
        with self.client.post(
            "/products/buy",
            json={"name": edge_case["name"], "quantity": edge_case["quantity"]},
            name=f"Edge_Case_Purchase_{edge_case['name']}",
            catch_response=True
        ) as response:
            expected_status = edge_case["expected_status"]
            
            # Check if we got the expected error response
            if response.status_code == expected_status:
                logger.debug(f"Edge case purchase returned expected status {expected_status}")
                response.success()  # Mark as success since we expected this error
            else:
                logger.warning(f"Edge case purchase returned unexpected status: got {response.status_code}, expected {expected_status}")
                response.failure(f"Expected {expected_status}, got {response.status_code}")
    
    @task(4)
    @tag("shop", "sequence")
    def browse_then_buy_sequence(self):
        """
        Realistic shopping sequence:
        1. Browse products by category
        2. View details of a specific product
        3. Buy the product
        """
        # Step 1: Browse by category
        possible_categories = (
            self.shared_data.get_categories() or 
            config.get("test_data", {}).get("possible_categories", ["Electronics"])
        )
        category = random.choice(possible_categories)
        
        with self.client.get(
            f"/products/category",
            params={"category": category},
            name=f"Sequence_Browse_Category",
            catch_response=True
        ) as response:
            if response.status_code != 200:
                logger.warning(f"Failed to browse category: {response.status_code}")
                return
                
            products = self._extract_products(response)
            if not products:
                logger.debug(f"No products found in category {category}")
                return
                
            # Step 2: Select a random product and view details
            selected_product = random.choice(products)
            
            with self.client.post(
                "/products/details",
                json={"name": selected_product["name"]},
                name="Sequence_View_Details",
                catch_response=True
            ) as detail_response:
                if detail_response.status_code != 200:
                    logger.warning(f"Failed to view product details: {detail_response.status_code}")
                    return
                    
                # Step 3: Buy the product
                quantity = random.randint(1, 3)
                
                with self.client.post(
                    "/products/buy",
                    json={"name": selected_product["name"], "quantity": quantity},
                    name="Sequence_Buy_Product",
                    catch_response=True
                ) as buy_response:
                    if buy_response.status_code == 200:
                        logger.debug(f"Successfully completed purchase sequence for {selected_product['name']}")
                    else:
                        logger.warning(f"Failed at purchase step in sequence: {buy_response.status_code}")
                        buy_response.failure(f"Purchase step failed: {buy_response.status_code}") 