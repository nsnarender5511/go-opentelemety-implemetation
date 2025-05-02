import requests
import time
import random
import threading
import uuid
import os
import json # Import json module
import logging
from concurrent.futures import ThreadPoolExecutor

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

BASE_URL = os.getenv("PRODUCT_SERVICE_URL", "http://localhost:8082/api/v1") # Updated default port to 8082
PRODUCTS_ENDPOINT = f"{BASE_URL}/products"
CONCURRENT_USERS = 20 # Increased concurrent users
DATA_FILE_PATH = os.getenv("DATA_FILE_PATH", "../product-service/data.json") # Relative path to data.json

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


NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}" # Generate a random non-existing ID
INVALID_FORMAT_PRODUCT_ID = "invalid-id-format"


def make_request(relative_endpoint, method="GET"):
    """Makes a request to the specified relative endpoint."""
    url = f"{BASE_URL}/{relative_endpoint}"

    try:
        start_time = time.time()
        response = requests.request(method, url, timeout=5) # Added timeout
        duration = time.time() - start_time

        if 200 <= response.status_code < 300:
            logging.info(f"SUCCESS: {method} {url} -> {response.status_code} ({duration:.2f}s)")
        else:
            # Use warning level for expected client errors (4xx), error for server errors (5xx)
            log_level = logging.WARNING if response.status_code < 500 else logging.ERROR
            logging.log(log_level, f"FAILED:  {method} {url} -> {response.status_code} {response.text[:100]} ({duration:.2f}s)")
            # Potentially add a small delay on failure only for server errors
            if response.status_code >= 500:
                 time.sleep(0.1)

    except requests.exceptions.ConnectionError as e:
        logging.error(f"CONNECTION_ERROR: {method} {url} -> {e}")
        time.sleep(1) # Longer delay on connection error
    except requests.exceptions.Timeout as e:
        logging.error(f"TIMEOUT_ERROR: {method} {url} -> {e}")
        time.sleep(0.5) # Delay on timeout
    except requests.exceptions.RequestException as e:
        logging.error(f"REQUEST_ERROR: {method} {url} -> {e}")
        time.sleep(0.2) # General request error delay

def get_all_products():
    """Requests the list of all products."""
    make_request("products") # Hit the base products endpoint

def get_product_by_id(product_id):
    """Requests a specific product by its ID."""
    make_request(f"products/{product_id}")

def hit_invalid_path():
    """Hits a deliberately non-existent path."""
    make_request(f"some/invalid/path/{uuid.uuid4()}")


stop_event = threading.Event()

def worker(worker_id):
    """Simulates a single user making various requests."""
    print(f"Worker {worker_id} started.")
    while not stop_event.is_set():
        # Removed stock-related actions, adjusted weights
        action_type = random.choices(
            population=['get_all', 'get_one_ok', 'get_one_404', 'get_one_invalid', 'bad_path'],
            weights=   [0.2,       0.4,          0.2,           0.1,            0.1], # Weights sum to 1.0
            k=1
        )[0]

        try:
            if action_type == 'get_all':
                get_all_products()
            elif action_type == 'get_one_ok':
                # Use IDs loaded from data.json
                get_product_by_id(random.choice(known_product_ids))
            elif action_type == 'get_one_404':
                get_product_by_id(NON_EXISTING_PRODUCT_ID)
            elif action_type == 'get_one_invalid':
                # Expected 404 based on Fiber routing for invalid path param format
                get_product_by_id(INVALID_FORMAT_PRODUCT_ID)
            # Removed elif blocks for get_stock actions
            elif action_type == 'bad_path':
                hit_invalid_path()

            # Random delay between requests for a single worker
            time.sleep(random.uniform(0.2, 1.5))

        except Exception as e:
            print(f"Worker {worker_id} encountered an error: {e}")
            time.sleep(1) # Avoid tight loop on error

    print(f"Worker {worker_id} stopped.")


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
