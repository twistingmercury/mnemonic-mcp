---
entity_name: REST API Authentication Implementation (Go)
entity_type: backend-implementation
language: go
domain: backend
description: Go implementation of REST API authentication using Gin middleware for JWT, API keys, and OAuth2 with proper error handling and security
tags:
  - REST
  - authentication
  - Gin
  - gin-gonic
  - JWT
  - middleware
  - OAuth2
  - security
version: Go 1.21+
related_patterns:
  - REST API Authentication Patterns
  - REST API Implementation Pattern (Go)
---

# REST API Authentication Implementation (Go)

This pattern demonstrates implementing authentication in Go using Gin framework middleware. It covers JWT Bearer tokens, API keys, and OAuth2 integration.

## Prerequisites

```bash
go get -u github.com/gin-gonic/gin
go get -u github.com/golang-jwt/jwt/v5
go get -u golang.org/x/oauth2
```

## JWT Bearer Token Authentication

### JWT Middleware

```go
// internal/api/middleware/jwt_auth.go
package middleware

import (
    "fmt"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
)

// Claims represents JWT claims
type Claims struct {
    UserID uuid.UUID `json:"user_id"`
    Email  string    `json:"email"`
    Roles  []string  `json:"roles,omitempty"`
    jwt.RegisteredClaims
}

// JWTAuth middleware validates JWT tokens
func JWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            respondUnauthorized(c, "missing authorization header")
            c.Abort()
            return
        }

        // Extract Bearer token
        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenString == authHeader {
            respondUnauthorized(c, "invalid authorization header format")
            c.Abort()
            return
        }

        // Parse and validate token
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            // Verify signing method
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil {
            respondUnauthorized(c, "invalid or expired token")
            c.Abort()
            return
        }

        // Extract claims
        claims, ok := token.Claims.(*Claims)
        if !ok || !token.Valid {
            respondUnauthorized(c, "invalid token claims")
            c.Abort()
            return
        }

        // Store claims in context
        c.Set("user_id", claims.UserID)
        c.Set("email", claims.Email)
        c.Set("roles", claims.Roles)

        c.Next()
    }
}

// OptionalJWTAuth allows requests with or without JWT
func OptionalJWTAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.Next()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err == nil {
            if claims, ok := token.Claims.(*Claims); ok && token.Valid {
                c.Set("user_id", claims.UserID)
                c.Set("email", claims.Email)
                c.Set("roles", claims.Roles)
            }
        }

        c.Next()
    }
}

func respondUnauthorized(c *gin.Context, message string) {
    c.JSON(http.StatusUnauthorized, gin.H{
        "error":   "unauthorized",
        "message": message,
    })
}
```

### Token Generation

```go
// internal/auth/token.go
package auth

import (
    "os"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "yourapp/internal/api/middleware"
)

// TokenPair represents access and refresh tokens
type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    TokenType    string `json:"token_type"`
    ExpiresIn    int    `json:"expires_in"`
}

// GenerateTokenPair creates access and refresh tokens
func GenerateTokenPair(userID uuid.UUID, email string, roles []string) (*TokenPair, error) {
    // Access token (15 minutes)
    accessToken, err := generateToken(userID, email, roles, 15*time.Minute)
    if err != nil {
        return nil, err
    }

    // Refresh token (7 days)
    refreshToken, err := generateToken(userID, email, roles, 7*24*time.Hour)
    if err != nil {
        return nil, err
    }

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        TokenType:    "Bearer",
        ExpiresIn:    900, // 15 minutes in seconds
    }, nil
}

func generateToken(userID uuid.UUID, email string, roles []string, duration time.Duration) (string, error) {
    now := time.Now()
    claims := middleware.Claims{
        UserID: userID,
        Email:  email,
        Roles:  roles,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
            IssuedAt:  jwt.NewNumericDate(now),
            NotBefore: jwt.NewNumericDate(now),
            Issuer:    os.Getenv("JWT_ISSUER"),
            Subject:   userID.String(),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
```

### Login Handler

