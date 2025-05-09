# Product Service Error Handling & Logging Enhancement Plan

This document outlines a comprehensive plan to improve error handling and logging in the product service by separating errors into business and application categories, implementing structured error handling, and enhancing error recovery mechanisms.

## Current Issues

- **Narrative-style logs**: Messages use "Shop Manager" and "Front Desk" personas instead of clear technical context
- **Mixed error domains**: No clear separation between business and technical errors
- **Limited error categorization**: Error codes don't fully represent the business domain
- **Basic exception handling**: Minimal handling of unexpected errors and panics
- **Inconsistent error context**: Error information is not consistently preserved through the call chain

## Implementation Plan

### Phase 1: Error Classification Framework

**Target: common/apierrors package**

#### Step 1: Add Error Categories

**Where**: Create a new file `common/apierrors/category.go`  
**Why**: To clearly distinguish between business rule violations and technical failures  
**How**: Define an error category type with distinct values  
**When**: Initial phase before updating error codes  

```go
package apierrors

// ErrorCategory distinguishes between different types of errors
type ErrorCategory string

const (
    // CategoryBusiness represents errors related to business rules violations
    CategoryBusiness ErrorCategory = "business"
    
    // CategoryApplication represents technical and infrastructure errors
    CategoryApplication ErrorCategory = "application"
)
```

#### Step 2: Define Business Error Codes

**Where**: Create a new file `common/apierrors/business_errors.go`  
**Why**: To provide specific error codes for business domain issues  
**How**: Define constants for business error scenarios  
**When**: After defining error categories  

```go
package apierrors

// Business error codes
const (
    // Product Domain Errors
    ErrCodeProductNotFound     = "PRODUCT_NOT_FOUND"     // When product doesn't exist
    ErrCodeInsufficientStock   = "INSUFFICIENT_STOCK"    // When purchase quantity exceeds stock
    ErrCodeInvalidProductData  = "INVALID_PRODUCT_DATA"  // When product information is invalid
    ErrCodeOrderLimitExceeded  = "ORDER_LIMIT_EXCEEDED"  // When purchase exceeds allowed quantity
    ErrCodePriceMismatch       = "PRICE_MISMATCH"        // When expected and actual prices don't match
)
```

#### Step 3: Define Application Error Codes

**Where**: Create a new file `common/apierrors/application_errors.go`  
**Why**: To provide specific error codes for technical and infrastructure issues  
**How**: Define constants for application error scenarios  
**When**: After defining error categories  

```go
package apierrors

// Application error codes
const (
    // System Errors
    ErrCodeDatabaseAccess       = "DATABASE_ACCESS_ERROR"      // Database interaction failures
    ErrCodeServiceUnavailable   = "SERVICE_UNAVAILABLE"        // When a dependency is unavailable
    ErrCodeRequestValidation    = "REQUEST_VALIDATION_ERROR"   // Input validation failures
    ErrCodeInternalProcessing   = "INTERNAL_PROCESSING_ERROR"  // Logic execution failures
    ErrCodeResourceConstraint   = "RESOURCE_CONSTRAINT_ERROR"  // Resource limitations (rate limits, etc.)
    
    // Unexpected Errors
    ErrCodeSystemPanic          = "SYSTEM_PANIC"               // Recovered panics
    ErrCodeNetworkError         = "NETWORK_ERROR"              // Network-related failures
    ErrCodeMalformedData        = "MALFORMED_DATA"             // Invalid data formats (JSON parse errors, etc.)
    ErrCodeRequestTimeout       = "REQUEST_TIMEOUT"            // Operation timeouts
    ErrCodeUnknown              = "UNKNOWN_ERROR"              // Fallback for unclassified errors
)

// Deprecated error codes - for backward compatibility
const (
    ErrCodeNotFound           = ErrCodeProductNotFound
    ErrCodeValidation         = ErrCodeRequestValidation
    ErrCodeDatabase           = ErrCodeDatabaseAccess
    ErrCodeInternal           = ErrCodeInternalProcessing
)
```

