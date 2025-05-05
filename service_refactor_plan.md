# ✨ Product Service Refactoring Plan: Request/Response/Error Handling ✨

---

## 🎯 Refactoring Summary

This plan details the comprehensive refactoring of the `product-service` API request handling, response generation, and error management. The goal is to establish a **consistent, robust, and maintainable** pattern for the entire request lifecycle. This involves:

1.  Implementing **Standardized Request Validation** using struct tags and a validation helper (✅ `requests.go`, `validation.go`).
2.  Implementing **Application-Specific Error Codes** using a custom `AppError` type (🚨 `errors.go`).
3.  Creating a **Centralized Fiber Error Middleware** to handle all errors, map them to appropriate HTTP status codes, and format them into a standard JSON error envelope (⚙️ `middleware.go`).
4.  Implementing **Standardized JSON Success Responses** using a consistent envelope (🎉 `responses.go`).

This holistic approach aims to improve API predictability, streamline development, enhance error reporting, and simplify client integration.

---

## ⚠️ Code Quality Issues Addressed

1.  **Inconsistent Error Handling (`handler.go`, `service.go`):** Scattered logic, manual `fiber.Error` creation, mixed error types.
2.  **Lack of Granular Error Information (`handler.go`, clients):** Generic `500` errors obscure root causes.
3.  **Mixing Concerns (`service.go`):** Service layer contains HTTP-specific `fiber.Error`.
4.  **Inconsistent Success Responses (`handler.go`):** Varying success data formats.
5.  **Ad-hoc Request Validation (`handler.go`):** Validation logic mixed directly within handlers, leading to potential duplication or omissions.

---

## 🛠️ Refactoring Plan Details

**1. Define Custom Error Type and Codes (📄 `product-service/src/errors.go` - New File)**

   *   **Where:** Create a new file `product-service/src/errors.go`.
   *   **What:** Define an `AppError` struct (holding `Code`, `Message`, `Err`) and constants for error codes (e.g., `ErrCodeValidation`, `ErrCodeNotFound`).
   *   **How:**
     ```go
     package main
     import "fmt"

     // Application-specific error codes
     const (
         ErrCodeUnknown          = "UNKNOWN_ERROR"
         ErrCodeNotFound         = "RESOURCE_NOT_FOUND"
         ErrCodeValidation       = "VALIDATION_ERROR"
         ErrCodeInsufficientStock = "INSUFFICIENT_STOCK"
         ErrCodeDatabase         = "DATABASE_ERROR"
         // Add more codes as needed ✍️
     )

     // AppError defines a standard application error.
     type AppError struct {
         Code    string
         Message string
         Err     error  // Underlying error
     }

     // Implement error interface and Unwrap
     func (e *AppError) Error() string { /* ... details ... */ }
     func (e *AppError) Unwrap() error { return e.Err }

     // Factory function
     func NewAppError(code, message string, cause error) *AppError { /* ... details ... */ }
     ```
   *   **Why ✅:** Establishes a standard internal error representation for specific handling and clear failure signaling.

**2. Define Standardized Response Structures (📄 `product-service/src/responses.go` - New File)**

   *   **Where:** Create `product-service/src/responses.go`.
   *   **What:** Define `SuccessResponse` (`{"status":"success", "data":...}`) and `ErrorResponse` (`{"status":"error", "error": {"code":..., "message":...}}`). Include helpers like `NewSuccessResponse`.
   *   **How:**
     ```go
     package main

     type SuccessResponse struct { Status string; Data interface{} }
     type ErrorResponse struct { Status string; Error ErrorDetail }
     type ErrorDetail struct { Code string; Message string }

     func NewSuccessResponse(data interface{}) SuccessResponse { /* ... */ }

     type ActionConfirmation struct { Message string } // Example optional data structure
     ```
   *   **Why ✅:** Ensures all API responses follow a predictable JSON structure, simplifying client integration.

