# 🧹 Code Cleanup Plan for Signoz_assignment

This plan outlines steps to remove dead code, unnecessary comments, and apply minor simplifications to the codebase.

## Phase 1: Comment & Dead Code Removal 💬🚮

This phase focuses on removing commented-out code and elements identified as unused across the project.

### Step 1: Remove Commented-Out Code Block in `common/errors/errors.go` 💬

* **What:** Delete the large commented-out `HandleServiceError` function. This logic appears to have been superseded by the `MapErrorToResponse` method in `product-service/src/handler.go`.
* **Where:** `common/errors/errors.go:70-107`
* **How:** Delete lines 70 through 107.
* **Rationale:** Commented-out code reduces readability and can become outdated.

### Step 2: Remove Unused Error Variables in `common/errors/errors.go` 🚮

* **What:** Remove unused exported error variables. These variables (`ErrUserNotFound`, `ErrCartNotFound`, `ErrOrderNotFound`, `ErrServiceCallFailed`) are defined but not referenced anywhere in the provided codebase (`product-service` or `common`).
* **Where:** `common/errors/errors.go:11-14`
* **How:** Delete lines 11, 12, 13, and 14.
* **Rationale:** Removes unused code, simplifying the error definitions. If these are intended for future services, they can be added back then.

### Step 3: Remove Unused Function in `common/telemetry/trace.go` 🚮

* **What:** Delete the unused `GetTracer` function. The standard way to get a tracer is `otel.Tracer()`, which is used directly in `manual_spans.go`.
* **Where:** `common/telemetry/trace.go:82-85`
* **How:** Delete lines 82 through 85.
* **Rationale:** Removes unused code.

### Step 4: Remove Unused Field in `common/telemetry/types.go` 🚮

* **What:** Remove the unused `Headers` field from the `TelemetryConfig` struct. This field is defined but never populated or used during telemetry initialization.
* **Where:** `common/telemetry/types.go:14`
* **How:** Delete line 14.
* **Rationale:** Simplifies the configuration struct by removing an unused option.

### Step 5: Remove Leftover Comments in `common/telemetry/init.go` 💬

* **What:** Remove comments indicating that functions were moved (`// newResource function REMOVED...`, `// configureLogrus function REMOVED...`).
* **Where:** `common/telemetry/init.go:144`, `common/telemetry/init.go:146`
* **How:** Delete lines 144 and 146.
* **Rationale:** Improves code clarity by removing outdated structural comments.

### Step 6: Remove Commented-Out Code/Fields in `product-service/src/repository.go` 💬🚮

* **What:** Remove the commented-out logger import, logger field definition, and logger assignment in the constructor. The repository now uses the global `logrus.StandardLogger()` or obtains a logger via context.
* **Where:**
    * `product-service/src/repository.go:10` (commented import)
    * `product-service/src/repository.go:23` (commented logger field)
    * `product-service/src/repository.go:30` (commented logger assignment)
* **How:** Delete lines 10, 23, and 30.
* **Rationale:** Cleans up remnants of previous implementation patterns.

### Step 7: Remove Informational Comment in `product-service/src/main.go` 💬

* **What:** Remove the comment explaining the removal of old signal handling logic.
* **Where:** `product-service/src/main.go:81`
* **How:** Delete line 81.
* **Rationale:** The code using `lifecycle.WaitForGracefulShutdown` is self-explanatory.

### Step 8: Remove Commented-Out Code in `tests/simulate_product_service.py` 💬

* **What:** Remove commented-out variables and debugging code.
    * `EXISTING_PRODUCT_IDS` variable.
    * Commented-out response body logging logic.
* **Where:**
    * `tests/simulate_product_service.py:27`
    * `tests/simulate_product_service.py:49-54`
* **How:** Delete line 27 and lines 49 through 54.
* **Rationale:** Cleans up unused variables and debugging snippets.

## Phase 2: Simplification & Refinement ✨

This phase focuses on minor code simplifications and improving consistency or clarity.

### Step 9: Simplify Redundant Config Variables in `common/config/config.go` ✨

* **What:** Remove the package-level variables (`productServicePort`, `logLevel`, etc.) that mirror fields in the `cfg *appConfig` struct. Modify the getter functions to access the validated values directly from the `cfg` struct. Also, remove the unused `loadOnce` and `loadErr` variables, as the `sync.Once` around `cfg` initialization handles the load-once logic and error capture.
* **Where:** `common/config/config.go`
    * Remove lines 40-52 (package variable declarations).
    * Remove lines 54-55 (`loadOnce`, `loadErr`).
    * Remove lines 160-170 (assignments from `cfg` to package variables).
    * Modify getter functions (lines 260-316) to access `cfg` fields (e.g., `return cfg.ProductServicePort`).
