from locust import LoadTestShape
import logging
from typing import Dict, List, Tuple, Optional

logger = logging.getLogger("stages_shape")

class StagesLoadShape(LoadTestShape):
    """
    A load shape with multiple stages for gradual ramp-up, plateau, and ramp-down.
    Each stage has a specified duration, target users, and spawn rate.
    """
    
    # Default stages if not provided
    default_stages = [
        {"duration": 60, "users": 10, "spawn_rate": 10},    # Ramp up to 10 users over 1 minute
        {"duration": 300, "users": 50, "spawn_rate": 10},   # Ramp up to 50 users over 5 minutes
        {"duration": 600, "users": 50, "spawn_rate": 10},   # Stay at 50 users for 10 minutes
        {"duration": 120, "users": 0, "spawn_rate": 10},    # Ramp down to 0 over 2 minutes
    ]
    
    def __init__(self):
        super().__init__()
        self.stages = self.default_stages
        
    def set_stages(self, stages: List[Dict[str, int]]) -> None:
        """
        Set custom stages for the load shape.
        
        Args:
            stages: List of stage dictionaries, each with duration, users, and spawn_rate keys
        """
        self.stages = stages
        logger.info(f"Set custom load stages: {len(stages)} stages")
    
    def tick(self) -> Optional[Tuple[int, float]]:
        """
        Return the number of users and spawn rate for the current time.
        
        Returns:
            Tuple of (user_count, spawn_rate) or None if the test is finished
        """
        run_time = self.get_run_time()
        
        elapsed = 0
        for stage in self.stages:
            if elapsed <= run_time < (elapsed + stage["duration"]):
                target_users = stage["users"]
                spawn_rate = stage["spawn_rate"]
                return target_users, spawn_rate
            elapsed += stage["duration"]
            
        return None  # All stages complete, test finished


class RampingLoadShape(LoadTestShape):
    """
    A load shape that continuously increases users until reaching a maximum,
    plateaus for a specified time, then ramps down.
    """
    
    def __init__(self):
        super().__init__()
        self.ramp_up_time = 600  # 10 minutes ramp-up
        self.ramp_up_users = 100  # Target 100 users
        self.plateau_time = 1200  # 20 minutes plateau
        self.ramp_down_time = 300  # 5 minutes ramp-down
        
    def tick(self) -> Optional[Tuple[int, float]]:
        """
        Return the number of users and spawn rate for the current time.
        
        Returns:
            Tuple of (user_count, spawn_rate) or None if the test is finished
        """
        run_time = self.get_run_time()
        
        # Ramp-up phase
        if run_time < self.ramp_up_time:
            # Linear ramp-up from 0 to ramp_up_users
            target_users = int((run_time / self.ramp_up_time) * self.ramp_up_users)
            return target_users, self.ramp_up_users / self.ramp_up_time * 60  # Spawn rate in users/second
            
        # Plateau phase
        elif run_time < (self.ramp_up_time + self.plateau_time):
            return self.ramp_up_users, self.ramp_up_users / self.ramp_up_time * 60
            
        # Ramp-down phase
        elif run_time < (self.ramp_up_time + self.plateau_time + self.ramp_down_time):
            # Linear ramp-down from ramp_up_users to 0
            elapsed_ramp_down = run_time - (self.ramp_up_time + self.plateau_time)
            target_users = int(self.ramp_up_users - (elapsed_ramp_down / self.ramp_down_time) * self.ramp_up_users)
            return target_users, self.ramp_up_users / self.ramp_down_time * 60
            
        # Test complete
        return None 