#### Step 4: Enhance AppError Structure

**Where**: Update `common/apierrors/errors.go`  
**Why**: To add context and categorization to error instances  
**How**: Extend AppError struct and add helper methods  
**When**: After defining error codes and categories  

**Before**:
```go
// AppError defines a standard application error.
type AppError struct {
    Code    string // Application-specific error code
    Message string // User-friendly error message
    Err     error  // Original underlying error (optional)
}

// NewAppError creates a new AppError. Use this for generating errors.
func NewAppError(code, message string, cause error) *AppError {
    return &AppError{
        Code:    code,
        Message: message,
        Err:     cause,
    }
}
```

**After**:
```go
// AppError defines a standard application error.
type AppError struct {
    Code        string                 // Application-specific error code
    Message     string                 // User-friendly error message
    Err         error                  // Original underlying error (optional)
    RequestID   string                 // For request tracing
    Timestamp   time.Time              // When error occurred
    ContextData map[string]interface{} // Additional context
    Category    ErrorCategory          // Business or Application
}

// WithRequestID adds a request ID to the error
func (e *AppError) WithRequestID(requestID string) *AppError {
    e.RequestID = requestID
    return e
}

// WithCategory sets the error category
func (e *AppError) WithCategory(category ErrorCategory) *AppError {
    e.Category = category
    return e
}

// WithContext adds context data to the error
func (e *AppError) WithContext(key string, value interface{}) *AppError {
    if e.ContextData == nil {
        e.ContextData = make(map[string]interface{})
    }
    e.ContextData[key] = value
    return e
}

// NewAppError creates a new AppError with defaults
func NewAppError(code, message string, cause error) *AppError {
    // Determine category based on code prefix
    category := CategoryApplication
    for _, prefix := range []string{
        ErrCodeProductNotFound,
        ErrCodeInsufficientStock,
        ErrCodeInvalidProductData,
        ErrCodeOrderLimitExceeded,
        ErrCodePriceMismatch,
    } {
        if code == prefix {
            category = CategoryBusiness
            break
        }
    }
    
    return &AppError{
        Code:      code,
        Message:   message,
        Err:       cause,
        Timestamp: time.Now(),
        Category:  category,
    }
}

// NewBusinessError creates a business domain error
func NewBusinessError(code, message string, cause error) *AppError {
    return NewAppError(code, message, cause).WithCategory(CategoryBusiness)
}

// NewApplicationError creates a technical/infrastructure error
func NewApplicationError(code, message string, cause error) *AppError {
    return NewAppError(code, message, cause).WithCategory(CategoryApplication)
}
```

### Phase 2: Enhanced Error Handling Middleware

**Target: common/middleware package**

#### Step 1: Add Request ID Middleware

**Where**: Update `common/middleware/middleware.go`  
**Why**: To ensure every request has a unique identifier for tracing  
**How**: Create a middleware that adds a UUID to each request  
**When**: After error classification framework  

```go
package middleware

import (
    "context"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        requestID := c.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
            c.Set("X-Request-ID", requestID)
        }
        
        // Store in both locals for middleware and context for logging
        c.Locals("requestID", requestID)
        ctx := context.WithValue(c.UserContext(), "requestID", requestID)
        c.SetUserContext(ctx)
        
        return c.Next()
    }
}
```

#### Step 2: Implement Panic Recovery Middleware

**Where**: Update `common/middleware/middleware.go`  
**Why**: To gracefully handle panics and convert them to structured errors  
**How**: Create a middleware with defer/recover pattern  
**When**: After request ID middleware  

