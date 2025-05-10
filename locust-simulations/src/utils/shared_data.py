import threading
import random
import time
import logging
from typing import List, Dict, Any, Optional

logger = logging.getLogger("shared_data")

class SharedData:
    
    def __init__(self):
        self._lock = threading.RLock()
        self._products = []
        self._categories = set()
        self._last_product_update = 0
    
    def update_products(self, products: List[Dict[str, Any]]) -> None:
        with self._lock:
            self._products = products
            self._last_product_update = time.time()
            
            # Extract categories
            for product in products:
                if "category" in product and product["category"]:
                    self._categories.add(product["category"])
            
            logger.debug(f"Updated shared products: {len(products)} products, {len(self._categories)} categories")
    
    def get_products(self) -> List[Dict[str, Any]]:
        with self._lock:
            return self._products.copy()
    
    def get_random_product(self) -> Optional[Dict[str, Any]]:
        with self._lock:
            if not self._products:
                return None
            return random.choice(self._products)
    
    def get_product_by_name(self, name: str) -> Optional[Dict[str, Any]]:
        with self._lock:
            for product in self._products:
                if product.get("name") == name:
                    return product
            return None
    
    def get_categories(self) -> List[str]:
        with self._lock:
            return list(self._categories)
    
    def get_random_category(self) -> Optional[str]:
        categories = self.get_categories()
        if not categories:
            return None
        return random.choice(categories) 