**3. Define Request Payload Structures with Validation (📄 `product-service/src/requests.go` - New File)**

   *   **Where:** Create `product-service/src/requests.go`.
   *   **What:** Move existing payload structs here (e.g., `GetByNameRequest`). Add `validate` tags (from `go-playground/validator`) like `required`, `gte=0`, `gt=0`.
   *   **How:**
     ```go
     package main

     // Requires: go get github.com/go-playground/validator/v10 📦

     type GetByNameRequest struct { Name string `json:"name" validate:"required"` }
     type UpdateStockRequest struct { 
         Name string `json:"name" validate:"required"`
         Stock int `json:"stock" validate:"required,gte=0"` // >= 0
     }
     type ProductBuyRequest struct { 
         Name string `json:"name" validate:"required"`
         Quantity int `json:"quantity" validate:"required,gt=0"` // > 0
     }
     ```
   *   **Why ✅:** Centralizes request definitions and uses declarative validation for cleaner, more maintainable code.

**4. Implement Validation Helper (📄 `product-service/src/validation.go` - New File)**

   *   **Where:** Create `product-service/src/validation.go`.
   *   **What:** Create a singleton `validator.Validate` instance and `validateRequest(payload interface{}) *AppError` helper.
   *   **How:**
     ```go
     package main
     import (
         "fmt"
         "strings"
         "github.com/go-playground/validator/v10"
     )

     var validate = validator.New()

     func validateRequest(payload interface{}) *AppError {
         err := validate.Struct(payload)
         if err != nil {
             // Format validation errors into a user-friendly message
             var validationErrors []string
             for _, err := range err.(validator.ValidationErrors) {
                 validationErrors = append(validationErrors, fmt.Sprintf("Field '%s' failed validation on '%s'", err.Field(), err.Tag()))
             }
             errMsg := "Validation failed: " + strings.Join(validationErrors, ", ")
             return NewAppError(ErrCodeValidation, errMsg, err)
         }
         return nil // Success 🎉
     }
     ```
   *   **Why ✅:** Provides a reusable way to trigger validation and convert library errors into standard `AppError`s.

**5. Modify Repository to Return `*AppError` (🔄 `product-service/src/repository.go`)**

   *   **Where:** All methods in `repository.go` (`GetAll`, `GetByName`, `UpdateStock`, `GetByCategory`).
   *   **What:** Change return type from `error` to `*AppError`. Replace `fmt.Errorf` with `NewAppError` using codes like `ErrCodeNotFound`, `ErrCodeDatabase`.
   *   **How:** Find `return ..., fmt.Errorf(...)` -> Replace with `return ..., NewAppError(AppropriateCode, "Error message", originalError)`.
   *   **Why ✅:** Propagates structured, application-specific errors from the data layer.

**6. Modify Service to Use/Propagate `*AppError` (🔄 `product-service/src/service.go`)**

   *   **Where:** All methods in `service.go`.
   *   **What:** Change return types to `*AppError`. Remove `fiber.NewError`. Propagate repository `*AppError`s. Generate new `*AppError`s for business logic failures (e.g., `ErrCodeInsufficientStock`).
   *   **How:** Update signatures. Handle repository errors. Replace `fiber.NewError` with `NewAppError(AppropriateCode, ...)`.
   *   **Why ✅:** Ensures service layer uses application errors, removes HTTP dependencies, clearly signals business rule failures.

