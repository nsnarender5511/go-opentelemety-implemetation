# Renamed from simulate_product_service.py
import time
import random
import uuid
import logging
import copy

# Local imports
import config
import client
import actions

# --- Logging Configuration ---
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s - %(message)s')

# --- Argument Parsing ---
mode, gap_duration, cli_overrides = config.parse_arguments()

# --- Initial Data --- 
# We will fetch products within the loop, start with empty list
known_products = [] 

# Generate a non-existing ID for this run (still potentially useful for testing 404s if we had such an endpoint)
NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}" # Example, might not be used now
POSSIBLE_CATEGORIES = ['Electronics', 'Apparel', 'Books', 'Kitchenware', 'Furniture', 'NonExistentCategory'] # Updated categories

# --- Base Action Configuration (Modify this in config.py ideally, but showing here) ---
# REMOVED: GET_ONE_OK, GET_ONE_404, GET_ONE_INVALID, CREATE_PRODUCT
# ADDED: GET_BY_CATEGORY, GET_BY_NAME, BUY_PRODUCT
BASE_ACTION_CONFIG = {
    'GET_ALL':        {'weight': 0.15, 'count': 0, 'func': None, 'arg_generator': lambda: ([], {})},
    'GET_BY_CATEGORY':{'weight': 0.15, 'count': 0, 'func': None, 'arg_generator': None}, # Needs generator
    'GET_BY_NAME':    {'weight': 0.15, 'count': 0, 'func': None, 'arg_generator': None}, # Needs generator
    'UPDATE_STOCK':   {'weight': 0.15, 'count': 0, 'func': None, 'arg_generator': None}, # Needs generator
    'BUY_PRODUCT':    {'weight': 0.15, 'count': 0, 'func': None, 'arg_generator': None}, # Needs generator
    'BAD_PATH':       {'weight': 0.10, 'count': 0, 'func': None, 'arg_generator': lambda: ([], {})},
    'STATUS_CHECK':   {'weight': 0.05, 'count': 0, 'func': None, 'arg_generator': lambda: ([], {})}, # Renamed from /status
    'HEALTH_CHECK':   {'weight': 0.10, 'count': 0, 'func': None, 'arg_generator': lambda: ([], {})},
    # Removed INVALID_FORMAT_PRODUCT_ID action if get_product_by_id is gone
}

# --- Action Configuration Setup --- 

