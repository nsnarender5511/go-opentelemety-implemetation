package debugutils

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/narender/common/globals"
	// Import common errors package
	apierrors "github.com/narender/common/apierrors"
)

// simulatedErrorBlueprint represents a blueprint for an error that can be simulated.
type simulatedErrorBlueprint struct {
	Code     string
	Category apierrors.ErrorCategory
	Message  string
}

var predefinedApplicationErrors = []simulatedErrorBlueprint{
	{Code: apierrors.ErrCodeDatabaseAccess, Category: apierrors.CategoryApplication, Message: "Simulated database access error"},
	{Code: apierrors.ErrCodeServiceUnavailable, Category: apierrors.CategoryApplication, Message: "Simulated service unavailability"},
	{Code: apierrors.ErrCodeRequestValidation, Category: apierrors.CategoryApplication, Message: "Simulated request validation error"},
	{Code: apierrors.ErrCodeInternalProcessing, Category: apierrors.CategoryApplication, Message: "Simulated internal processing error"},
	{Code: apierrors.ErrCodeSystemPanic, Category: apierrors.CategoryApplication, Message: "Simulated system panic event"},
	{Code: apierrors.ErrCodeMalformedData, Category: apierrors.CategoryApplication, Message: "Simulated malformed data error"},
	{Code: apierrors.ErrCodeNetworkError, Category: apierrors.CategoryApplication, Message: "Simulated network error"},
}

var predefinedBusinessErrors = []simulatedErrorBlueprint{
	{Code: apierrors.ErrCodeProductNotFound, Category: apierrors.CategoryBusiness, Message: "Simulated product not found error"},
	{Code: apierrors.ErrCodeInsufficientStock, Category: apierrors.CategoryBusiness, Message: "Simulated insufficient stock error"},
	{Code: apierrors.ErrCodeInvalidProductData, Category: apierrors.CategoryBusiness, Message: "Simulated invalid product data"},
}

// Simulate now returns *apierrors.AppError or nil
func Simulate(ctx context.Context) *apierrors.AppError {
	cfg := globals.Cfg() // Assuming Cfg() returns a struct that will have the new fields

	// It's good practice to seed the random number generator only once if possible,
	// but for a debug utility called potentially spread out, per-call seeding is acceptable.
	// Using a single rng instance per call, seeded once.
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	// Existing Delay Simulation Logic
	if cfg.SimulateDelayEnabled {
		// Check for valid delay configuration
		if !(cfg.SimulateDelayMinMs < 0 || cfg.SimulateDelayMaxMs <= 0 || cfg.SimulateDelayMinMs >= cfg.SimulateDelayMaxMs) {
			delayRange := cfg.SimulateDelayMaxMs - cfg.SimulateDelayMinMs
			randomDelayMs := rng.Intn(delayRange+1) + cfg.SimulateDelayMinMs
			delayDuration := time.Duration(randomDelayMs) * time.Millisecond
			time.Sleep(delayDuration)
		}
	}

	// Check if the random error simulation feature is enabled
	// Assumes SimulateRandomErrorEnabled, SimulateOverallErrorChance,
	// SimulateApplicationErrorWeight, and SimulateBusinessErrorWeight are available in cfg.
	if !cfg.SimulateRandomErrorEnabled { // Master switch for this feature
		return nil
	}

	overallErrorChance := cfg.SimulateOverallErrorChance
	if overallErrorChance <= 0 || overallErrorChance > 1.0 { // Validate and default overall chance
		overallErrorChance = 0.1
	}

	// Decide if *any* error should be thrown based on the overall chance
	if rng.Float64() < overallErrorChance {
		appWeight := cfg.SimulateApplicationErrorWeight
		bizWeight := cfg.SimulateBusinessErrorWeight

		// Ensure weights are not negative
		if appWeight < 0 {
			appWeight = 0
		}
		if bizWeight < 0 {
			bizWeight = 0
		}

		canSimulateApp := appWeight > 0 && len(predefinedApplicationErrors) > 0
		canSimulateBiz := bizWeight > 0 && len(predefinedBusinessErrors) > 0

		var chosenBlueprint *simulatedErrorBlueprint

		if canSimulateApp && !canSimulateBiz { // Only application errors are possible
			selectedIndex := rng.Intn(len(predefinedApplicationErrors))
			blblueprint := predefinedApplicationErrors[selectedIndex] // Corrected variable name
			chosenBlueprint = &blblueprint
		} else if !canSimulateApp && canSimulateBiz { // Only business errors are possible
			selectedIndex := rng.Intn(len(predefinedBusinessErrors))
			blblueprint := predefinedBusinessErrors[selectedIndex] // Corrected variable name
			chosenBlueprint = &blblueprint
		} else if canSimulateApp && canSimulateBiz { // Both categories are possible, use weights
			totalWeight := appWeight + bizWeight
			// totalWeight should be > 0 here because canSimulateApp and canSimulateBiz are true
			decisionRoll := rng.Intn(totalWeight)

			if decisionRoll < appWeight {
				selectedIndex := rng.Intn(len(predefinedApplicationErrors))
				blblueprint := predefinedApplicationErrors[selectedIndex] // Corrected variable name
				chosenBlueprint = &blblueprint
			} else {
				selectedIndex := rng.Intn(len(predefinedBusinessErrors))
				blblueprint := predefinedBusinessErrors[selectedIndex] // Corrected variable name
				chosenBlueprint = &blblueprint
			}
		}
		// If chosenBlueprint is still nil here, it means neither category could be chosen
		// (e.g., both weights were 0, or both catalogs were empty initially, covered by canSimulate checks)

		if chosenBlueprint != nil {
			errMsg := fmt.Sprintf("%s from debug utils", chosenBlueprint.Message)
			if chosenBlueprint.Category == apierrors.CategoryBusiness {
				return apierrors.NewBusinessError(chosenBlueprint.Code, errMsg, nil)
			}
			return apierrors.NewApplicationError(chosenBlueprint.Code, errMsg, nil)
		}
	}

	return nil // No error simulated
}
