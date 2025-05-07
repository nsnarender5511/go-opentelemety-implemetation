package debugutils

import (
	"context"
	"math/rand"
	"time"

	"github.com/narender/common/globals"
	// Import common errors package
	apierrors "github.com/narender/common/apierrors"
)

// Simulate now returns *apierrors.AppError or nil
func Simulate(ctx context.Context) *apierrors.AppError {

	if globals.Cfg().SimulateDelayEnabled {
		if globals.Cfg().SimulateDelayMinMs < 0 || globals.Cfg().SimulateDelayMaxMs <= 0 || globals.Cfg().SimulateDelayMinMs >= globals.Cfg().SimulateDelayMaxMs {
			// Invalid config for delay, but proceed to error simulation
		} else {
			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)

			delayRange := globals.Cfg().SimulateDelayMaxMs - globals.Cfg().SimulateDelayMinMs
			randomDelayMs := rng.Intn(delayRange+1) + globals.Cfg().SimulateDelayMinMs
			delayDuration := time.Duration(randomDelayMs) * time.Millisecond

			time.Sleep(delayDuration)
		}
	}

	// --- New Simple Error Simulation Logic --- (No config needed for this version)
	// Seed again just in case delay wasn't enabled/run
	// Note: Seeding frequently is okay for this debug purpose, not ideal for crypto.
	// source := rand.NewSource(time.Now().UnixNano() + 1) // Add offset to avoid same seed as delay potentially
	// rng := rand.New(source)

	// if rng.Intn(5) == 0 { // Approx 20% chance
	// 	// Return a predefined AppError directly
	// 	return apierrors.NewAppError(apierrors.ErrCodeUnknown, "Simulated debug error from Simulate()", nil)
	// }

	// If no error simulated, return nil
	return nil
}
