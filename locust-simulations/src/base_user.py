import os
import logging
import time
from typing import Dict, Any, List, Optional, Callable

from locust import HttpUser, between
from locust.clients import ResponseContextManager

from src.utils.shared_data import SharedData
from src.utils.http_validation import validate_response
from src.utils.product_parser import parse_products_from_data

# Configure logging
logger = logging.getLogger("base_user")

class BaseAPIUser(HttpUser):
    abstract = True
    
    wait_time = between(1, 3)
    
    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.task_counts = {}
        self.use_nginx_proxy = os.environ.get("USE_NGINX_PROXY", "false").lower() == "true"
        self.service_prefix = ""  # Default empty prefix, to be overridden by subclasses
        self.service_name = ""    # Default empty name, to be overridden by subclasses
        
    def on_start(self):
        pass
    
    def _get_path(self, endpoint):
        if self.use_nginx_proxy and self.service_prefix:
            # For nginx proxy, prepend /prefix/
            return f"/{self.service_prefix}{endpoint}"
        # Direct access
        return endpoint
    
    def _can_execute_task(self, task_name):
        max_executions_arg = f"max_{task_name}"
        max_executions = getattr(self.environment.parsed_options, max_executions_arg, -1) if hasattr(self.environment, "parsed_options") else -1
        
        if max_executions == 0:
            return False
        if max_executions < 0:
            return True
        current_count = self.task_counts.get(task_name, 0)
        return current_count < max_executions
    
    def _increment_task_count(self, task_name):
        self.task_counts[task_name] = self.task_counts.get(task_name, 0) + 1
        return self.task_counts[task_name]
    
    def _extract_products(self, response: ResponseContextManager) -> List[Dict[str, Any]]:
        try:
            json_data = response.json()
            return parse_products_from_data(json_data, logger)
        except Exception as e:
            logger.error(f"Error extracting products: {e} - Response text: {response.text[:500]}")
            response.failure(f"Failed to parse JSON response: {e}")
            return []
    
    def _retry_request(self, request_func, name, validators=None, max_retries=3):
        retry_delay = 1
        
        for attempt in range(max_retries):
            try:
                response = request_func()
                
                if 500 <= response.status_code < 600:
                    if attempt < max_retries - 1:
                        logger.warning(f"{name} failed with status {response.status_code}, retrying in {retry_delay}s ({attempt+1}/{max_retries})")
                        time.sleep(retry_delay)
                        retry_delay *= 2
                        continue
                
                if validators:
                    valid = validate_response(response, validators)
                    if not valid and attempt < max_retries - 1:
                        logger.warning(f"{name} validation failed, retrying in {retry_delay}s ({attempt+1}/{max_retries})")
                        time.sleep(retry_delay)
                        retry_delay *= 2
                        continue
                
                return response
            
            except Exception as e:
                if attempt < max_retries - 1:
                    logger.error(f"Error during {name} (attempt {attempt+1}): {str(e)}")
                    time.sleep(retry_delay)
                    retry_delay *= 2
                else:
                    logger.error(f"Final error during {name} after {max_retries} attempts: {str(e)}")
        
        return None
    
    def process_products_response(self, response, update_shared_data=True):
        if response and response.status_code == 200:
            products = self._extract_products(response)
            if products and update_shared_data:
                self.shared_data.update_products(products)
            return products
        return None 