def create_action_config(base_config, current_known_products): 
    """Creates the final action config, populating functions and arg generators."""
    action_config = copy.deepcopy(base_config) # Deep copy

    # Assign functions from actions module
    action_config['GET_ALL']['func'] = actions.get_all_products
    action_config['GET_BY_CATEGORY']['func'] = actions.get_products_by_category
    action_config['GET_BY_NAME']['func'] = actions.get_product_by_name
    action_config['UPDATE_STOCK']['func'] = actions.update_product_stock
    action_config['BUY_PRODUCT']['func'] = actions.buy_product
    action_config['BAD_PATH']['func'] = actions.hit_invalid_path
    action_config['STATUS_CHECK']['func'] = actions.hit_status_endpoint # Uses /health path now
    action_config['HEALTH_CHECK']['func'] = actions.hit_health_endpoint

    # --- Define Argument Generators --- 
    # Note: These generators now use 'current_known_products' which is a list of product dicts

    def get_by_category_args():
        # Simple category selection
        return ([random.choice(POSSIBLE_CATEGORIES)], {})

    def get_by_name_args():
        if not current_known_products:
            logging.debug("GET_BY_NAME: No known products available, skipping generation.")
            return None # Signal to skip
        product = random.choice(current_known_products) # Choose a product dict
        return ([product['name']], {}) # Use the 'name' from the dict

    def update_stock_args():
        if not current_known_products:
            logging.debug("UPDATE_STOCK: No known products available, skipping generation.")
            return None # Signal to skip
        product = random.choice(current_known_products) # Choose a product dict
        return ([product['name'], random.randint(0, 100)], {}) # Use the 'name' from the dict

    def buy_product_args():
        if not current_known_products:
            logging.debug("BUY_PRODUCT: No known products available, skipping generation.")
            return None # Signal to skip
        product = random.choice(current_known_products) # Choose a product dict
        # Buy a small quantity to avoid depleting stock too quickly in simulation
        quantity = random.randint(1, 5) 
        return ([product['name'], quantity], {}) # Use the 'name' from the dict

    # Assign generators
    action_config['GET_BY_CATEGORY']['arg_generator'] = get_by_category_args
    action_config['GET_BY_NAME']['arg_generator'] = get_by_name_args
    action_config['UPDATE_STOCK']['arg_generator'] = update_stock_args
    action_config['BUY_PRODUCT']['arg_generator'] = buy_product_args
    # Others are simple lambdas defined inline or in base_config

    # --- Calculate Weights (Simplified - using base weights unless overridden) ---
    # You might want to retain the more complex weight calculation logic if needed
    total_override_weight = sum(cli_overrides.get(name, 0) for name in action_config)
    num_overridden = sum(1 for name in action_config if name in cli_overrides)
    num_not_overridden = len(action_config) - num_overridden
    remaining_weight_total = max(0.0, 1.0 - total_override_weight)
    weight_per_non_overridden = remaining_weight_total / num_not_overridden if num_not_overridden > 0 else 0

    logging.info(f"Calculating weights. Overrides: {cli_overrides}")
    logging.info(f"Total override weight: {total_override_weight:.4f}")
    logging.info(f"Remaining weight for {num_not_overridden} actions: {remaining_weight_total:.4f}")
    logging.info(f"Weight per non-overridden action: {weight_per_non_overridden:.4f}")

    final_total_weight = 0
    for name, details in action_config.items():
        if name in cli_overrides:
            details['weight'] = cli_overrides[name]
        elif details.get('weight') is None: # Assign default if not overridden and not in base
             details['weight'] = weight_per_non_overridden 
        # else: use weight from base_config if not overridden
        final_total_weight += details['weight']

    # Normalize weights
    if len(action_config) > 0 and abs(final_total_weight - 1.0) > 1e-9:
        logging.warning(f"Normalizing weights. Initial sum: {final_total_weight}")
        norm_factor = 1.0 / final_total_weight
        for name in action_config:
            action_config[name]['weight'] *= norm_factor

    logging.info("Final Action Weights:")
    for name, details in action_config.items():
         logging.info(f"  - {name}: {details.get('weight', 'N/A'):.4f}")

    return action_config

# Create the dynamic action config (initial call, might be rebuilt if needed)
ACTION_CONFIG = create_action_config(BASE_ACTION_CONFIG, known_products)

# --- Simulation Loop --- 
logging.info("Starting simulation...")
logging.info(f"Mode: {mode}, Gap between requests: {gap_duration}s")