```go
// internal/api/handlers/auth_handler.go
package handlers

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "yourapp/internal/auth"
    "yourapp/internal/services"
)

type LoginRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required"`
}

func Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   "validation_error",
            "message": "Invalid request",
        })
        return
    }

    // Authenticate user
    user, err := userService.Authenticate(c.Request.Context(), req.Email, req.Password)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "Invalid credentials",
        })
        return
    }

    // Generate tokens
    tokens, err := auth.GenerateTokenPair(user.ID, user.Email, user.Roles)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{
            "error":   "internal_error",
            "message": "Failed to generate tokens",
        })
        return
    }

    c.JSON(http.StatusOK, tokens)
}
```

## API Key Authentication

### API Key Middleware

```go
// internal/api/middleware/api_key.go
package middleware

import (
    "net/http"
    "os"
    "strings"

    "github.com/gin-gonic/gin"
)

// APIKeyAuth middleware validates API keys
func APIKeyAuth() gin.HandlerFunc {
    validKeys := loadValidAPIKeys()

    return func(c *gin.Context) {
        apiKey := c.GetHeader("X-API-Key")
        if apiKey == "" {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Missing API key",
            })
            c.Abort()
            return
        }

        // Validate API key
        keyInfo, valid := validKeys[apiKey]
        if !valid {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "Invalid API key",
            })
            c.Abort()
            return
        }

        // Check if key is expired
        if keyInfo.IsExpired() {
            c.JSON(http.StatusUnauthorized, gin.H{
                "error":   "unauthorized",
                "message": "API key expired",
            })
            c.Abort()
            return
        }

        // Store key info in context
        c.Set("api_key_id", keyInfo.ID)
        c.Set("api_key_name", keyInfo.Name)
        c.Set("api_key_scopes", keyInfo.Scopes)

        c.Next()
    }
}

type APIKeyInfo struct {
    ID        string
    Name      string
    Scopes    []string
    ExpiresAt *time.Time
}

func (k *APIKeyInfo) IsExpired() bool {
    if k.ExpiresAt == nil {
        return false
    }
    return time.Now().After(*k.ExpiresAt)
}

func loadValidAPIKeys() map[string]*APIKeyInfo {
    // In production, load from database
    // For demo, load from environment
    keys := make(map[string]*APIKeyInfo)

    // Example: API_KEYS=key1:name1:scope1,scope2;key2:name2:scope3
    keysEnv := os.Getenv("API_KEYS")
    for _, keyDef := range strings.Split(keysEnv, ";") {
        parts := strings.Split(keyDef, ":")
        if len(parts) >= 2 {
            keys[parts[0]] = &APIKeyInfo{
                ID:     parts[0],
                Name:   parts[1],
                Scopes: strings.Split(parts[2], ","),
            }
        }
    }

    return keys
}
```

## Multiple Authentication Options (OR)

### Either JWT or API Key

```go
// internal/api/middleware/flexible_auth.go
package middleware

import (
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
)

// FlexibleAuth accepts either JWT or API key
func FlexibleAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Try JWT first
        authHeader := c.GetHeader("Authorization")
        if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
            // Attempt JWT authentication
            JWTAuth()(c)
            if !c.IsAborted() {
                return
            }
            c.Abort()
        }

        // Reset abort status for API key attempt
        c.Writer = &responseWriter{c.Writer, false}

        // Try API key
        apiKey := c.GetHeader("X-API-Key")
        if apiKey != "" {
            APIKeyAuth()(c)
            if !c.IsAborted() {
                return
            }
        }

        // Both failed
        c.JSON(http.StatusUnauthorized, gin.H{
            "error":   "unauthorized",
            "message": "Valid JWT token or API key required",
        })
        c.Abort()
    }
}
```

## Role-Based Authorization

### Role Check Middleware

```go
// internal/api/middleware/rbac.go
package middleware

import (
    "net/http"

    "github.com/gin-gonic/gin"
)

