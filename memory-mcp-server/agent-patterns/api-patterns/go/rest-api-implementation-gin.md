---
entity_name: REST API Implementation Pattern (Go)
entity_type: backend-implementation
language: go
domain: backend
description: Go implementation of RESTful API using Gin framework with middleware, error handling, validation, and OpenAPI integration
tags:
  - REST
  - Gin
  - gin-gonic
  - middleware
  - error-handling
  - OpenAPI
  - validation
version: Go 1.21+
related_patterns:
  - REST API Specification Pattern
  - REST API Testing Pattern (Go)
  - REST API Authentication Patterns
---

# REST API Implementation Pattern (Go)

This pattern demonstrates implementing a RESTful API in Go using the Gin framework (github.com/gin-gonic/gin). It shows how to implement an OpenAPI specification with proper middleware, error handling, and validation.

## Prerequisites

```bash
go get -u github.com/gin-gonic/gin
go get -u github.com/go-playground/validator/v10
go get -u github.com/golang-jwt/jwt/v5
```

## Project Structure

```
project/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── api/
│   │   ├── handlers/            # HTTP handlers
│   │   │   ├── user_handler.go
│   │   │   └── error_handler.go
│   │   ├── middleware/          # Gin middleware
│   │   │   ├── auth.go
│   │   │   ├── cors.go
│   │   │   └── logging.go
│   │   └── router/              # Route setup
│   │       └── router.go
│   ├── models/                  # Domain models
│   │   └── user.go
│   └── services/                # Business logic
│       └── user_service.go
└── openapi.yaml                 # API specification
```

## Main Server Setup

```go
// cmd/server/main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gin-gonic/gin"
    "yourapp/internal/api/router"
)

func main() {
    // Set Gin mode
    if os.Getenv("ENV") == "production" {
        gin.SetMode(gin.ReleaseMode)
    }

    // Initialize router
    r := router.Setup()

    // Create HTTP server
    srv := &http.Server{
        Addr:           ":8080",
        Handler:        r,
        ReadTimeout:    10 * time.Second,
        WriteTimeout:   10 * time.Second,
        MaxHeaderBytes: 1 << 20, // 1 MB
    }

    // Start server in goroutine
    go func() {
        log.Printf("Starting server on %s", srv.Addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server error: %v", err)
        }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server stopped")
}
```

## Router Setup

```go
// internal/api/router/router.go
package router

import (
    "github.com/gin-gonic/gin"
    "yourapp/internal/api/handlers"
    "yourapp/internal/api/middleware"
)

func Setup() *gin.Engine {
    r := gin.New()

    // Global middleware
    r.Use(gin.Recovery())
    r.Use(middleware.Logger())
    r.Use(middleware.CORS())

    // Health check endpoint (no auth required)
    r.GET("/health", func(c *gin.Context) {
        c.JSON(200, gin.H{
            "status": "healthy",
            "timestamp": time.Now().UTC(),
        })
    })

    // API v1 routes
    v1 := r.Group("/v1")
    {
        // User routes (require authentication)
        users := v1.Group("/users")
        users.Use(middleware.JWTAuth())
        {
            users.GET("", handlers.ListUsers)
            users.POST("", handlers.CreateUser)
            users.GET("/:userId", handlers.GetUser)
            users.PUT("/:userId", handlers.UpdateUser)
            users.PATCH("/:userId", handlers.PatchUser)
            users.DELETE("/:userId", handlers.DeleteUser)
        }
    }

    return r
}
```

## Models

```go
// internal/models/user.go
package models

import (
    "time"
    "github.com/google/uuid"
)

type UserStatus string

const (
    UserStatusActive    UserStatus = "active"
    UserStatusInactive  UserStatus = "inactive"
    UserStatusSuspended UserStatus = "suspended"
)

type User struct {
    ID        uuid.UUID  `json:"id"`
    Email     string     `json:"email" binding:"required,email"`
    Name      string     `json:"name" binding:"required,min=1,max=100"`
    Status    UserStatus `json:"status" binding:"omitempty,oneof=active inactive suspended"`
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
}

type CreateUserRequest struct {
    Email string `json:"email" binding:"required,email"`
    Name  string `json:"name" binding:"required,min=1,max=100"`
}

type UpdateUserRequest struct {
    Email  string     `json:"email" binding:"required,email"`
    Name   string     `json:"name" binding:"required,min=1,max=100"`
    Status UserStatus `json:"status" binding:"required,oneof=active inactive suspended"`
}

type PatchUserRequest struct {
    Email  *string     `json:"email" binding:"omitempty,email"`
    Name   *string     `json:"name" binding:"omitempty,min=1,max=100"`
    Status *UserStatus `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}