def run_simulation():
    global known_products # Allow modification
    global ACTION_CONFIG     # Allow modification

    # Initialize known_products by fetching from API
    logging.info("Performing initial product fetch...")
    known_products = client.fetch_product_ids_from_api() # Changed function name's implication, it fetches dicts
    if not known_products:
         logging.warning("Initial fetch returned no products. Simulation might be limited.")
    # Rebuild action config with initially fetched products for generators
    ACTION_CONFIG = create_action_config(BASE_ACTION_CONFIG, known_products)
    action_names = list(ACTION_CONFIG.keys())
    
    while True:
        action_weights = [ACTION_CONFIG[name]['weight'] for name in action_names]
        # Filter out actions that cannot run based on current known_products
        runnable_actions = []
        runnable_weights = []
        # Use a snapshot of known_products for this iteration's checks
        temp_known_products = known_products 

        actions_requiring_products = {'UPDATE_STOCK', 'GET_BY_NAME', 'BUY_PRODUCT'}

        for name in action_names:
            details = ACTION_CONFIG[name]
            # Check if the action requires known products and if they are available
            if name in actions_requiring_products and not temp_known_products:
                logging.debug(f"Skipping {name} temporarily as known_products is empty.")
                continue # Skip this action for selection
            runnable_actions.append(name)
            runnable_weights.append(details['weight'])

        if not runnable_actions:
            logging.warning("No runnable actions available (likely waiting for initial products via GET_ALL). Forcing GET_ALL.")
            # Force GET_ALL if nothing else is possible
            chosen_action_name = 'GET_ALL'
            # If GET_ALL itself isn't runnable (e.g. weight 0), sleep and retry
            if chosen_action_name not in ACTION_CONFIG or ACTION_CONFIG[chosen_action_name]['weight'] <=0:
                 logging.error("GET_ALL action not configured or has zero weight. Cannot recover. Sleeping.")
                 time.sleep(gap_duration * 5)
                 continue
        else:
            # Normalize runnable_weights if some actions were skipped
            total_runnable_weight = sum(runnable_weights)
            if total_runnable_weight <= 0:
                 logging.error("Total runnable weight is zero or negative, cannot choose action. Skipping iteration.")
                 time.sleep(gap_duration)
                 continue
            normalized_runnable_weights = [w / total_runnable_weight for w in runnable_weights]
            # --- Action Selection ---
            chosen_action_name = random.choices(runnable_actions, weights=normalized_runnable_weights, k=1)[0]

        action_details = ACTION_CONFIG[chosen_action_name]
        action_func = action_details['func']
        arg_generator = action_details['arg_generator']

        logging.debug(f"Choosing action: {chosen_action_name}")

        # --- Argument Generation ---
        # Note: arg_generator uses the *current* known_products via the closure
        # in create_action_config, but we pass it explicitly if we rebuild config often
        # For now, it uses the list available when create_action_config was last called.
        # Rebuilding config inside the loop might be better if product list changes drastically. 
        generated_args = arg_generator()

        # Handle cases where generator signals impossibility (returned None)
        if generated_args is None:
             logging.warning(f"Arg generator for {chosen_action_name} returned None (likely no known products), skipping execution.")
             time.sleep(0.1) # Small pause before next choice
             continue

        pos_args, kw_args = generated_args

        # --- Action Execution ---
        logging.info(f"Executing: {chosen_action_name} with args: {pos_args}")
        action_details['count'] += 1
        try:
            # Pass the make_request function as the first argument
            result = action_func(client.make_request, *pos_args, **kw_args)

            # Handle specific action results, like updating known products
            if chosen_action_name == 'GET_ALL' and result is not None:
                # Simple comparison: update if lists differ based on names
                # Assumes result is the new list of product dicts from get_all_products
                current_names = {p['name'] for p in known_products if isinstance(p, dict) and 'name' in p}
                new_names = {p['name'] for p in result if isinstance(p, dict) and 'name' in p}
                if current_names != new_names:
                    logging.info(f"GET_ALL updated known_products. Old count: {len(known_products)}, New count: {len(result)}")
                    known_products = result # Update with the new list of dicts
                    # OPTIONAL: Rebuild ACTION_CONFIG if generators need the absolute latest list immediately
                    # ACTION_CONFIG = create_action_config(BASE_ACTION_CONFIG, known_products)
                    # action_names = list(ACTION_CONFIG.keys())

            # No action needed for CREATE_PRODUCT anymore

        except Exception as e:
            logging.error(f"Exception during action execution {chosen_action_name}: {e}", exc_info=True)

        # --- Delay ---
        if mode == 'slow':
            time.sleep(gap_duration)
        else:
            time.sleep(0.05) # 50ms delay

# --- Main Execution --- 
if __name__ == "__main__":
    try:
        run_simulation()
    except KeyboardInterrupt:
        logging.info("Simulation interrupted by user.")
    finally:
        logging.info("Simulation finished.")
        # Log final counts
        logging.info("Action Counts:")
        total_executed = 0
        # Use the final state of ACTION_CONFIG
        for name, details in ACTION_CONFIG.items():
            logging.info(f"  - {name}: {details['count']}")
            total_executed += details['count']
        logging.info(f"Total actions executed: {total_executed}")