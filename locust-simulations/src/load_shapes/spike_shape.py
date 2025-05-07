from locust import LoadTestShape
import logging
from typing import Dict, List, Tuple, Optional

logger = logging.getLogger("spike_shape")

class SpikeLoadShape(LoadTestShape):
    """
    A load shape that creates sudden spikes in traffic to test system resilience.
    Normal load -> Spike -> Recovery -> Normal load
    """
    
    def __init__(self):
        super().__init__()
        # Default settings
        self.stages = [
            {"duration": 60, "users": 10, "spawn_rate": 10},     # Normal load for 1 minute
            {"duration": 30, "users": 100, "spawn_rate": 100},   # Spike to 100 users over 30 seconds
            {"duration": 60, "users": 100, "spawn_rate": 10},    # Maintain spike for 1 minute
            {"duration": 30, "users": 10, "spawn_rate": 100},    # Quick recovery to normal load
            {"duration": 120, "users": 10, "spawn_rate": 10},    # Continue with normal load for 2 minutes
        ]
    
    def set_stages(self, stages: List[Dict[str, int]]) -> None:
        """
        Set custom spike profile.
        
        Args:
            stages: List of stage dictionaries with duration, users, and spawn_rate
        """
        self.stages = stages
        logger.info(f"Set custom spike stages: {len(stages)} stages")
    
    def tick(self) -> Optional[Tuple[int, float]]:
        """
        Return users and spawn rate for current time.
        
        Returns:
            Tuple of (user_count, spawn_rate) or None if test is finished
        """
        run_time = self.get_run_time()
        
        # Find current stage
        elapsed = 0
        for stage in self.stages:
            if elapsed <= run_time < (elapsed + stage["duration"]):
                return stage["users"], stage["spawn_rate"]
            elapsed += stage["duration"]
        
        # All stages complete
        return None


class MultipleSpikeLoadShape(LoadTestShape):
    """
    A load shape that creates multiple spikes over time to test resilience
    under repeated stress conditions.
    """
    
    def __init__(self):
        super().__init__()
        
        # Base load level between spikes
        self.base_users = 10
        self.spawn_rate = 10
        
        # Spike configuration
        self.spike_users = 100    # Peak users during spike
        self.spike_duration = 60  # Duration of each spike in seconds
        self.recovery_duration = 120  # Recovery time between spikes
        
        # Number and timing of spikes
        self.num_spikes = 3
        self.first_spike_at = 120  # First spike after 2 minutes
        
        # Total test duration
        self.test_duration = self.first_spike_at + self.num_spikes * (self.spike_duration + self.recovery_duration)
    
    def tick(self) -> Optional[Tuple[int, float]]:
        """
        Return the number of users and spawn rate at the current time.
        
        Returns:
            Tuple of (user_count, spawn_rate) or None if test is finished
        """
        run_time = self.get_run_time()
        
        # Check if test is complete
        if run_time >= self.test_duration:
            return None
            
        # Before first spike, maintain base load
        if run_time < self.first_spike_at:
            return self.base_users, self.spawn_rate
        
        # Calculate time since first spike
        time_since_first_spike = run_time - self.first_spike_at
        
        # Determine which cycle we're in (spike + recovery)
        cycle_duration = self.spike_duration + self.recovery_duration
        current_cycle = int(time_since_first_spike / cycle_duration)
        
        # Check if we've exceeded the number of spikes
        if current_cycle >= self.num_spikes:
            return self.base_users, self.spawn_rate
        
        # Determine position within the current cycle
        cycle_position = time_since_first_spike % cycle_duration
        
        # If in spike phase
        if cycle_position < self.spike_duration:
            return self.spike_users, self.spawn_rate * 10  # Higher spawn rate during spike
        
        # Otherwise in recovery phase
        return self.base_users, self.spawn_rate 