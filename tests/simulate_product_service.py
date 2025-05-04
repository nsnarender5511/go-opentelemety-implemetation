# Script for simulating product service interactions
import requests
import time
import random
import uuid
import os
import json
import logging
import sys
# import argparse # Removed argparse

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
            # Don't exit, just warn and return empty list
            logging.warning("No product IDs found in the response from /products. Starting with empty list.")
            # exit(1) # REMOVED EXIT
            return [] # Return empty list
        
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

# Fetch known product IDs at startup (can now be empty)
known_product_ids = fetch_product_ids_from_api()

# --- Action Functions ---
def get_all_products():
    """Requests the list of all products and updates known_product_ids."""
    global known_product_ids # Declare intent to modify global
    response = make_request("products")

    if response is not None:
        try:
            data = response.json()
            if not isinstance(data, list):
                logging.error(f"Expected a list from GET /products during simulation, got {type(data)}.")
                return # Don't update if structure is wrong

            # Extract IDs and update global list
            current_ids = {item['productID'] for item in data if isinstance(item, dict) and 'productID' in item}
            new_ids = list(current_ids) # Convert set back to list

            if set(known_product_ids) != current_ids: # Avoid logging if no change
                 logging.info(f"GET_ALL updated known_product_ids. Old count: {len(known_product_ids)}, New count: {len(new_ids)}")
                 known_product_ids = new_ids # Update the global list

        except json.JSONDecodeError:
            logging.error("Failed to decode JSON response from GET /products during simulation.")
        except KeyError:
             logging.error("Response items from GET /products during simulation missing 'productID' key.")
        except Exception as e:
            logging.error(f"An unexpected error occurred processing GET /products response during simulation: {e}", exc_info=True)

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

def create_product(name, description, price, stock):
    """Creates a new product via POST /products."""
    payload = {
        "name": name,
        "description": description,
        "price": price,
        "stock": stock
    }
    make_request("products", method="POST", json_payload=payload)

# --- Action Configuration ---
# Define actions with function objects, initial counters, and arg generators
# Weights will be calculated dynamically
BASE_ACTION_CONFIG = {
    'GET_ALL':        {'func': get_all_products, 'count': 0, 'arg_generator': lambda: ([], {})},
    'GET_ONE_OK':     {'func': get_product_by_id, 'count': 0, 'arg_generator': lambda: ([random.choice(known_product_ids)], {})},
    'GET_ONE_404':    {'func': get_product_by_id, 'count': 0, 'arg_generator': lambda: ([NON_EXISTING_PRODUCT_ID], {})},
    'GET_ONE_INVALID':{'func': get_product_by_id, 'count': 0, 'arg_generator': lambda: ([INVALID_FORMAT_PRODUCT_ID], {})},
    'UPDATE_STOCK':   {'func': update_product_stock, 'count': 0, 'arg_generator': lambda: ([random.choice(known_product_ids), random.randint(0, 100)], {})},
    'CREATE_PRODUCT': {'func': create_product, 'count': 0, 'arg_generator': lambda: (
        [
            f"Simulated Product {uuid.uuid4().hex[:6]}",  # name
            "Created by simulation script",            # description
            round(random.uniform(5.0, 500.0), 2),      # price
            random.randint(1, 200)                     # stock
        ], 
        {}
    )},
    'BAD_PATH':       {'func': hit_invalid_path, 'count': 0, 'arg_generator': lambda: ([], {})},
    'STATUS_CHECK':   {'func': hit_status_endpoint, 'count': 0, 'arg_generator': lambda: ([], {})},
    'HEALTH_CHECK':   {'func': hit_health_endpoint, 'count': 0, 'arg_generator': lambda: ([], {})},
}

# --- Manual Argument Parsing ---

# Defaults
mode = 'fast'
gap_duration = 1.0
cli_overrides = {}
parsing_errors = []
allowed_actions = set(BASE_ACTION_CONFIG.keys())

