import uuid
import logging
import json

# No direct client import needed here if make_request is passed
# from client import make_request 

# --- Action Functions ---
# Each function now accepts the make_request function as its first argument

def get_all_products(make_request):
    """Requests the list of all products.
    Returns the list of product IDs found, or None if the request failed or response was invalid.
    """
    response = make_request("products")
    if response is not None:
        try:
            data = response.json()
            if not isinstance(data, list):
                logging.error(f"Expected a list from GET /products during simulation, got {type(data)}.")
                return None # Indicate error/invalid structure

            # Extract IDs
            current_ids = [item['productID'] for item in data if isinstance(item, dict) and 'productID' in item]
            logging.info(f"GET_ALL found {len(current_ids)} products.")
            return current_ids # Return the found IDs

        except json.JSONDecodeError:
            logging.error("Failed to decode JSON response from GET /products during simulation.")
        except KeyError:
             logging.error("Response items from GET /products during simulation missing 'productID' key.")
        except Exception as e:
            logging.error(f"An unexpected error occurred processing GET /products response during simulation: {e}", exc_info=True)
    return None # Return None if request failed or processing errored

def get_product_by_id(make_request, product_id):
    """Requests a specific product by its ID."""
    make_request(f"products/{product_id}") # Response object not needed here unless we validate

def update_product_stock(make_request, product_id, new_stock):
    """Updates the stock for a specific product ID."""
    payload = {"stock": new_stock}
    make_request(f"products/{product_id}/stock", method="PATCH", json_payload=payload)

def hit_invalid_path(make_request):
    """Hits a deliberately non-existent path."""
    make_request(f"some/invalid/path/{uuid.uuid4()}")

def hit_status_endpoint(make_request):
    """Hits the /status health check endpoint."""
    make_request("status")

def hit_health_endpoint(make_request):
    """Hits the /health minimal health check endpoint."""
    make_request("health")

def create_product(make_request, name, description, price, stock):
    """Creates a new product via POST /products."""
    payload = {
        "name": name,
        "description": description,
        "price": price,
        "stock": stock
    }
    # Optionally, we could return the productID of the created product if the API provides it
    make_request("products", method="POST", json_payload=payload) 