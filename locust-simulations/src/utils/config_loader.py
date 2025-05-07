import yaml
import os
import logging
from typing import Dict, Any, Optional

logger = logging.getLogger("config_loader")

def load_config(config_path: str = None) -> Dict[str, Any]:
    """
    Load configuration from a YAML file with environment variable overrides.
    
    Args:
        config_path: Path to the config YAML file. If None, will try default locations.
        
    Returns:
        Dict containing configuration settings
    """
    # Default config as fallback
    default_config = {
        "service": {"base_url": "http://localhost:8082", "request_timeout_seconds": 10},
        "scenario_weights": {
            "browse_all_products": 10,
            "get_products_by_category": 8,
            "get_product_details": 10,
            "buy_product": 7,
            "update_stock": 3,
            "hit_invalid_path": 1,
            "health_check": 2
        }
    }
    
    # Try to find config file if not specified
    if config_path is None:
        possible_paths = [
            os.path.join(os.getcwd(), "config", "config.yaml"),
            os.path.join(os.getcwd(), "config.yaml"),
            os.path.join(os.path.dirname(os.path.dirname(os.path.dirname(__file__))), "config", "config.yaml")
        ]
        
        for path in possible_paths:
            if os.path.exists(path):
                config_path = path
                break
    
    # Load from file if it exists
    config = default_config.copy()
    if config_path and os.path.exists(config_path):
        try:
            with open(config_path, 'r') as f:
                file_config = yaml.safe_load(f)
                if file_config:
                    deep_merge(config, file_config)
                    logger.info(f"Loaded configuration from {config_path}")
        except Exception as e:
            logger.error(f"Error loading config from {config_path}: {e}")
    else:
        logger.warning(f"Config file not found, using defaults")
    
    # Override with environment variables
    if "PRODUCT_SERVICE_URL" in os.environ:
        config["service"]["base_url"] = os.environ["PRODUCT_SERVICE_URL"]
        logger.info(f"Overriding service base_url from environment: {config['service']['base_url']}")
    
    return config

def deep_merge(base: Dict[str, Any], override: Dict[str, Any]) -> None:
    """
    Recursively merge two dictionaries, modifying base in-place.
    
    Args:
        base: Base dictionary to merge into
        override: Dictionary with values to override in base
    """
    for key, value in override.items():
        if key in base and isinstance(base[key], dict) and isinstance(value, dict):
            deep_merge(base[key], value)
        else:
            base[key] = value 