logging.info(f"Raw arguments: {sys.argv[1:]}")

for arg in sys.argv[1:]:
    if not arg.startswith("--") or '=' not in arg:
        parsing_errors.append(f"Invalid argument format: '{arg}'. Expected format: --KEY=value")
        continue

    try:
        key_value = arg[2:] # Remove --
        key, value_str = key_value.split('=', 1)
        key_upper = key.upper()

        if key_upper == 'MODE':
            if value_str.lower() in ['fast', 'slow']:
                mode = value_str.lower()
                logging.info(f"Argument Parsed: MODE={mode}")
            else:
                parsing_errors.append(f"Invalid value for --MODE: '{value_str}'. Allowed: fast, slow")
        elif key_upper == 'GAP':
            try:
                gap_duration = float(value_str)
                if gap_duration < 0:
                    parsing_errors.append(f"Value for --GAP cannot be negative: {gap_duration}")
                else:
                    logging.info(f"Argument Parsed: GAP={gap_duration}")
            except ValueError:
                parsing_errors.append(f"Invalid float value for --GAP: '{value_str}'")
        elif key_upper in allowed_actions:
            try:
                weight = float(value_str)
                if weight < 0:
                     parsing_errors.append(f"Weight cannot be negative for --{key_upper}: {weight}")
                else:
                    cli_overrides[key_upper] = weight
                    logging.info(f"Argument Parsed: {key_upper}={weight}")
            except ValueError:
                 parsing_errors.append(f"Invalid float value for --{key_upper}: '{value_str}'")
        else:
            parsing_errors.append(f"Unknown argument key: '{key}'")

    except Exception as e:
        # Catch potential errors during splitting etc.
        parsing_errors.append(f"Error parsing argument '{arg}': {e}")

# Check for parsing errors before proceeding
if parsing_errors:
    logging.error("Errors processing command-line arguments:")
    for err in parsing_errors:
        logging.error(f"  - {err}")
    logging.error("Exiting due to invalid arguments.")
    exit(1)

# --- Weight Calculation (using manually parsed cli_overrides) ---

# 1. Calculate Initial Equal Weights (remains same)
num_actions = len(BASE_ACTION_CONFIG)
equal_weight = 1.0 / num_actions if num_actions > 0 else 0
ACTION_CONFIG = {
    name: {**details, 'weight': equal_weight}
    for name, details in BASE_ACTION_CONFIG.items()
}
logging.info(f"Initial equal weight per action: {equal_weight:.4f}")

# 2. Parse Command-Line Weight Overrides using argparse # REMOVED
# parser = argparse.ArgumentParser(description="Simulate product service load with optional action weight overrides and speed modes.")
# Add mode and gap arguments # REMOVED
# Add an argument for each possible action override # REMOVED
# args = parser.parse_args() # REMOVED
# Build cli_overrides dictionary from parsed args # REMOVED

# 3. Apply Overrides and Adjust Remaining Weights (using manually parsed cli_overrides)
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
    # Use manually parsed mode and gap_duration
    logging.info(f" - Mode: {mode}")
    if mode == 'slow':
        # Validation already done during parsing
        logging.info(f" - Gap between requests: {gap_duration:.2f}s")
    logging.info(f" - Product IDs: {len(known_product_ids)} IDs fetched from API")
    print(f"--- Running {TOTAL_REQUESTS} sequential requests ({mode} mode) ---           ")
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
                # Generate arguments - Handle potential IndexError if known_product_ids is empty
                try:
                    args, kwargs = arg_gen()
                except IndexError:
                    logging.warning(f"Skipping action {action_type} as no known product IDs are available.")
                    ACTION_CONFIG[action_type]['count'] -= 1 # Decrement count as it wasn't actually run
                    continue # Skip to next iteration

                # Call the function with generated arguments
                func_to_call(*args, **kwargs)

                # Add delay if in slow mode
                if mode == 'slow':
                    time.sleep(gap_duration)

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