type UserListResponse struct {
    Data       []User         `json:"data"`
    Pagination PaginationInfo `json:"pagination"`
}

type PaginationInfo struct {
    Page       int `json:"page"`
    PageSize   int `json:"page_size"`
    TotalItems int `json:"total_items"`
    TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
    Error   string        `json:"error"`
    Message string        `json:"message"`
    Details []ErrorDetail `json:"details,omitempty"`
}

type ErrorDetail struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}
```

## Handlers

```go
// internal/api/handlers/user_handler.go
package handlers

import (
    "net/http"
    "strconv"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "yourapp/internal/models"
    "yourapp/internal/services"
)

type UserHandler struct {
    userService *services.UserService
}

func NewUserHandler(userService *services.UserService) *UserHandler {
    return &UserHandler{userService: userService}
}

// ListUsers godoc
// @Summary List users
// @Description Retrieve a paginated list of users with optional filtering
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param email query string false "Filter by email"
// @Param status query string false "Filter by status"
// @Success 200 {object} models.UserListResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /v1/users [get]
func ListUsers(c *gin.Context) {
    // Parse pagination parameters
    page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
    pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

    // Validate pagination
    if page < 1 {
        page = 1
    }
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }

    // Parse filters
    email := c.Query("email")
    status := c.Query("status")

    // Get users from service
    users, total, err := userService.ListUsers(c.Request.Context(), page, pageSize, email, status)
    if err != nil {
        RespondWithError(c, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }

    // Calculate pagination
    totalPages := (total + pageSize - 1) / pageSize

    response := models.UserListResponse{
        Data: users,
        Pagination: models.PaginationInfo{
            Page:       page,
            PageSize:   pageSize,
            TotalItems: total,
            TotalPages: totalPages,
        },
    }

    c.JSON(http.StatusOK, response)
}