```go
// RecoverMiddleware handles panics gracefully
func RecoverMiddleware() fiber.Handler {
    logger := globals.Logger()
    
    return func(c *fiber.Ctx) error {
        defer func() {
            if r := recover(); r != nil {
                err, ok := r.(error)
                if !ok {
                    err = fmt.Errorf("panic: %v", r)
                }
                
                stack := string(debug.Stack())
                requestID := c.Locals("requestID").(string)
                
                logger.ErrorContext(c.UserContext(), "CRITICAL: Unhandled panic recovered",
                    slog.String("request_id", requestID),
                    slog.String("error", err.Error()),
                    slog.String("stack", stack),
                    slog.String("path", c.Path()),
                    slog.String("method", c.Method()),
                )
                
                appErr := apierrors.NewApplicationError(
                    apierrors.ErrCodeSystemPanic,
                    "A critical system error occurred. Our team has been notified.",
                    err,
                ).WithRequestID(requestID)
                
                // Handle through the normal error handler
                _ = ErrorHandler()(c, appErr)
            }
        }()
        return c.Next()
    }
}
```

#### Step 3: Enhance Error Handler Middleware

**Where**: Update `common/middleware/middleware.go`  
**Why**: To properly handle and categorize different error types  
**How**: Update the existing error handler with improved classification  
**When**: After panic recovery middleware  

**Before**:
```go
// ErrorHandler creates a Fiber error handler middleware.
func ErrorHandler() fiber.ErrorHandler {
    logger := globals.Logger()

    return func(c *fiber.Ctx, err error) error {
        var appErr *apierrors.AppError                                               
        var statusCode int = http.StatusInternalServerError                          
        var errCode string = apierrors.ErrCodeUnknown                                
        var message string = "An unexpected error occurred. Please try again later." 

        if errors.As(err, &appErr) {
            // Handle our custom AppError
            errCode = appErr.Code
            message = appErr.Message 

            // Map AppError Code to HTTP Status Code
            switch appErr.Code {
            case apierrors.ErrCodeNotFound:
                statusCode = http.StatusNotFound // 404
            case apierrors.ErrCodeValidation, apierrors.ErrCodeInsufficientStock:
                statusCode = http.StatusBadRequest // 400
            case apierrors.ErrCodeDatabase:
                statusCode = http.StatusInternalServerError // 500
            default:
                statusCode = http.StatusInternalServerError
            }
            
            logger.ErrorContext(c.UserContext(), "API Error Handled",
                slog.String("msg", message),
                slog.Any("cause", appErr.Unwrap()),
            )
        } else {
            logger.ErrorContext(c.UserContext(), "API Unhandled Error",
                slog.String("type", fmt.Sprintf("%T", err)),
                slog.String("error", err.Error()),
            )
            message = "An internal server error occurred."
        }

        // Send standardized JSON error response
        c.Status(statusCode)
        return c.JSON(apiresponses.ErrorResponse{
            Status: "error",
            Error: apiresponses.ErrorDetail{
                Code:    errCode,
                Message: message,
            },
        })
    }
}
```

