import requests
import time
import random
import threading
import uuid
import os
import json # Import json module

# Configuration
BASE_URL = os.getenv("PRODUCT_SERVICE_URL", "http://localhost:8082")
PRODUCTS_ENDPOINT = f"{BASE_URL}/products"
CONCURRENT_USERS = 20 # Increased concurrent users
DATA_FILE_PATH = "../product-service/data.json" # Relative path to data.json

# --- Load Product Data ---
try:
    with open(DATA_FILE_PATH, 'r') as f:
        product_data = json.load(f)
    # Use actual IDs from data.json
    known_product_ids = list(product_data.keys())
    if not known_product_ids:
        print(f"Warning: No product IDs found in {DATA_FILE_PATH}. Using fallback.")
        known_product_ids = ["prod_fallback_1", "prod_fallback_2"] # Basic fallback
except FileNotFoundError:
    print(f"Error: data.json not found at {DATA_FILE_PATH}. Using fallback IDs.")
    known_product_ids = ["prod_fallback_1", "prod_fallback_2"]
except json.JSONDecodeError:
    print(f"Error: Could not decode JSON from {DATA_FILE_PATH}. Using fallback IDs.")
    known_product_ids = ["prod_fallback_1", "prod_fallback_2"]


# Product IDs for testing
NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}" # Generate a random non-existing ID
INVALID_FORMAT_PRODUCT_ID = "invalid-id-format"

# --- Request Functions ---

def make_request(method, url, expected_status=None):
    """Helper function to make requests and print status."""
    try:
        response = requests.request(method, url, timeout=10) # Increased timeout slightly
        status_match = "OK" if expected_status and response.status_code == expected_status else "UNEXPECTED"
        
        log_message = f"-> {method} {url}: Status {response.status_code} ({status_match})"
        if expected_status and response.status_code != expected_status:
            log_message += f", Expected: {expected_status}"
            
        print(log_message)

        # Optionally: response.raise_for_status() # Or handle errors more gracefully
    except requests.exceptions.Timeout:
        print(f"-> {method} {url}: TIMEOUT")
    except requests.exceptions.ConnectionError:
        print(f"-> {method} {url}: CONNECTION ERROR (Is the service running?)")
    except requests.exceptions.RequestException as e:
        print(f"-> {method} {url}: Error {e}")
    except Exception as e: # Catch broader exceptions during request handling
        print(f"-> {method} {url}: UNEXPECTED ERROR during request: {e}")

def get_all_products():
    make_request("GET", PRODUCTS_ENDPOINT, expected_status=200)

def get_product_by_id(product_id, expected_status=200):
    url = f"{PRODUCTS_ENDPOINT}/{product_id}"
    make_request("GET", url, expected_status=expected_status)

def get_product_stock(product_id, expected_status=200):
    url = f"{PRODUCTS_ENDPOINT}/{product_id}/stock"
    make_request("GET", url, expected_status=expected_status)

def hit_invalid_path():
    url = f"{PRODUCTS_ENDPOINT}/some/invalid/path/{uuid.uuid4()}"
    make_request("GET", url, expected_status=404) # Fiber often returns 404 for bad routes

# --- Simulation Worker ---

stop_event = threading.Event()

def worker(worker_id):
    """Simulates a single user making various requests."""
    print(f"Worker {worker_id} started.")
    while not stop_event.is_set():
        action_type = random.choices(
            population=['get_all', 'get_one_ok', 'get_one_404', 'get_one_invalid', 
                        'get_stock_ok', 'get_stock_404', 'get_stock_invalid', 'bad_path'],
            weights=[0.1, 0.2, 0.1, 0.1, 0.2, 0.1, 0.1, 0.1], # Adjust weights as needed
            k=1
        )[0]

        try:
            if action_type == 'get_all':
                get_all_products()
            elif action_type == 'get_one_ok':
                # Use IDs loaded from data.json
                get_product_by_id(random.choice(known_product_ids), expected_status=200)
            elif action_type == 'get_one_404':
                get_product_by_id(NON_EXISTING_PRODUCT_ID, expected_status=404)
            elif action_type == 'get_one_invalid':
                # Expected 404 based on Fiber routing for invalid path param format
                get_product_by_id(INVALID_FORMAT_PRODUCT_ID, expected_status=404)
            elif action_type == 'get_stock_ok':
                # Use IDs loaded from data.json
                get_product_stock(random.choice(known_product_ids), expected_status=200)
            elif action_type == 'get_stock_404':
                get_product_stock(NON_EXISTING_PRODUCT_ID, expected_status=404)
            elif action_type == 'get_stock_invalid':
                # Expected 404 based on Fiber routing for invalid path param format
                get_product_stock(INVALID_FORMAT_PRODUCT_ID, expected_status=404)
            elif action_type == 'bad_path':
                hit_invalid_path()

            # Random delay between requests for a single worker
            time.sleep(random.uniform(0.2, 1.5))

        except Exception as e:
            print(f"Worker {worker_id} encountered an error: {e}")
            time.sleep(1) # Avoid tight loop on error

    print(f"Worker {worker_id} stopped.")

# --- Main Execution ---

if __name__ == "__main__":
    print(f"Starting Product Service Simulation...")
    print(f" - Simulating {CONCURRENT_USERS} concurrent users indefinitely.")
    print(f" - Target: {BASE_URL}")
    print(f" - Known IDs from {DATA_FILE_PATH}: {known_product_ids if known_product_ids else 'None found, using fallback'}")
    print("--- Press Ctrl+C to stop ---")

    threads = []
    for i in range(CONCURRENT_USERS):
        thread = threading.Thread(target=worker, args=(i + 1,))
        threads.append(thread)
        thread.start()

    # Let the simulation run indefinitely until interrupted
    try:
        # Keep the main thread alive while workers run
        while True:
            # Check if any worker thread has unexpectedly died
            for t in threads:
                if not t.is_alive():
                    print(f"Warning: Worker thread {t.name} is no longer alive.")
                    # Optionally: Respawn the worker thread if needed
            time.sleep(5) # Check thread status periodically
    except KeyboardInterrupt:
        print("\nCtrl+C detected. Stopping simulation...")
    finally:
        # Signal workers to stop and wait for them
        stop_event.set()
        print("Waiting for workers to finish...")
        for thread in threads:
            thread.join()

    print("\nSimulation Complete.")