// RequireRole ensures user has required role
func RequireRole(requiredRoles ...string) gin.HandlerFunc {
    return func(c *gin.Context) {
        roles, exists := c.Get("roles")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "forbidden",
                "message": "No roles found in token",
            })
            c.Abort()
            return
        }

        userRoles, ok := roles.([]string)
        if !ok {
            c.JSON(http.StatusForbidden, gin.H{
                "error":   "forbidden",
                "message": "Invalid roles format",
            })
            c.Abort()
            return
        }

        // Check if user has any of the required roles
        for _, required := range requiredRoles {
            for _, userRole := range userRoles {
                if userRole == required {
                    c.Next()
                    return
                }
            }
        }

        c.JSON(http.StatusForbidden, gin.H{
            "error":   "forbidden",
            "message": "Insufficient permissions",
        })
        c.Abort()
    }
}

// Usage in router:
// adminRoutes.Use(middleware.JWTAuth(), middleware.RequireRole("admin"))
```

## Rate Limiting

### Rate Limiter Middleware

```go
// internal/api/middleware/rate_limit.go
package middleware

import (
    "net/http"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
)

type rateLimiter struct {
    mu       sync.Mutex
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
    rl := &rateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }

    // Cleanup old entries periodically
    go rl.cleanup()

    return rl
}

func (rl *rateLimiter) cleanup() {
    ticker := time.NewTicker(rl.window)
    defer ticker.Stop()

    for range ticker.C {
        rl.mu.Lock()
        now := time.Now()
        for key, times := range rl.requests {
            var valid []time.Time
            for _, t := range times {
                if now.Sub(t) < rl.window {
                    valid = append(valid, t)
                }
            }
            if len(valid) == 0 {
                delete(rl.requests, key)
            } else {
                rl.requests[key] = valid
            }
        }
        rl.mu.Unlock()
    }
}

func (rl *rateLimiter) Allow(key string) (bool, time.Duration) {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    times := rl.requests[key]

    // Remove old requests outside window
    var validTimes []time.Time
    for _, t := range times {
        if now.Sub(t) < rl.window {
            validTimes = append(validTimes, t)
        }
    }

    if len(validTimes) >= rl.limit {
        // Rate limit exceeded
        oldest := validTimes[0]
        retryAfter := rl.window - now.Sub(oldest)
        return false, retryAfter
    }

    // Add current request
    validTimes = append(validTimes, now)
    rl.requests[key] = validTimes

    return true, 0
}

// RateLimit middleware limits requests per identifier
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
    limiter := newRateLimiter(limit, window)

    return func(c *gin.Context) {
        // Use user ID if authenticated, otherwise use IP
        identifier := c.ClientIP()
        if userID, exists := c.Get("user_id"); exists {
            identifier = userID.(string)
        }

        allowed, retryAfter := limiter.Allow(identifier)
        if !allowed {
            c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
            c.Header("X-RateLimit-Remaining", "0")
            c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(retryAfter).Unix()))
            c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

            c.JSON(http.StatusTooManyRequests, gin.H{
                "error":       "rate_limit_exceeded",
                "message":     "Too many requests",
                "retry_after": int(retryAfter.Seconds()),
            })
            c.Abort()
            return
        }

        c.Next()
    }
}
```

## Router Configuration

### Complete Authentication Setup

```go
// internal/api/router/router.go
package router

import (
    "time"

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

    // Public endpoints (no auth)
    r.GET("/health", handlers.HealthCheck)
    r.POST("/auth/login", handlers.Login)
    r.POST("/auth/refresh", handlers.RefreshToken)

    // API v1 routes
    v1 := r.Group("/v1")
    {
        // User routes (JWT required)
        users := v1.Group("/users")
        users.Use(middleware.JWTAuth())
        users.Use(middleware.RateLimit(100, time.Hour))
        {
            users.GET("", handlers.ListUsers)
            users.POST("", handlers.CreateUser)
            users.GET("/:userId", handlers.GetUser)
            users.PUT("/:userId", handlers.UpdateUser)
            users.DELETE("/:userId", handlers.DeleteUser)
        }

        // Admin routes (JWT + admin role required)
        admin := v1.Group("/admin")
        admin.Use(middleware.JWTAuth())
        admin.Use(middleware.RequireRole("admin"))
        {
            admin.GET("/users", handlers.AdminListUsers)
            admin.PUT("/config", handlers.UpdateConfig)
        }

        // Webhook routes (API key required)
        webhooks := v1.Group("/webhooks")
        webhooks.Use(middleware.APIKeyAuth())
        {
            webhooks.POST("/events", handlers.HandleWebhook)
        }

        // Flexible auth routes (JWT or API key)
        data := v1.Group("/data")
        data.Use(middleware.FlexibleAuth())
        {
            data.GET("/export", handlers.ExportData)
        }
    }

    return r
}
```

## Testing Authentication

### Test Helpers

```go
// internal/api/handlers/auth_test.go
package handlers_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "yourapp/internal/auth"
)