**After**:
```go
// ErrorHandler creates a Fiber error handler middleware.
func ErrorHandler() fiber.ErrorHandler {
    logger := globals.Logger()

    return func(c *fiber.Ctx, err error) error {
        var appErr *apierrors.AppError
        var statusCode int = http.StatusInternalServerError
        var errCode string = apierrors.ErrCodeUnknown
        var message string = "An unexpected error occurred. Please try again later."
        var requestID string = c.Locals("requestID").(string)

        if errors.As(err, &appErr) {
            // Handle our custom AppError
            errCode = appErr.Code
            message = appErr.Message
            
            // Ensure RequestID is set
            if appErr.RequestID == "" {
                appErr.RequestID = requestID
            }

            // Map AppError Code to HTTP Status Code based on category and code
            if appErr.Category == apierrors.CategoryBusiness {
                switch appErr.Code {
                case apierrors.ErrCodeProductNotFound:
                    statusCode = http.StatusNotFound
                case apierrors.ErrCodeInsufficientStock, 
                     apierrors.ErrCodeInvalidProductData,
                     apierrors.ErrCodeOrderLimitExceeded, 
                     apierrors.ErrCodePriceMismatch:
                    statusCode = http.StatusBadRequest
                default:
                    statusCode = http.StatusBadRequest
                }
            } else {
                // Application category
                switch appErr.Code {
                case apierrors.ErrCodeDatabaseAccess, 
                     apierrors.ErrCodeInternalProcessing,
                     apierrors.ErrCodeSystemPanic:
                    statusCode = http.StatusInternalServerError
                case apierrors.ErrCodeServiceUnavailable,
                     apierrors.ErrCodeNetworkError:
                    statusCode = http.StatusServiceUnavailable
                case apierrors.ErrCodeRequestValidation,
                     apierrors.ErrCodeMalformedData:
                    statusCode = http.StatusBadRequest
                case apierrors.ErrCodeResourceConstraint:
                    statusCode = http.StatusTooManyRequests
                case apierrors.ErrCodeRequestTimeout:
                    statusCode = http.StatusRequestTimeout
                default:
                    statusCode = http.StatusInternalServerError
                }
            }
            
            // Log with appropriate level based on category and status code
            if appErr.Category == apierrors.CategoryBusiness && statusCode < 500 {
                logger.WarnContext(c.UserContext(), "Business rule violation",
                    slog.String("error_code", appErr.Code),
                    slog.String("message", appErr.Message),
                    slog.String("request_id", appErr.RequestID),
                    slog.String("path", c.Path()),
                )
            } else {
                logger.ErrorContext(c.UserContext(), "Error occurred",
                    slog.String("error_code", appErr.Code),
                    slog.String("category", string(appErr.Category)),
                    slog.String("message", appErr.Message),
                    slog.Any("cause", appErr.Unwrap()),
                    slog.String("request_id", appErr.RequestID),
                    slog.String("path", c.Path()),
                )
            }
        } else {
            // Handle unexpected errors with better classification
            var netErr net.Error
            var jsonErr *json.SyntaxError
            var timeoutErr context.DeadlineExceeded
            var canceledErr context.Canceled
            
            switch {
            case errors.As(err, &netErr):
                errCode = apierrors.ErrCodeNetworkError
                statusCode = http.StatusServiceUnavailable
                message = "Network connectivity issue occurred"
                
            case errors.As(err, &jsonErr):
                errCode = apierrors.ErrCodeMalformedData
                statusCode = http.StatusBadRequest
                message = "Invalid data format in request"
                
            case errors.Is(err, timeoutErr):
                errCode = apierrors.ErrCodeRequestTimeout
                statusCode = http.StatusRequestTimeout
                message = "Request processing timed out"
                
            case errors.Is(err, canceledErr):
                errCode = apierrors.ErrCodeRequestTimeout
                statusCode = http.StatusRequestTimeout
                message = "Request was canceled"
                
            default:
                errCode = apierrors.ErrCodeUnknown
                statusCode = http.StatusInternalServerError
                message = "An unexpected error occurred"
            }
            
            logger.ErrorContext(c.UserContext(), "Unhandled error",
                slog.String("error_type", fmt.Sprintf("%T", err)),
                slog.String("error", err.Error()),
                slog.String("error_code", errCode),
                slog.String("request_id", requestID),
                slog.String("path", c.Path()),
            )
        }

        // Send standardized JSON error response
        c.Status(statusCode)
        return c.JSON(apiresponses.ErrorResponse{
            Status: "error",
            Error: apiresponses.ErrorDetail{
                Code:      errCode,
                Message:   message,
                RequestID: requestID,
                Timestamp: time.Now().UTC().Format(time.RFC3339),
            },
        })
    }
}
```

#### Step 4: Update Main Application to Use Middleware

**Where**: Update `product-service/src/main.go`  
**Why**: To integrate the new middleware into the request pipeline  
**How**: Add middleware registration in the Fiber app setup  
**When**: After middleware implementation  

