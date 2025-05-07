#!/bin/bash

# Stop any running Locust instances
pkill -f locust

# Wait a moment for processes to terminate
sleep 1

# Start Locust with UI mode
cd "$(dirname "$0")/.."
./scripts/run_tests.sh ui

# Open browser to Locust UI
open http://localhost:8089 