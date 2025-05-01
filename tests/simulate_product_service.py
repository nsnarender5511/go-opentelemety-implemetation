import requests
import time
import random
import threading
import uuid

# Assuming the product service runs on localhost:8082
BASE_URL = "http://localhost:8082/products"

# Product IDs from repository.go
EXISTING_PRODUCT_IDS = ["prod_123", "prod_456", "prod_789"]
NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}" # Generate a random non-existing ID
INVALID_FORMAT_PRODUCT_ID = "invalid-id-format"

# --- Request Functions ---

def make_request(method, url, expected_status=None):
    """Helper function to make requests and print status."""
    try:
        response = requests.request(method, url, timeout=5) # Added timeout
        status_match = "OK" if expected_status and response.status_code == expected_status else "UNEXPECTED"
        if expected_status and response.status_code != expected_status:
            print(f"-> {method} {url}: Status {response.status_code} ({status_match}), Expected: {expected_status}")
        else:
            print(f"-> {method} {url}: Status {response.status_code} ({status_match})")
        # Optionally: response.raise_for_status() # Or handle errors more gracefully
    except requests.exceptions.Timeout:
        print(f"-> {method} {url}: TIMEOUT")
    except requests.exceptions.ConnectionError:
        print(f"-> {method} {url}: CONNECTION ERROR (Is the service running?)")
    except requests.exceptions.RequestException as e:
        print(f"-> {method} {url}: Error {e}")

def get_all_products():
    make_request("GET", BASE_URL, expected_status=200)

def get_product_by_id(product_id, expected_status=200):
    url = f"{BASE_URL}/{product_id}"
    make_request("GET", url, expected_status=expected_status)

def get_product_stock(product_id, expected_status=200):
    url = f"{BASE_URL}/{product_id}/stock"
    make_request("GET", url, expected_status=expected_status)

def hit_invalid_path():
    url = f"{BASE_URL}/some/invalid/path/{uuid.uuid4()}"
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
                get_product_by_id(random.choice(EXISTING_PRODUCT_IDS), expected_status=200)
            elif action_type == 'get_one_404':
                get_product_by_id(NON_EXISTING_PRODUCT_ID, expected_status=404)
            elif action_type == 'get_one_invalid':
                # Depending on router, might be 400 or 404 - check service logs
                get_product_by_id(INVALID_FORMAT_PRODUCT_ID, expected_status=404) 
            elif action_type == 'get_stock_ok':
                get_product_stock(random.choice(EXISTING_PRODUCT_IDS), expected_status=200)
            elif action_type == 'get_stock_404':
                get_product_stock(NON_EXISTING_PRODUCT_ID, expected_status=404)
            elif action_type == 'get_stock_invalid':
                # Depending on router, might be 400 or 404
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
    NUM_THREADS = 5 # Number of concurrent users
    SIMULATION_DURATION_SECONDS = 30 # How long to run the simulation

    print(f"Starting Product Service Simulation...")
    print(f" - Simulating {NUM_THREADS} concurrent users for {SIMULATION_DURATION_SECONDS} seconds.")
    print(f" - Target: {BASE_URL}")
    print(f" - Known IDs: {EXISTING_PRODUCT_IDS}")
    print("---")

    threads = []
    for i in range(NUM_THREADS):
        thread = threading.Thread(target=worker, args=(i + 1,))
        threads.append(thread)
        thread.start()

    # Let the simulation run for the specified duration
    try:
        time.sleep(SIMULATION_DURATION_SECONDS)
    except KeyboardInterrupt:
        print("\nCtrl+C detected. Stopping simulation...")
    finally:
        # Signal workers to stop and wait for them
        stop_event.set()
        print("Waiting for workers to finish...")
        for thread in threads:
            thread.join()

    print("\nSimulation Complete.")