**Before**:
```go
// --- Middleware Configuration ---
app.Use(cors.New(cors.Config{
    AllowOrigins: "*",
    AllowHeaders: "Origin, Content-Type, Accept",
}))
app.Use(recover.New())          // Recover from panics
app.Use(otelfiber.Middleware()) // otelfiber instrumentation
```

**After**:
```go
// --- Middleware Configuration ---
app.Use(cors.New(cors.Config{
    AllowOrigins: "*",
    AllowHeaders: "Origin, Content-Type, Accept",
}))
app.Use(commonMiddleware.RequestIDMiddleware()) // Add request ID to all requests
app.Use(commonMiddleware.RecoverMiddleware())   // Custom panic recovery
app.Use(otelfiber.Middleware())                 // otelfiber instrumentation
```

### Phase 3: Update API Response Models

**Target: common/apiresponses package**

#### Step 1: Enhance Error Response Structure

**Where**: Update `common/apiresponses/responses.go`  
**Why**: To include request IDs and timestamps in responses  
**How**: Extend the error and success response structures  
**When**: After middleware enhancements  

**Before**:
```go
// ErrorDetail contains structured error information
type ErrorDetail struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

// ErrorResponse is the standard error response envelope
type ErrorResponse struct {
    Status string      `json:"status"`
    Error  ErrorDetail `json:"error"`
}

// SuccessResponse is the standard success response envelope
type SuccessResponse struct {
    Status string      `json:"status"`
    Data   interface{} `json:"data"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(data interface{}) SuccessResponse {
    return SuccessResponse{
        Status: "success",
        Data:   data,
    }
}
```

**After**:
```go
// ErrorDetail contains structured error information
type ErrorDetail struct {
    Code      string `json:"code"`
    Message   string `json:"message"`
    RequestID string `json:"requestId,omitempty"`
    Timestamp string `json:"timestamp,omitempty"`
}

// ErrorResponse is the standard error response envelope
type ErrorResponse struct {
    Status string      `json:"status"`
    Error  ErrorDetail `json:"error"`
}

// SuccessResponse is the standard success response envelope
type SuccessResponse struct {
    Status    string      `json:"status"`
    Data      interface{} `json:"data"`
    RequestID string      `json:"requestId,omitempty"`
    Timestamp string      `json:"timestamp,omitempty"`
}

// NewSuccessResponse creates a new success response
func NewSuccessResponse(data interface{}) SuccessResponse {
    return SuccessResponse{
        Status:    "success",
        Data:      data,
        Timestamp: time.Now().UTC().Format(time.RFC3339),
    }
}

// WithRequestID adds a request ID to the success response
func (r SuccessResponse) WithRequestID(requestID string) SuccessResponse {
    r.RequestID = requestID
    return r
}
```

### Phase 4: Service Layer Refactoring

**Target: product-service/src/services package**

#### Step 1: Update Service Error Handling

**Where**: All service files, starting with `buy_product_service.go`  
**Why**: To use the new error categorization and remove narrative styles  
**How**: Replace error creation with new methods and update logging  
**When**: After error classification and middleware updates  

**Before** (`buy_product_service.go` excerpt):
```go
if product.Stock < quantity {
    errMsg := fmt.Sprintf("Insufficient stock for product '%s'. Available: %d, Requested: %d", name, product.Stock, quantity)
    s.logger.WarnContext(ctx, "Shop Manager: Purchase blocked - insufficient stock",
        slog.String("product_name", name),
        slog.Int("requested", quantity),
        slog.Int("available", product.Stock),
    )
    if span != nil {
        span.SetStatus(codes.Error, "Insufficient stock") // Specific message for span
    }
    appErr = apierrors.NewAppError(apierrors.ErrCodeInsufficientStock, errMsg, nil)
    // Track error metrics
    metric.IncrementErrorCount(ctx, apierrors.ErrCodeInsufficientStock, "buy_product", "service")
    return 0, appErr // Return zero revenue with the error
}
s.logger.DebugContext(ctx, "Shop Manager: Stock available for purchase")
```

**After**:
```go
if product.Stock < quantity {
    errMsg := fmt.Sprintf("Insufficient stock for product '%s'. Available: %d, Requested: %d", name, product.Stock, quantity)
    
    // Get request ID from context
    var requestID string
    if id, ok := ctx.Value("requestID").(string); ok {
        requestID = id
    }
    
    s.logger.WarnContext(ctx, "Purchase rejected: insufficient stock",
        slog.String("product_name", name),
        slog.Int("requested", quantity),
        slog.Int("available", product.Stock),
        slog.String("error_code", apierrors.ErrCodeInsufficientStock),
        slog.String("request_id", requestID),
        slog.String("event_type", "purchase_rejected"),
    )
    
    if span != nil {
        span.SetStatus(codes.Error, "Insufficient stock")
    }
    
    // Create business error with request ID
    appErr = apierrors.NewBusinessError(
        apierrors.ErrCodeInsufficientStock,
        errMsg,
        nil,
    ).WithRequestID(requestID)
    
    // Track error metrics
    metric.IncrementErrorCount(ctx, apierrors.ErrCodeInsufficientStock, "buy_product", "service")
    return 0, appErr
}
s.logger.DebugContext(ctx, "Stock verification completed: sufficient stock available",
    slog.String("product_name", name),
    slog.Int("available", product.Stock),
    slog.Int("requested", quantity),
    slog.String("event_type", "stock_verified"))
