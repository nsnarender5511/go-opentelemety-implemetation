import requests
import time
import random
import uuid
import os
import json
import logging
import sys
import argparse # Import argparse

# --- Logging Configuration ---
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

# --- Global Configuration & Constants ---
BASE_URL = os.getenv("PRODUCT_SERVICE_URL", "http://localhost:8082")
TOTAL_REQUESTS = int(os.getenv("TOTAL_REQUESTS", 1000)) # Number of sequential requests
# Test IDs (Generate non-existing ID dynamically)
NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}"
INVALID_FORMAT_PRODUCT_ID = "invalid-id-format"
REQUEST_TIMEOUT = 10 # Seconds

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
        time.sleep(1) # Longer delaya
    except requests.exceptions.Timeout as e:
        logging.error(f"TIMEOUT_ERROR: {method} {url} -> {e}")
        time.sleep(0.5)
    except requests.exceptions.RequestException as e:
        logging.error(f"REQUEST_ERROR: {method} {url} -> {e}")
        time.sleep(0.2)
    
    return None # Return None if not successful or an exception occurred

# --- Initial Data Fetch --- 
def fetch_product_ids_from_api():
    """Fetches product IDs by calling the GET /products endpoint."""
    logging.info(f"Attempting to fetch product IDs from {BASE_URL}/products...")
    response = make_request("products", method="GET")

    if response is None:
        logging.error("Failed to get response from /products endpoint. Exiting.")
        exit(1)

    try:
        data = response.json()
        if not isinstance(data, list):
            logging.error(f"Expected a list from /products, got {type(data)}. Exiting.")
            exit(1)
        
        ids = [item['productID'] for item in data if isinstance(item, dict) and 'productID' in item]
        
        if not ids:
            logging.error("No product IDs found in the response from /products. Exiting.")
            exit(1)
        
        logging.info(f"Successfully fetched {len(ids)} product IDs from API.")
        return ids

    except json.JSONDecodeError:
        logging.error("Failed to decode JSON response from /products. Exiting.")
        exit(1)
    except KeyError:
        logging.error("Response items from /products missing 'productID' key. Exiting.")
        exit(1)
    except Exception as e:
        logging.error(f"An unexpected error occurred processing /products response: {e}. Exiting.", exc_info=True)
        exit(1)

# Fetch known product IDs at startup
known_product_ids = fetch_product_ids_from_api()

# --- Action Functions ---
def get_all_products():
    """Requests the list of all products."""
    make_request("products")

def get_product_by_id(product_id):
    """Requests a specific product by its ID."""
    make_request(f"products/{product_id}")

def update_product_stock(product_id, new_stock):
    """Updates the stock for a specific product ID."""
    payload = {"stock": new_stock}
    make_request(f"products/{product_id}/stock", method="PATCH", json_payload=payload)

def hit_invalid_path():
    """Hits a deliberately non-existent path."""
    make_request(f"some/invalid/path/{uuid.uuid4()}")

def hit_status_endpoint():
    """Hits the /status health check endpoint."""
    make_request("status")

def hit_health_endpoint():
    """Hits the /health minimal health check endpoint."""
    make_request("health")

# --- Action Configuration ---
# Define actions with function objects, initial counters, and arg generators
# Weights will be calculated dynamically
BASE_ACTION_CONFIG = {
    'GET_ALL':        {'func': get_all_products, 'count': 0, 'arg_generator': lambda: ([], {})},
    'GET_ONE_OK':     {'func': get_product_by_id, 'count': 0, 'arg_generator': lambda: ([random.choice(known_product_ids)], {})},
    'GET_ONE_404':    {'func': get_product_by_id, 'count': 0, 'arg_generator': lambda: ([NON_EXISTING_PRODUCT_ID], {})},
    'GET_ONE_INVALID':{'func': get_product_by_id, 'count': 0, 'arg_generator': lambda: ([INVALID_FORMAT_PRODUCT_ID], {})},
    'BAD_PATH':       {'func': hit_invalid_path, 'count': 0, 'arg_generator': lambda: ([], {})},
    'STATUS_CHECK':   {'func': hit_status_endpoint, 'count': 0, 'arg_generator': lambda: ([], {})},
    'HEALTH_CHECK':   {'func': hit_health_endpoint, 'count': 0, 'arg_generator': lambda: ([], {})},
    'UPDATE_STOCK':   {'func': update_product_stock, 'count': 0, 'arg_generator': lambda: ([random.choice(known_product_ids), random.randint(0, 100)], {})}
}

# 1. Calculate Initial Equal Weights
num_actions = len(BASE_ACTION_CONFIG)
equal_weight = 1.0 / num_actions if num_actions > 0 else 0
ACTION_CONFIG = {
    name: {**details, 'weight': equal_weight}
    for name, details in BASE_ACTION_CONFIG.items()
}
logging.info(f"Initial equal weight per action: {equal_weight:.4f}")

# 2. Parse Command-Line Weight Overrides using argparse
parser = argparse.ArgumentParser(description="Simulate product service load with optional action weight overrides.")

# Add an argument for each possible action override
for action_name in BASE_ACTION_CONFIG.keys():
    parser.add_argument(
        f"--{action_name}",
        type=float,
        default=None, # Default to None to detect if it was provided
        metavar='WEIGHT',
        help=f"Override weight for {action_name} action (0.0 to 1.0)."
    )

