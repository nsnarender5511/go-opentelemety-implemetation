import logging
from typing import List, Dict, Any

def parse_products_from_data(json_data: Any, logger_instance: logging.Logger) -> List[Dict[str, Any]]:
    """
    Parses product data from various raw JSON structures.
    """
    processed_data = json_data

    # Handle potential wrapper object (e.g., {"status": ..., "data": ...})
    if isinstance(json_data, dict) and "data" in json_data:
        # One could potentially check json_data["status"] here if needed
        processed_data = json_data["data"]
    
    if isinstance(processed_data, list):
        # Ensure all items are dicts (basic check)
        return [p for p in processed_data if isinstance(p, dict)]

    if isinstance(processed_data, dict):
        # Convert dictionary of products (keyed by name/id) to a list
        products_list = []
        for _key, product_data in processed_data.items(): # _key is unused
            if isinstance(product_data, dict) and "name" in product_data: # Basic check for product-like structure
                products_list.append(product_data)
            # Example from original SimulationUser._extract_products, can be re-added if error key is expected
            # elif _key == "error" and not isinstance(product_data, dict):
            #    logger_instance.warning(f"API error in product data: {product_data}")
        return products_list
    
    logger_instance.warning(f"Unexpected product data structure after processing: {type(processed_data)}")
    return [] 