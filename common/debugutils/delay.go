package debugutils

import (
	"context"
	"math/rand"
	"time"

	"github.com/narender/common/globals"
)

func Simulate(ctx context.Context) {

	if globals.Cfg().SimulateDelayEnabled {
		if globals.Cfg().SimulateDelayMinMs < 0 || globals.Cfg().SimulateDelayMaxMs <= 0 || globals.Cfg().SimulateDelayMinMs >= globals.Cfg().SimulateDelayMaxMs {
			return
		}
		
		source := rand.NewSource(time.Now().UnixNano())
		rng := rand.New(source)

		delayRange := globals.Cfg().SimulateDelayMaxMs - globals.Cfg().SimulateDelayMinMs
		randomDelayMs := rng.Intn(delayRange+1) + globals.Cfg().SimulateDelayMinMs 
		delayDuration := time.Duration(randomDelayMs) * time.Millisecond

		time.Sleep(delayDuration)
	}
}
