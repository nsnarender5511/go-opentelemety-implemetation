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

# --- Initial Data Fetch ---
# Fetch known product IDs at startup (can now be empty)
known_product_ids = client.fetch_product_ids_from_api()

# Generate a non-existing ID for this run
NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}"

# --- Action Configuration Setup ---

def create_action_config(base_config, known_ids):
    """Creates the final action config, populating functions and arg generators."""
    action_config = copy.deepcopy(base_config) # Deep copy to avoid modifying the base

    # Assign functions from actions module
    action_config['GET_ALL']['func'] = actions.get_all_products
    action_config['GET_ONE_OK']['func'] = actions.get_product_by_id
    action_config['GET_ONE_404']['func'] = actions.get_product_by_id
    action_config['GET_ONE_INVALID']['func'] = actions.get_product_by_id
    action_config['UPDATE_STOCK']['func'] = actions.update_product_stock
    action_config['CREATE_PRODUCT']['func'] = actions.create_product
    action_config['BAD_PATH']['func'] = actions.hit_invalid_path
    action_config['STATUS_CHECK']['func'] = actions.hit_status_endpoint
    action_config['HEALTH_CHECK']['func'] = actions.hit_health_endpoint

    # Define arg generators that can close over known_ids
    def get_one_ok_args():
        if not known_ids:
            logging.warning("GET_ONE_OK: No known product IDs available, skipping action potentially.")
            # Return a placeholder or handle appropriately if needed
            # Returning None signals to potentially skip this action if chosen
            return None
        return ([random.choice(known_ids)], {})

    def update_stock_args():
        if not known_ids:
            logging.warning("UPDATE_STOCK: No known product IDs available, skipping action potentially.")
            return None
        return ([random.choice(known_ids), random.randint(0, 100)], {})

    action_config['GET_ONE_OK']['arg_generator'] = get_one_ok_args
    # Use the dynamically generated non-existing ID
    action_config['GET_ONE_404']['arg_generator'] = lambda: ([NON_EXISTING_PRODUCT_ID], {})
    # INVALID_FORMAT_PRODUCT_ID is from config
    action_config['GET_ONE_INVALID']['arg_generator'] = lambda: ([config.INVALID_FORMAT_PRODUCT_ID], {})
    action_config['UPDATE_STOCK']['arg_generator'] = update_stock_args
    # CREATE_PRODUCT, GET_ALL, BAD_PATH, STATUS_CHECK, HEALTH_CHECK generators are already fine in base_config

    # Calculate weights (similar logic as before)
    num_actions = len(action_config)
    equal_weight = 1.0 / num_actions if num_actions > 0 else 0
    total_override_weight = sum(cli_overrides.values())
    num_overridden = len(cli_overrides)
    num_not_overridden = num_actions - num_overridden

    # Calculate remaining weight for non-overridden actions
    remaining_weight_total = max(0.0, 1.0 - total_override_weight)
    weight_per_non_overridden = remaining_weight_total / num_not_overridden if num_not_overridden > 0 else 0

    logging.info(f"Calculating weights. Overrides: {cli_overrides}")
    logging.info(f"Total override weight: {total_override_weight:.4f}")
    logging.info(f"Remaining weight for {num_not_overridden} actions: {remaining_weight_total:.4f}")
    logging.info(f"Weight per non-overridden action: {weight_per_non_overridden:.4f}")

    # Apply weights
    final_total_weight = 0
    for name, details in action_config.items():
        if name in cli_overrides:
            details['weight'] = cli_overrides[name]
        else:
            details['weight'] = weight_per_non_overridden
        final_total_weight += details['weight']

    # Normalize weights if they don't exactly sum to 1 due to float precision
    if num_actions > 0 and abs(final_total_weight - 1.0) > 1e-9:
        logging.warning(f"Normalizing weights. Initial sum: {final_total_weight}")
        for name in action_config:
            action_config[name]['weight'] /= final_total_weight

    logging.info("Final Action Weights:")
    for name, details in action_config.items():
         logging.info(f"  - {name}: {details['weight']:.4f}")

    return action_config