args = parser.parse_args()

# Build cli_overrides dictionary from parsed args
cli_overrides = {}
override_errors = []
for action_name in ACTION_CONFIG.keys():
    weight = getattr(args, action_name, None) # Use getattr safely
    if weight is not None:
        if weight < 0:
            override_errors.append(f"Weight cannot be negative for {action_name}: {weight}")
        else:
            cli_overrides[action_name] = weight
            logging.info(f"CLI override received: {action_name}={weight}")

# Check for errors accumulated during override processing
if override_errors:
    logging.error("Errors processing command-line weight overrides:")
    for err in override_errors:
        logging.error(f"  - {err}")
    logging.error("Exiting due to invalid overrides.")
    exit(1)

# 3. Apply Overrides and Adjust Remaining Weights
if cli_overrides:
    total_overridden_weight = sum(cli_overrides.values())
    num_overridden = len(cli_overrides)
    num_total_actions = len(ACTION_CONFIG)
    num_non_overridden = num_total_actions - num_overridden

    if total_overridden_weight > 1.0 + 1e-9: # Add tolerance for float comparison
         logging.error(f"Sum of overridden weights ({total_overridden_weight}) exceeds 1.0. Cannot adjust. Exiting.")
         exit(1)

    remaining_weight_pool = max(0.0, 1.0 - total_overridden_weight) # Ensure pool is not negative

    adjusted_weight = 0.0
    if num_non_overridden > 0:
        adjusted_weight = remaining_weight_pool / num_non_overridden
        logging.info(f"Adjusting {num_non_overridden} non-overridden actions to weight: {adjusted_weight:.4f} each")
    elif abs(remaining_weight_pool) > 1e-9: # All overridden, but sum isn't 1.0
         logging.warning(f"All actions overridden, but weights sum to {total_overridden_weight:.4f} (should be 1.0).")

    # Apply the overrides and adjustments
    for action_name, config in ACTION_CONFIG.items():
        if action_name in cli_overrides:
            config['weight'] = cli_overrides[action_name]
        elif num_non_overridden > 0: # Only adjust if there were non-overridden actions
            config['weight'] = adjusted_weight
        # If num_non_overridden is 0, weights are kept as specified in overrides

# 4. Prepare Final Lists for random.choices
ACTION_POPULATION = list(ACTION_CONFIG.keys())
ACTION_WEIGHTS = [config['weight'] for config in ACTION_CONFIG.values()]

# Validate Final Action Weights (optional sanity check)
final_sum = sum(ACTION_WEIGHTS)
if not (1.0 - 1e-9 < final_sum < 1.0 + 1e-9): # Use tolerance
    logging.warning(f"Final action weights sum to {final_sum:.4f}, which is unexpected after adjustment. Check logic.")
else:
    logging.info(f"Final action weights calculated. Sum: {final_sum:.4f}")

# --- Main Execution ---
if __name__ == "__main__":
    logging.info(f"Starting Product Service Simulation (Sequential)...")
    logging.info(f" - Target: {BASE_URL}")
    logging.info(f" - Total Requests: {TOTAL_REQUESTS}")
    logging.info(f" - Product IDs: {len(known_product_ids)} IDs fetched from API")
    print(f"--- Running {TOTAL_REQUESTS} sequential requests ---           ")
    print("--- Press Ctrl+C to stop early ---           ")

    try:
        # Main sequential loop
        for i in range(TOTAL_REQUESTS):
            # Select action based on globally defined weights
            action_type = random.choices(
                population=ACTION_POPULATION,
                weights=ACTION_WEIGHTS,
                k=1
            )[0]

            # Increment counter for the selected action
            ACTION_CONFIG[action_type]['count'] += 1

            # Get the function and argument generator
            action_details = ACTION_CONFIG[action_type]
            func_to_call = action_details['func']
            arg_gen = action_details['arg_generator']

            try:
                # Generate arguments
                args, kwargs = arg_gen()

                # Call the function with generated arguments
                func_to_call(*args, **kwargs)

            except Exception as e:
                # Log error but continue the loop
                logging.error(f"Request {i+1}/{TOTAL_REQUESTS} encountered an error during action {action_type}: {e}", exc_info=True)

    except KeyboardInterrupt:
        print("\\nCtrl+C detected. Stopping sequential execution early...")
        logging.info("KeyboardInterrupt received. Stopping sequential execution...")
    finally:
        logging.info("Sequential execution finished or stopped.")

        # Print final action counts (remains the same)
        print("\\n--- Final Action Counts ---")
        sorted_actions = sorted(ACTION_CONFIG.items())
        total_actions = 0
        # Calculate total executed actions based on counters
        total_executed = sum(config['count'] for config in ACTION_CONFIG.values())
        print(f"(Executed {total_executed} actions before stopping)")

        for action, config in sorted_actions:
            count = config['count']
            print(f"  {action:<15}: {count}")
        print("---------------------------")
        print(f"  {'Total Executed':<15}: {total_executed}")
        print("---------------------------")

    logging.info("Simulation Complete.")
    print("\\nSimulation Complete.")