func createTestJWT(t *testing.T, userID uuid.UUID, email string) string {
    tokens, err := auth.GenerateTokenPair(userID, email, []string{"user"})
    assert.NoError(t, err)
    return tokens.AccessToken
}

func TestJWTAuth_ValidToken(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := setupTestRouter()

    token := createTestJWT(t, uuid.New(), "test@example.com")

    req, _ := http.NewRequest("GET", "/v1/users/me", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_MissingToken(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := setupTestRouter()

    req, _ := http.NewRequest("GET", "/v1/users/me", nil)

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidToken(t *testing.T) {
    gin.SetMode(gin.TestMode)
    r := setupTestRouter()

    req, _ := http.NewRequest("GET", "/v1/users/me", nil)
    req.Header.Set("Authorization", "Bearer invalid-token")

    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusUnauthorized, w.Code)
}
```

## Security Best Practices

### 1. Token Storage

```go
// Store secrets in environment variables
jwtSecret := os.Getenv("JWT_SECRET")
if jwtSecret == "" {
    log.Fatal("JWT_SECRET environment variable not set")
}

// Never hardcode secrets
// BAD: secret := []byte("my-secret-key")
// GOOD: secret := []byte(os.Getenv("JWT_SECRET"))
```

### 2. Token Validation

```go
// Always validate all claims
claims, ok := token.Claims.(*Claims)
if !ok || !token.Valid {
    return errors.New("invalid token")
}

// Verify issuer and audience
if claims.Issuer != expectedIssuer {
    return errors.New("invalid issuer")
}

if claims.Audience != expectedAudience {
    return errors.New("invalid audience")
}
```

### 3. Error Messages

```go
// Don't leak information in error messages
// BAD: "User john@example.com not found"
// GOOD: "Invalid credentials"

// BAD: "Password incorrect for user ID 123"
// GOOD: "Invalid credentials"
```

### 4. Password Hashing

```go
import "golang.org/x/crypto/bcrypt"

// Hash password before storing
func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(hash), err
}

// Compare password with hash
func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

## Common Patterns

### Extract User from Context

```go
func GetCurrentUser(c *gin.Context) (uuid.UUID, error) {
    userID, exists := c.Get("user_id")
    if !exists {
        return uuid.Nil, errors.New("user not authenticated")
    }

    id, ok := userID.(uuid.UUID)
    if !ok {
        return uuid.Nil, errors.New("invalid user ID format")
    }

    return id, nil
}

// Usage in handler
func GetProfile(c *gin.Context) {
    userID, err := GetCurrentUser(c)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
        return
    }

    // Use userID...
}
```

### Scoped Permissions

```go
func RequireScope(requiredScope string) gin.HandlerFunc {
    return func(c *gin.Context) {
        scopes, exists := c.Get("api_key_scopes")
        if !exists {
            c.JSON(http.StatusForbidden, gin.H{
                "error": "forbidden",
                "message": "No scopes found",
            })
            c.Abort()
            return
        }

        scopeList := scopes.([]string)
        for _, scope := range scopeList {
            if scope == requiredScope || scope == "*" {
                c.Next()
                return
            }
        }

        c.JSON(http.StatusForbidden, gin.H{
            "error": "forbidden",
            "message": "Insufficient scope",
        })
        c.Abort()
    }
}
```
