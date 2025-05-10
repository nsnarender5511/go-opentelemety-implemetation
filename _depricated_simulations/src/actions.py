import uuid
import logging
import json

# No direct client import needed here if make_request is passed
# from client import make_request 

# --- Action Functions ---
# Each function now accepts the make_request function as its first argument

def get_all_products(make_request):
    """Requests the list of all products.
    Returns the list of product dictionaries found, or None if the request failed or response was invalid.
    """
    response = make_request("products")
    if response is not None:
        try:
            data = response.json()
            # Check if the top-level structure contains a 'data' key
            if isinstance(data, dict) and 'data' in data:
                products_list = data['data']
            elif isinstance(data, list):
                 # Handle cases where the API might return just the list directly
                 products_list = data
            else:
                logging.error(f"Expected a dict with 'data' key or a list from GET /products, got {type(data)}.")
                return None

            if not isinstance(products_list, list):
                logging.error(f"Expected 'data' field to be a list or direct list response from GET /products, got {type(products_list)}.")
                return None # Indicate error/invalid structure

            # Filter for valid product dictionaries (basic check)
            valid_products = [item for item in products_list if isinstance(item, dict) and 'productID' in item and 'name' in item]
            logging.info(f"GET_ALL found {len(valid_products)} valid products.")
            return valid_products # Return the list of valid product dicts

        except json.JSONDecodeError:
            logging.error("Failed to decode JSON response from GET /products during simulation.")
        # Keep KeyError for potential issues within items if needed, though checked above
        except KeyError as e:
             logging.error(f"Response items from GET /products during simulation might be missing keys: {e}")
             # Depending on strictness, might return partial list or None
             # Let's return None for now if basic structure fails
             return None
        except Exception as e:
            logging.error(f"An unexpected error occurred processing GET /products response during simulation: {e}", exc_info=True)
    return None # Return None if request failed or processing errored

def update_product_stock(make_request, product_name, new_stock):
    """Updates the stock for a specific product NAME via PATCH /products/stock."""
    # Payload requires name, not productID, based on handler.go
    payload = {"name": product_name, "stock": new_stock}
    # Path changed from /products/{id}/stock to /products/stock
    make_request("products/stock", method="PATCH", json_payload=payload)

def hit_invalid_path(make_request):
    """Hits a deliberately non-existent path."""
    make_request(f"some/invalid/path/{uuid.uuid4()}")

def hit_status_endpoint(make_request):
    """Hits the /health health check endpoint (previously was /status)."""
    make_request("health") # Path changed from "status"

def hit_health_endpoint(make_request):
    """Hits the /health minimal health check endpoint."""
    make_request("health")

# --- New Action Functions based on updated API ---

def get_products_by_category(make_request, category):
    """Requests products by category via GET /products/category."""
    response = make_request(f"products/category?category={category}")
    if response is not None:
        try:
            data = response.json()
            # Check for the 'data' wrapper
            if isinstance(data, dict) and 'data' in data and isinstance(data['data'], list):
                 products_list = data['data']
            elif isinstance(data, list): # Fallback for direct list
                 products_list = data
            else:
                logging.error(f"Expected a dict with 'data' key or a list from GET /products/category, got {type(data)}.")
                return None

            # Extract product dictionaries containing 'name'
            category_products = [item for item in products_list if isinstance(item, dict) and 'name' in item]
            logging.info(f"GET_CATEGORY '{category}' found {len(category_products)} products.")
            return category_products # Return list of product dicts
        except json.JSONDecodeError:
            logging.error("Failed to decode JSON response from GET /products/category.")
        # Removed KeyError check for productID
        except Exception as e:
            logging.error(f"An unexpected error occurred processing GET /products/category response: {e}", exc_info=True)
    return None

def get_product_by_name(make_request, name):
    """Requests product details by name via POST /products/details."""
    payload = {"name": name}
    response = make_request("products/details", method="POST", json_payload=payload)
    if response is not None:
        try:
            # Assuming the response is the product details if successful
            data = response.json()
            if isinstance(data, dict) and 'productID' in data:
                 logging.info(f"GET_BY_NAME found product: {data.get('productID')}")
                 return data # Return the full product dict
            else:
                logging.error(f"Unexpected response structure from POST /products/details: {data}")
                return None
        except json.JSONDecodeError:
            logging.error("Failed to decode JSON response from POST /products/details.")
        except Exception as e:
            logging.error(f"An unexpected error occurred processing POST /products/details response: {e}", exc_info=True)
    return None

def buy_product(make_request, product_name, quantity):
    """Attempts to buy a product by NAME via POST /products/buy."""
    # Payload requires name, not productID, based on handler.go
    payload = {"name": product_name, "quantity": quantity}
    response = make_request("products/buy", method="POST", json_payload=payload)
    # Check for success based on status code (assuming 2xx is success)
    if response is not None and 200 <= response.status_code < 300:
        logging.info(f"BUY_PRODUCT successful for NAME {product_name}, quantity {quantity}.")
        return True
    else:
        status = response.status_code if response is not None else 'No Response'
        logging.warning(f"BUY_PRODUCT failed for NAME {product_name}, quantity {quantity}. Status: {status}")
        return False 