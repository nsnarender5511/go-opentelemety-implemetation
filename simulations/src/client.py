import requests
import time
import logging
import json
import os

# Import necessary config values
from config import BASE_URL, REQUEST_TIMEOUT

# --- Core Request Function ---
def make_request(relative_endpoint, method="GET", json_payload=None):
    """Makes a request to the specified relative endpoint.
    Returns the requests.Response object on success (2xx status), None otherwise.
    """
    url = f"{BASE_URL}/{relative_endpoint}"
    response = None # Initialize response to None
    try:
        start_time = time.time()
        response = requests.request(method, url, json=json_payload, timeout=REQUEST_TIMEOUT)
        duration = time.time() - start_time

        if 200 <= response.status_code < 300:
            logging.info(f"SUCCESS: {method} {url} -> {response.status_code} ({duration:.2f}s)")
            return response # Return response object on success
        else:
            log_level = logging.WARNING if response.status_code < 500 else logging.ERROR
            logging.log(log_level, f"FAILED:  {method} {url} -> {response.status_code} {response.text[:100]} ({duration:.2f}s)")
            if response.status_code >= 500:
                 time.sleep(0.1) # Small delay on server error

    except requests.exceptions.ConnectionError as e:
        logging.error(f"CONNECTION_ERROR: {method} {url} -> {e}")
        time.sleep(1) # Longer delay
    except requests.exceptions.Timeout as e:
        logging.error(f"TIMEOUT_ERROR: {method} {url} -> {e}")
        time.sleep(0.5)
    except requests.exceptions.RequestException as e:
        logging.error(f"REQUEST_ERROR: {method} {url} -> {e}")
        time.sleep(0.2)
    
    return None # Return None if not successful or an exception occurred

# --- Initial Data Fetch ---
def fetch_product_ids_from_api():
    """Fetches product dictionaries by calling the GET /products endpoint.
    Returns a list of product dictionaries, or an empty list if fetch fails or no products exist.
    Exits if the API response is fundamentally malformed (not list/dict, JSON error).
    """
    logging.info(f"Attempting to fetch product dictionaries from {BASE_URL}/products...")
    response = make_request("products", method="GET")

    if response is None:
        logging.error("Failed to get response from /products endpoint during initial fetch. Returning empty list.")
        return [] # Return empty list instead of exiting

    try:
        data = response.json()
        # Check if the top-level structure contains a 'data' key or is a direct list
        product_list = []
        if isinstance(data, dict) and 'data' in data and isinstance(data['data'], list):
            product_list = data['data']
        elif isinstance(data, list): # Handle potential direct list response just in case
            product_list = data
        else:
            logging.error(f"Unexpected response structure from /products: {type(data)}. Expected dict with 'data' list or direct list. Exiting.")
            exit(1) # Exit here, as it indicates a significant API contract issue

        # Filter for valid product dictionaries containing 'name'
        products = [item for item in product_list if isinstance(item, dict) and 'name' in item]
        
        if not products:
            logging.warning("No valid products found in the response from /products. Starting with empty list.")
            return [] # Return empty list
        
        logging.info(f"Successfully fetched {len(products)} products from API.")
        return products # Return list of product dictionaries

    except json.JSONDecodeError:
        logging.error("Failed to decode JSON response from /products. Exiting.")
        exit(1) # Exit here, critical error
    except Exception as e:
        logging.error(f"An unexpected error occurred processing /products response: {e}. Exiting.", exc_info=True)
        exit(1) # Exit on other unexpected errors during initial fetch 