**7. Implement Centralized Error Handling Middleware (⚙️ `product-service/src/middleware.go` - New File)**

   *   **Where:** Create `product-service/src/middleware.go`.
   *   **What:** Implement a Fiber `ErrorHandler` (`CentralErrorHandler`). It catches errors, checks if `*AppError`, maps `appErr.Code` to HTTP status, logs, and sends `ErrorResponse` JSON.
   *   **How:**
     ```go
     package main
     import ( /* fiber, errors, net/http, slog, globals */ )

     func CentralErrorHandler() fiber.ErrorHandler {
         logger := globals.Logger()
         return func(c *fiber.Ctx, err error) error {
             var appErr *AppError
             statusCode := http.StatusInternalServerError
             errCode := ErrCodeUnknown
             message := "Unexpected error occurred. Please try again later."

             if errors.As(err, &appErr) {
                 errCode = appErr.Code
                 message = appErr.Message
                 switch appErr.Code {
                   case ErrCodeNotFound: statusCode = http.StatusNotFound // 404
                   case ErrCodeValidation, ErrCodeInsufficientStock: statusCode = http.StatusBadRequest // 400
                   case ErrCodeDatabase: statusCode = http.StatusInternalServerError // 500
                   // ... other mappings ...
                 }
                 logger.ErrorContext(c.UserContext(), "Application error", slog.String("code", errCode), slog.String("msg", message), slog.Any("cause", appErr.Unwrap()))
             } else {
                 // Handle non-AppError types
                 logger.ErrorContext(c.UserContext(), "Unhandled error", slog.String("type", fmt.Sprintf("%T", err)), slog.String("error", err.Error()))
             }

             c.Status(statusCode)
             return c.JSON(ErrorResponse{ Status: "error", Error: ErrorDetail{ Code: errCode, Message: message } })
         }
     }
     ```
   *   **Why ✅:** Centralizes error-to-HTTP mapping, ensures consistency, simplifies handlers.

**8. Refactor Handlers (🔄 `product-service/src/handler.go`)**

   *   **Where:** All handler functions in `handler.go`.
   *   **What:**
        *   **Request:** Parse body into request struct, call `validateRequest`, `return err` if validation fails. Handle query params manually, return `NewAppError(ErrCodeValidation, ...)` on failure.
        *   **Error:** Change signatures to `func(...) error`. If service calls return `err`, `return err`.
        *   **Success:** On success, `return c.Status(http.StatusOK).JSON(NewSuccessResponse(data))`.
   *   **How (Example - GetProductByName):**
     ```go
     func (h *ProductHandler) GetProductByName(c *fiber.Ctx) error { // Signature change ✨
         // ... parse request into 'req' ...
         if err := c.BodyParser(&req); err != nil { return NewAppError(ErrCodeValidation, ...) } 
         if validationErr := validateRequest(&req); validationErr != nil { return validationErr } // Validate!

         product, err := h.service.GetByName(ctx, req.Name) // Call service
         if err != nil { return err } // Propagate!

         return c.Status(http.StatusOK).JSON(NewSuccessResponse(product)) // Success! 🎉
     }
     ```
   *   **Why ✅:** Simplifies handler logic by delegating validation, error handling, and response formatting.

**9. Register Error Middleware (🔄 `product-service/src/main.go`)**

   *   **Where:** The `main` function in `main.go`.
   *   **What:** Configure the `fiber.New` instance with `ErrorHandler: CentralErrorHandler()`.
   *   **How:**
     ```go
     app := fiber.New(fiber.Config{
         ErrorHandler: CentralErrorHandler(), // Register middleware ✅
     })
     ```
   *   **Why ✅:** Activates the centralized error handling for the application.

---

## 🚀 Implementation Sequence

1.  `go get github.com/go-playground/validator/v10` 📦
2.  Create `errors.go` 📄
3.  Create `responses.go` 📄
4.  Create `requests.go` 📄
5.  Create `validation.go` 📄
6.  Refactor `repository.go` methods 🔄
7.  Refactor `service.go` methods 🔄
8.  Create `middleware.go` 📄⚙️
9.  **Refactor all handlers** in `handler.go` 🔄✨
10. Update `main.go` to register middleware 🔄⚙️

---

##🧪 Testing Strategy

*   **Unit Tests:** Cover `AppError`, `validateRequest`, repository/service error logic.
*   **Integration/API Tests:** Verify all endpoints return correct HTTP status codes and standard JSON structures (`SuccessResponse` / `ErrorResponse`) for various valid and invalid scenarios (including validation failures).

---

## ✅ Expected Outcomes

*   ✅ Consistent request validation.
*   ✅ Consistent JSON responses (success and error).
*   ✅ Clear error codes for API clients.
*   ✅ Simplified handler logic.
*   ✅ Improved separation of concerns.
*   ✅ Centralized, maintainable error mapping, validation, and response formatting. 