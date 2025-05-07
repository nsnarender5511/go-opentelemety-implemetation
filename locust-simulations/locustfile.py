"""
Main Locust entry point.
This file imports the various user classes, load shapes, and telemetry components.
"""
import os
import logging
from typing import Dict, Any

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger("locust")

# Load configuration
from src.utils.config_loader import load_config
config = load_config()

# Import user classes
from src.users.browser_user import BrowserUser
from src.users.shopper_user import ShopperUser
from src.users.admin_user import AdminUser

# Import load shape if specified (optional)
load_shape = None
load_shape_name = os.environ.get("LOAD_SHAPE", "").lower()

if load_shape_name == "stages":
    from src.load_shapes.stages_shape import StagesLoadShape
    load_shape = StagesLoadShape()
    logger.info("Using stages load shape")
elif load_shape_name == "spike":
    from src.load_shapes.spike_shape import SpikeLoadShape
    load_shape = SpikeLoadShape()
    logger.info("Using spike load shape")
elif load_shape_name == "multiple_spikes":
    from src.load_shapes.spike_shape import MultipleSpikeLoadShape
    load_shape = MultipleSpikeLoadShape()
    logger.info("Using multiple spikes load shape")
elif load_shape_name == "ramping":
    from src.load_shapes.stages_shape import RampingLoadShape
    load_shape = RampingLoadShape()
    logger.info("Using ramping load shape")
elif load_shape_name:
    logger.warning(f"Unknown load shape: {load_shape_name}")

# Set up telemetry
from src.telemetry.monitoring import setup_opentelemetry
from src.telemetry.event_hooks import register_stats_event_handlers

# Initialize OpenTelemetry if configured and enabled
setup_opentelemetry(config)

# Register custom statistics event handlers
register_stats_event_handlers()

logger.info("Locust initialization complete")
logger.info(f"Available user classes: BrowserUser, ShopperUser, AdminUser")
if load_shape:
    logger.info(f"Using load shape: {type(load_shape).__name__}")
else:
    logger.info("Using standard Locust load control (no custom shape)") 