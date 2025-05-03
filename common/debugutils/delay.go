package debugutils

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/narender/common/config"
)





func Simulate(ctx context.Context) {
	cfg := config.Get() 

	
	
	if cfg.ServiceName == "" { 
		log.Println("WARN: Simulate called with potentially uninitialized config. Delay simulation might use defaults or be disabled.")
	}

	if cfg.SimulateDelayEnabled {
		if cfg.SimulateDelayMinMs < 0 || cfg.SimulateDelayMaxMs <= 0 || cfg.SimulateDelayMinMs >= cfg.SimulateDelayMaxMs {
			log.Printf("WARN: Invalid delay configuration: Min=%dms, Max=%dms. Skipping delay.", cfg.SimulateDelayMinMs, cfg.SimulateDelayMaxMs)
			return
		}
		
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		delayRange := cfg.SimulateDelayMaxMs - cfg.SimulateDelayMinMs
		randomDelayMs := rng.Intn(delayRange+1) + cfg.SimulateDelayMinMs 
		delayDuration := time.Duration(randomDelayMs) * time.Millisecond

		log.Printf("DEBUG: Simulating delay for %v", delayDuration)
		time.Sleep(delayDuration)
	}
}
