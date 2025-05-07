#!/bin/bash

# Run Locust tests in different modes
# Usage: ./run_tests.sh [MODE] [USERS] [DURATION] [HOST] [SHAPE]
# MODE: ui (default) or headless
# USERS: number of users to simulate (default: 10)
# DURATION: duration in seconds (default: 300)
# HOST: target host URL (default: http://localhost:8082 or from env var)
# SHAPE: load shape to use (stages, spike, multiple_spikes, ramping)

# Load persisted settings if file exists
if [ -f "/app/settings.env" ]; then
    echo "Loading settings from settings.env"
    source /app/settings.env
fi

MODE=${1:-ui}
USERS=${2:-10}
SPAWN_RATE=${3:-10}
DURATION=${4:-300}
HOST=${PRODUCT_SERVICE_URL:-${5:-http://localhost:8082}}
SHAPE=${LOAD_SHAPE:-${6:-""}}

# Move to the directory containing this script
cd "$(dirname "$0")/.."

# Export load shape environment variable if specified
if [ -n "$SHAPE" ]; then
    export LOAD_SHAPE="$SHAPE"
    echo "Using load shape: $SHAPE"
fi

# Create results directory if it doesn't exist
mkdir -p results

echo "Running tests in $MODE mode with $USERS users for $DURATION seconds against $HOST"

case $MODE in
    ui)
        # Start Locust with web UI and class-picker enabled
        locust -f locustfile.py --host=$HOST --class-picker
        ;;
    headless)
        # Run Locust in headless mode with class-picker enabled
        locust -f locustfile.py --headless -u $USERS -r $SPAWN_RATE -t ${DURATION}s --host=$HOST --html=results/report.html --csv=results/stats --class-picker
        echo "Test complete, results in results/ directory"
        ;;
    distributed-master)
        # Run as distributed master with class-picker enabled
        locust -f locustfile.py --master --host=$HOST --class-picker
        ;;
    distributed-worker)
        # Run as distributed worker
        MASTER_HOST=${7:-"localhost"}
        locust -f locustfile.py --worker --master-host=$MASTER_HOST
        ;;
    *)
        echo "Unknown mode: $MODE. Use 'ui', 'headless', 'distributed-master', or 'distributed-worker'."
        exit 1
        ;;
esac 