```

### Phase 5: Repository Layer Refactoring

**Target: product-service/src/repositories package**

#### Step 1: Update Repository Error Handling

**Where**: All repository files  
**Why**: To use new error categorization and improve error context  
**How**: Update error creation with new methods and better context  
**When**: After service layer updates  

**Before** (example from repository):
```go
product, exists := r.database.GetProduct(name)
if !exists {
    r.logger.WarnContext(ctx, "Stock room worker: Product not found", slog.String("product_name", name))
    return models.Product{}, apierrors.NewAppError(
        apierrors.ErrCodeNotFound,
        fmt.Sprintf("Product '%s' not found", name),
        nil,
    )
}
```

**After**:
```go
product, exists := r.database.GetProduct(name)
if !exists {
    // Get request ID from context
    var requestID string
    if id, ok := ctx.Value("requestID").(string); ok {
        requestID = id
    }
    
    r.logger.WarnContext(ctx, "Product not found in database",
        slog.String("product_name", name),
        slog.String("error_code", apierrors.ErrCodeProductNotFound),
        slog.String("request_id", requestID),
        slog.String("operation", "get_by_name"),
    )
    
    return models.Product{}, apierrors.NewBusinessError(
        apierrors.ErrCodeProductNotFound,
        fmt.Sprintf("Product '%s' not found", name),
        nil,
    ).WithRequestID(requestID).WithContext("operation", "get_by_name")
}
```

### Phase 6: Handler Layer Refactoring

**Target: product-service/src/handlers package**

#### Step 1: Update Handler Success Responses

**Where**: All handler files  
**Why**: To include request IDs in successful responses  
**How**: Update response creation with request IDs  
**When**: After repository layer updates  

**Before** (`buy_product_handler.go` excerpt):
```go
err = c.Status(http.StatusOK).JSON(apiresponses.NewSuccessResponse(fiber.Map{
    "productName": productName,
    "quantity":    quantity,
    "revenue":     revenue,
}))
```

**After**:
```go
// Get request ID
requestID := c.Locals("requestID").(string)

// Create response with request ID
response := apiresponses.NewSuccessResponse(fiber.Map{
    "productName": productName,
    "quantity":    quantity,
    "revenue":     revenue,
}).WithRequestID(requestID)