* **How:**
    * Delete the specified lines.
    * Refactor getters:
        ```go
        // Before (Example)
        func ProductServicePort() string {
            return productServicePort // Accessing package variable
        }

        // After (Example)
        func ProductServicePort() string {
            if cfg == nil { // Add nil check for safety, though LoadConfig should prevent this state on success
                 configLogger.Panic("Configuration accessed before LoadConfig() completed successfully") // Or return default / handle differently
            }
            return cfg.ProductServicePort // Accessing struct field directly
        }
        // Apply similar changes to all getter functions.
        ```
* **Rationale:** Reduces redundancy by having a single source of truth (`cfg` struct) after validation. Simplifies the state management within the package. ⚠️ **Note:** Ensure `LoadConfig()` is always called successfully before any getter is invoked; add panic/error handling in getters if `cfg` is nil.

### Step 10: Refine Telemetry Initialization Logging in `common/telemetry/init.go` ✨

* **What:** Consolidate log level parsing and slightly simplify the `handleInitError` logic.
* **Where:** `common/telemetry/init.go`
* **How:**
    * Parse the log level once and use the `parsedLevel` for both the `setupLogger` and when calling `configureLogrus`. Remove the redundant re-parsing or passing of the string level.
    * Minor simplification in `handleInitError`: The context creation for cleanup could potentially be embedded directly within the error handling logic where `handleInitError` is called, reducing the function's scope slightly. (Optional refinement).
* **Rationale:** Improves consistency in log level handling during setup and potentially makes error handling flow slightly cleaner.

### Step 11: Review Config Helper Function Logging in `common/config/config.go` ✨

* **What:** Review the `Warnf` logging within the `getEnv...` helper functions (e.g., `getEnvBool`, `getEnvInt`). Consider if this level of logging for parsing errors (when a fallback is used) is necessary or potentially noisy.
* **Where:** `common/config/config.go:223-259`
* **How:** Either remove the `configLogger.Warnf` calls or change them to `Debugf` if the information is only useful during detailed debugging.
* **Rationale:** Reduces potential log noise during startup if environment variables are intentionally unset and relying on defaults is expected behavior.

### Step 12: Consolidate Repository Initialization in `product-service/src/repository.go` ✨

* **What:** Refactor the `NewProductRepository` function to perform the data loading (`readData`) directly within the constructor logic rather than creating the struct and then calling the method. Also, consider consistently using the global logger or accepting a logger instance.
* **Where:** `product-service/src/repository.go:25-53`
* **How:**
    ```go
    // Before (Simplified)
    func NewProductRepository() (ProductRepository, error) {
        repo := &productRepository{ /*...*/ }
        logger := logrus.StandardLogger()
        // Stat/Create file...
        if err := repo.readData(ctx); err != nil { /*...*/ }
        logger.Info("Initialized product repository")
        return repo, nil
    }

    // After (Conceptual)
    func NewProductRepository(ctx context.Context, logger *logrus.Entry) (ProductRepository, error) { // Accept logger/context
        filePath := config.DataFilepath()
        logger.WithField("path", filePath).Info("Initializing product repository...")

        // Stat/Create file... (use logger)

        // Read data bytes
        data, err := os.ReadFile(filePath)
        if err != nil { /* handle error, use logger, return */ }

        // Unmarshal data
        var productsMap map[string]Product
        if err := json.Unmarshal(data, &productsMap); err != nil { /* handle error, use logger, return */ }

        repo := &productRepository{
            products: productsMap,
            filePath: filePath,
            mu:       sync.RWMutex{}, // Initialize mutex
        }
        logger.WithField("count", len(repo.products)).Info("Initialized product repository successfully")
        return repo, nil
    }
    // Note: This also suggests passing logger/context instead of using global logger,
    // which would require changes in main.go as well. If sticking to global logger,
    // adapt the 'After' example accordingly.
    ```
* **Rationale:** Makes the constructor fully responsible for returning a ready-to-use repository instance. Improves consistency in logger usage (if logger instance is passed).

---

This plan provides a clear path to cleaning up comments, removing dead code, and making minor refinements for better clarity and consistency. Remember to test the application thoroughly after applying these changes.