# Create the dynamic action config
ACTION_CONFIG = create_action_config(config.BASE_ACTION_CONFIG, known_product_ids)

# --- Simulation Loop ---
logging.info("Starting simulation...")
logging.info(f"Mode: {mode}, Gap between requests: {gap_duration}s")

def run_simulation():
    global known_product_ids # Allow modification
    global ACTION_CONFIG     # Allow modification (if weights were dynamically updated)

    action_names = list(ACTION_CONFIG.keys())
    action_weights = [ACTION_CONFIG[name]['weight'] for name in action_names]

    while True:
        # Recalculate weights if needed (e.g., based on success rates - not implemented)
        # For now, weights are static after initial calculation

        # Filter out actions that cannot run (e.g., need known_ids but list is empty)
        runnable_actions = []
        runnable_weights = []
        temp_known_ids = known_product_ids # Use a snapshot for this iteration's checks

        for name in action_names:
            details = ACTION_CONFIG[name]
            # Check if the action requires known_ids and if they are available
            if name in ('GET_ONE_OK', 'UPDATE_STOCK') and not temp_known_ids:
                # logging.debug(f"Skipping {name} temporarily as known_product_ids is empty.")
                continue # Skip this action for selection if no IDs are known
            runnable_actions.append(name)
            runnable_weights.append(details['weight'])

        if not runnable_actions:
            logging.warning("No runnable actions available (possibly waiting for initial products). Retrying after delay.")
            time.sleep(gap_duration * 2)
            # Attempt to refresh known IDs
            refreshed_ids = actions.get_all_products(client.make_request)
            if refreshed_ids is not None and refreshed_ids != known_product_ids:
                 logging.info(f"Refreshed known_product_ids. Old count: {len(known_product_ids)}, New count: {len(refreshed_ids)}")
                 known_product_ids = refreshed_ids
                 # We could rebuild ACTION_CONFIG here if generators depend heavily on it,
                 # but for now, just update the global list used by generators.
            continue

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
        # Note: arg_generator now closes over the *current* known_product_ids
        generated_args = arg_generator()

        # Handle cases where generator signals impossibility (e.g., returned None)
        if generated_args is None:
             logging.warning(f"Arg generator for {chosen_action_name} returned None, skipping execution.")
             time.sleep(0.1) # Small pause before next choice
             continue

        pos_args, kw_args = generated_args

        # --- Action Execution ---
        logging.info(f"Executing: {chosen_action_name} with args: {pos_args}")
        action_details['count'] += 1
        try:
            # Pass the make_request function as the first argument
            result = action_func(client.make_request, *pos_args, **kw_args)

            # Handle specific action results, like updating known IDs
            if chosen_action_name == 'GET_ALL' and result is not None:
                # Update known_product_ids if the list actually changed
                if set(result) != set(known_product_ids):
                    logging.info(f"GET_ALL updated known_product_ids. Old count: {len(known_product_ids)}, New count: {len(result)}")
                    known_product_ids = result
                    # Note: ACTION_CONFIG arg generators will now use the updated list
                    # If weights needed dynamic update based on list size, recalculate here.
            # elif chosen_action_name == 'CREATE_PRODUCT':
            #     # If create_product returned the new ID, add it to known_product_ids
            #     pass # Assuming create_product doesn't return ID currently

        except Exception as e:
            logging.error(f"Exception during action execution {chosen_action_name}: {e}", exc_info=True)

        # --- Delay ---
        if mode == 'slow':
            time.sleep(gap_duration)
        else:
            # Optional: small delay even in fast mode to prevent overwhelming the service
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
        for name, details in ACTION_CONFIG.items():
            logging.info(f"  - {name}: {details['count']}")
            total_executed += details['count']
        logging.info(f"Total actions executed: {total_executed}")