err = c.Status(http.StatusOK).JSON(response)
```

#### Step 2: Update Handler Error Creation

**Where**: All handler files  
**Why**: To use new error types and include request IDs  
**How**: Replace error creation with new methods  
**When**: After updating success responses  

**Before**:
```go
if parseErr := c.BodyParser(&req); parseErr != nil {
    h.logger.ErrorContext(ctx, "Front Desk: Invalid purchase request format", slog.String("error", parseErr.Error()))
    err = apierrors.NewAppError(apierrors.ErrCodeValidation, "Invalid request body format", parseErr)
    return
}
```

**After**:
```go
if parseErr := c.BodyParser(&req); parseErr != nil {
    requestID := c.Locals("requestID").(string)
    
    h.logger.WarnContext(ctx, "Request rejected: invalid request format",
        slog.String("error", parseErr.Error()),
        slog.String("error_code", apierrors.ErrCodeRequestValidation),
        slog.String("request_id", requestID),
        slog.String("path", c.Path()),
    )
    
    err = apierrors.NewApplicationError(
        apierrors.ErrCodeRequestValidation, 
        "Invalid request body format", 
        parseErr,
    ).WithRequestID(requestID)
    
    return
}
```

#### Step 3: Update Logging in Handlers

**Where**: All handler files  
**Why**: To replace narrative logs with structured business events  
**How**: Update log messages with event-focused approach  
**When**: After error handling updates  

**Before**:
```go
h.logger.InfoContext(ctx, "Front_Desk: Customer wants to buy a product")
// ...
h.logger.InfoContext(ctx, "Front Desk: Asking shop manager to process purchase", 
    slog.String("product_name", productName), 
    slog.Int("quantity", quantity))
// ...
h.logger.InfoContext(ctx, "Front Desk: Purchase successful!",
    slog.String("product_name", productName),
    slog.Int("quantity_bought", quantity),
    slog.Float64("revenue", revenue),
)
```

**After**:
```go
requestID := c.Locals("requestID").(string)

h.logger.InfoContext(ctx, "Purchase request received",
    slog.String("request_id", requestID),
    slog.String("path", c.Path()),
    slog.String("method", c.Method()),
    slog.String("event_type", "purchase_initiated"))
// ...
h.logger.InfoContext(ctx, "Processing purchase request",
    slog.String("product_name", productName),
    slog.Int("quantity", quantity),
    slog.String("request_id", requestID),
    slog.String("event_type", "purchase_processing"))
// ...
h.logger.InfoContext(ctx, "Purchase completed successfully",
    slog.String("product_name", productName),
    slog.Int("quantity", quantity),
    slog.Float64("revenue", revenue),
    slog.String("request_id", requestID),
    slog.String("event_type", "purchase_completed"))
```

## Testing Strategy

### Phase 1: Unit Testing

1. **Error Type Testing**: Verify error categorization and context functions
2. **Middleware Testing**: Test each middleware component in isolation
3. **Error Handler Testing**: Validate HTTP status code mapping

### Phase 2: Integration Testing

1. **Error Flow Testing**: Verify end-to-end error handling
2. **Context Propagation**: Test request ID propagation across layers
3. **Recovery Testing**: Simulate panics to test recovery mechanism

### Phase 3: Performance Testing

1. **Measure Overhead**: Compare performance before and after changes
2. **Benchmark Recovery**: Test panic recovery performance

### Phase 4: API Backward Compatibility Testing

1. **Client Integration**: Ensure clients can handle new error formats
2. **Response Structure**: Verify responses maintain backward compatibility

## Deployment Plan

### Phase 1: Development Deployment

1. Implement changes in development environment
2. Run unit and integration tests
3. Perform manual testing with sample requests

### Phase 2: Staging Deployment

1. Deploy to staging environment
2. Run full test suite including performance tests
3. Verify with real-world test scenarios

### Phase 3: Production Deployment

1. Deploy to production during low-traffic period
2. Monitor error rates and performance metrics
3. Roll back plan if issues are detected

## Success Metrics

1. **Improved Error Clarity**: Measure reduction in time-to-resolution
2. **Better Observability**: Track correlation between logs and errors
3. **Enhanced Debugging**: Measure time spent debugging issues
4. **Error Coverage**: Percentage of errors properly categorized 