// CreateUser godoc
// @Summary Create user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.CreateUserRequest true "User data"
// @Success 201 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /v1/users [post]
func CreateUser(c *gin.Context) {
    var req models.CreateUserRequest

    if err := c.ShouldBindJSON(&req); err != nil {
        RespondWithValidationError(c, err)
        return
    }

    user, err := userService.CreateUser(c.Request.Context(), req)
    if err != nil {
        // Check for duplicate email
        if errors.Is(err, services.ErrUserExists) {
            RespondWithError(c, http.StatusConflict, "user_exists", "User with this email already exists")
            return
        }
        RespondWithError(c, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }

    // Set Location header
    c.Header("Location", fmt.Sprintf("/v1/users/%s", user.ID))
    c.JSON(http.StatusCreated, user)
}

// GetUser godoc
// @Summary Get user
// @Description Retrieve a specific user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /v1/users/{userId} [get]
func GetUser(c *gin.Context) {
    userID, err := uuid.Parse(c.Param("userId"))
    if err != nil {
        RespondWithError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID format")
        return
    }

    user, err := userService.GetUser(c.Request.Context(), userID)
    if err != nil {
        if errors.Is(err, services.ErrUserNotFound) {
            RespondWithError(c, http.StatusNotFound, "user_not_found", "User not found")
            return
        }
        RespondWithError(c, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }

    c.JSON(http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Update user
// @Description Update an existing user (full update)
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Param user body models.UpdateUserRequest true "User data"
// @Success 200 {object} models.User
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /v1/users/{userId} [put]
func UpdateUser(c *gin.Context) {
    userID, err := uuid.Parse(c.Param("userId"))
    if err != nil {
        RespondWithError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID format")
        return
    }

    var req models.UpdateUserRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        RespondWithValidationError(c, err)
        return
    }

    user, err := userService.UpdateUser(c.Request.Context(), userID, req)
    if err != nil {
        if errors.Is(err, services.ErrUserNotFound) {
            RespondWithError(c, http.StatusNotFound, "user_not_found", "User not found")
            return
        }
        RespondWithError(c, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }

    c.JSON(http.StatusOK, user)
}

// DeleteUser godoc
// @Summary Delete user
// @Description Delete a user account
// @Tags users
// @Accept json
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Success 204 "No Content"
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Security BearerAuth
// @Router /v1/users/{userId} [delete]
func DeleteUser(c *gin.Context) {
    userID, err := uuid.Parse(c.Param("userId"))
    if err != nil {
        RespondWithError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID format")
        return
    }

    err = userService.DeleteUser(c.Request.Context(), userID)
    if err != nil {
        if errors.Is(err, services.ErrUserNotFound) {
            RespondWithError(c, http.StatusNotFound, "user_not_found", "User not found")
            return
        }
        RespondWithError(c, http.StatusInternalServerError, "internal_error", err.Error())
        return
    }

    c.Status(http.StatusNoContent)
}
```

## Error Handling

```go
// internal/api/handlers/error_handler.go
package handlers

import (
    "github.com/gin-gonic/gin"
    "github.com/go-playground/validator/v10"
    "yourapp/internal/models"
)

func RespondWithError(c *gin.Context, statusCode int, errorCode, message string) {
    c.JSON(statusCode, models.ErrorResponse{
        Error:   errorCode,
        Message: message,
    })
}

func RespondWithValidationError(c *gin.Context, err error) {
    var details []models.ErrorDetail

    if validationErrs, ok := err.(validator.ValidationErrors); ok {
        for _, e := range validationErrs {
            details = append(details, models.ErrorDetail{
                Field:   e.Field(),
                Message: getValidationMessage(e),
            })
        }
    }

    c.JSON(http.StatusBadRequest, models.ErrorResponse{
        Error:   "validation_error",
        Message: "Invalid input data",
        Details: details,
    })
}

func getValidationMessage(e validator.FieldError) string {
    switch e.Tag() {
    case "required":
        return "This field is required"
    case "email":
        return "Invalid email format"
    case "min":
        return fmt.Sprintf("Minimum length is %s", e.Param())
    case "max":
        return fmt.Sprintf("Maximum length is %s", e.Param())
    case "oneof":
        return fmt.Sprintf("Must be one of: %s", e.Param())
    default:
        return "Invalid value"
    }
}
```

## Middleware

```go
// internal/api/middleware/auth.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Missing authorization header",
            })
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Invalid authorization header format",
            })
            c.Abort()
            return
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            // Verify signing method
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method")
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Invalid or expired token",
            })
            c.Abort()
            return
        }

        // Store claims in context
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Set("user_id", claims["user_id"])
            c.Set("email", claims["email"])
        }

        c.Next()
    }
}
```

```go
// internal/api/middleware/logging.go
package middleware

import (
    "log"
    "time"

    "github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        path := c.Request.URL.Path
        raw := c.Request.URL.RawQuery

        c.Next()

        latency := time.Since(start)
        statusCode := c.Writer.Status()
        clientIP := c.ClientIP()
        method := c.Request.Method

        if raw != "" {
            path = path + "?" + raw
        }

        log.Printf("%s | %3d | %13v | %15s | %-7s %s",
            time.Now().Format("2006/01/02 - 15:04:05"),
            statusCode,
            latency,
            clientIP,
            method,
            path,
        )
    }
}
```

```go
// internal/api/middleware/cors.go
package middleware

import (
    "github.com/gin-gonic/gin"
)

func CORS() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
        c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
        c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Authorization, X-API-Key")
        c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

        if c.Request.Method == "OPTIONS" {
            c.AbortWithStatus(204)
            return
        }

        c.Next()
    }
}
```

## Key Implementation Patterns

### Gin Router Best Practices

1. **Use route groups** for versioning and shared middleware
2. **Bind JSON with validation** using struct tags
3. **Return consistent error responses** across all handlers
4. **Use HTTP status codes correctly** (201 for creation, 204 for deletion)
5. **Set Location header** for created resources

### Error Handling

1. **Map validation errors** to user-friendly messages
2. **Distinguish error types** (not found, conflict, validation, internal)
3. **Use custom error types** in service layer
4. **Never expose internal errors** to clients

### Middleware Order

1. Recovery (catch panics)
2. Logging (track all requests)
3. CORS (handle preflight)
4. Authentication (validate tokens)
5. Application handlers

### Security

1. **Use HTTPS** in production
2. **Validate all inputs** with struct tags
3. **Sanitize error messages** (no stack traces)
4. **Set security headers** (CORS, CSP)
5. **Rate limit** API endpoints

## Testing

See "REST API Testing Pattern (Go)" for comprehensive testing examples using Gin's test mode and httptest package.
