import os
import uuid
import sys
import logging
import random # Added for arg_generator defaults

# --- Global Configuration & Constants ---
BASE_URL = os.getenv("PRODUCT_SERVICE_URL", "http://localhost:8082")
# NON_EXISTING_PRODUCT_ID needs to be dynamic if multiple test runs happen concurrently
# Let's generate it when needed or make it configurable if persistence needed
# For now, generate dynamically within the simulation logic if needed
# NON_EXISTING_PRODUCT_ID = f"prod_{uuid.uuid4()}" # Moved to simulate.py or where needed
INVALID_FORMAT_PRODUCT_ID = "invalid-id-format"
REQUEST_TIMEOUT = 10 # Seconds

# --- Base Action Configuration ---
# Define actions with function placeholders (will be replaced in simulate.py)
# arg_generator placeholders (will be created in simulate.py where known_product_ids exists)
BASE_ACTION_CONFIG = {
    'GET_ALL':        {'func': None, 'count': 0, 'arg_generator': lambda: ([], {})},
    'GET_ONE_OK':     {'func': None, 'count': 0, 'arg_generator': lambda: (["placeholder_id"], {})}, # Placeholder ID
    'GET_ONE_404':    {'func': None, 'count': 0, 'arg_generator': lambda: (["placeholder_non_existing_id"], {})}, # Placeholder ID
    'GET_ONE_INVALID':{'func': None, 'count': 0, 'arg_generator': lambda: ([INVALID_FORMAT_PRODUCT_ID], {})},
    'UPDATE_STOCK':   {'func': None, 'count': 0, 'arg_generator': lambda: (["placeholder_id", 0], {})}, # Placeholder ID and stock
    'CREATE_PRODUCT': {'func': None, 'count': 0, 'arg_generator': lambda: (
        [
            f"Simulated Product {uuid.uuid4().hex[:6]}",
            "Created by simulation script",
            round(random.uniform(5.0, 500.0), 2),
            random.randint(1, 200)
        ],
        {}
    )},
    'BAD_PATH':       {'func': None, 'count': 0, 'arg_generator': lambda: ([], {})},
    'STATUS_CHECK':   {'func': None, 'count': 0, 'arg_generator': lambda: ([], {})},
    'HEALTH_CHECK':   {'func': None, 'count': 0, 'arg_generator': lambda: ([], {})},
}


# --- Manual Argument Parsing ---
def parse_arguments():
    """Parses command line arguments manually."""
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

    return mode, gap_duration, cli_overrides 