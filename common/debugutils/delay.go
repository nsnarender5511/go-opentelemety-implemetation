package debugutils

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/narender/common/config"
)

// Simulate introduces a random delay based on the global configuration settings.
// It checks if delay simulation is enabled via config.Get().
// If enabled, it sleeps for a random duration between Min and Max milliseconds.
// Logs a warning if config.Get() returns a potentially default/uninitialized config.
func Simulate(ctx context.Context) {
	cfg := config.Get() // Fetch the global configuration

	// Basic check: If config seems default (e.g., returned due to Get() called before Load()), log warning.
	// This check might need refinement based on how Get handles uninitialized state.
	if cfg.ServiceName == "" { // Example check, might need a better indicator
		log.Println("WARN: Simulate called with potentially uninitialized config. Delay simulation might use defaults or be disabled.")
	}

	if cfg.SimulateDelayEnabled {
		if cfg.SimulateDelayMinMs < 0 || cfg.SimulateDelayMaxMs <= 0 || cfg.SimulateDelayMinMs >= cfg.SimulateDelayMaxMs {
			log.Printf("WARN: Invalid delay configuration: Min=%dms, Max=%dms. Skipping delay.", cfg.SimulateDelayMinMs, cfg.SimulateDelayMaxMs)
			return
		}
		// Seed random number generator (ideally once globally, but doing it here for simplicity)
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		delayRange := cfg.SimulateDelayMaxMs - cfg.SimulateDelayMinMs
		randomDelayMs := rng.Intn(delayRange+1) + cfg.SimulateDelayMinMs // +1 to include MaxMs
		delayDuration := time.Duration(randomDelayMs) * time.Millisecond

		log.Printf("DEBUG: Simulating delay for %v", delayDuration)
		time.Sleep(delayDuration)
	}
}
