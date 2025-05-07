import logging
import time
import csv
import os
from typing import Dict, Any
from locust import events

logger = logging.getLogger("event_hooks")

# Custom statistics to track
custom_stats = {
    "successful_purchases": 0,
    "failed_purchases": 0,
    "out_of_stock": 0,
    "stock_updates": 0,
    "product_views": 0,
    "non_existent_product_requests": 0
}

# Request response times by name
request_response_times = {}

def register_stats_event_handlers():
    """Register event handlers for custom statistics tracking."""
    
    @events.request.add_listener
    def on_request(request_type, name, response_time, response_length, exception, **kwargs):
        """Track custom metrics based on request type and outcome."""
        # Track response times
        if name not in request_response_times:
            request_response_times[name] = []
        request_response_times[name].append(response_time)
        
        # Track custom metrics based on request name and response
        response = kwargs.get("response")
        
        if "Buy_Product" in name:
            if exception or (response and response.status_code != 200):
                custom_stats["failed_purchases"] += 1
                # Check if it's specifically out of stock
                if response and response.status_code == 409:
                    custom_stats["out_of_stock"] += 1
            else:
                custom_stats["successful_purchases"] += 1
                
        elif "Update_Product_Stock" in name and not exception and response and response.status_code == 200:
            custom_stats["stock_updates"] += 1
            
        elif "Get_Product_Details" in name and not exception and response and response.status_code == 200:
            custom_stats["product_views"] += 1
            
        elif "Get_Product_Details" in name and response and response.status_code == 404:
            custom_stats["non_existent_product_requests"] += 1
    
    @events.test_start.add_listener
    def on_test_start(environment, **kwargs):
        """Reset statistics when test starts."""
        # Reset custom stats
        for key in custom_stats:
            custom_stats[key] = 0
            
        # Reset response times
        request_response_times.clear()
        
        logger.info("Test started, statistics reset")
    
    @events.test_stop.add_listener
    def on_test_stop(environment, **kwargs):
        """Generate reports when test completes."""
        # Print custom statistics to console
        logger.info("\n=== Custom Statistics ===")
        for key, value in custom_stats.items():
            logger.info(f"{key}: {value}")
        
        # Generate CSV report
        try:
            results_dir = "results"
            if not os.path.exists(results_dir):
                os.makedirs(results_dir)
                
            timestamp = time.strftime("%Y%m%d_%H%M%S")
            
            # Save custom statistics
            with open(f"{results_dir}/custom_stats_{timestamp}.csv", "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(["Metric", "Value"])
                for key, value in custom_stats.items():
                    writer.writerow([key, value])
            
            # Save response time statistics
            with open(f"{results_dir}/response_times_{timestamp}.csv", "w", newline="") as f:
                writer = csv.writer(f)
                writer.writerow(["Request", "Min", "Max", "Avg", "Median", "Count"])
                for name, times in request_response_times.items():
                    if times:
                        times.sort()
                        min_time = min(times)
                        max_time = max(times)
                        avg_time = sum(times) / len(times)
                        median_time = times[len(times) // 2]
                        writer.writerow([name, min_time, max_time, avg_time, median_time, len(times)])
            
            logger.info(f"Reports saved to {results_dir}/")
            
        except Exception as e:
            logger.error(f"Error generating reports: {e}")

# Additional custom event handlers
request_count_threshold = 1000  # Threshold to log stats during test
last_stats_time = 0
stats_interval = 60  # Log stats every 60 seconds

@events.request.add_listener
def log_periodic_stats(request_type, name, response_time, response_length, exception, **kwargs):
    """Log stats at regular intervals during the test."""
    global last_stats_time
    
    current_time = time.time()
    if current_time - last_stats_time >= stats_interval:
        last_stats_time = current_time
        
        total_requests = sum(len(times) for times in request_response_times.values())
        if total_requests >= request_count_threshold:
            logger.info(f"\n=== Periodic Stats Update ({total_requests} requests) ===")
            for key, value in custom_stats.items():
                logger.info(f"{key}: {value}") 