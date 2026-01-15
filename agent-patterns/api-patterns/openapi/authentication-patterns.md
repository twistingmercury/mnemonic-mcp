---
entity_name: OpenAPI Authentication Patterns
entity_type: OpenAPI Pattern
language: agnostic
domain: api-design
description: Comprehensive authentication and security patterns for OpenAPI specifications including JWT, OAuth2, and API keys
tags:
  - OpenAPI
  - authentication
  - security
  - JWT
  - OAuth2
  - API-keys
---

# OpenAPI Authentication Patterns

This pattern demonstrates various authentication schemes for RESTful APIs in OpenAPI 3.1 specifications.

## Multiple Authentication Schemes

```yaml
openapi: 3.1.0
info:
  title: Authentication Examples API
  version: 1.0.0
  description: API demonstrating multiple authentication patterns

components:
  securitySchemes:
    # JWT Bearer Token
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: |
        JWT token authentication. Include the token in the Authorization header:
        `Authorization: Bearer <token>`

    # API Key in Header
    apiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
      description: API key for service-to-service authentication

    # API Key in Query Parameter
    apiKeyQuery:
      type: apiKey
      in: query
      name: api_key
      description: API key as query parameter (less secure, use only for development)

    # OAuth 2.0 Authorization Code Flow
    oauth2AuthCode:
      type: oauth2
      description: OAuth 2.0 authorization code flow for web applications
      flows:
        authorizationCode:
          authorizationUrl: https://auth.example.com/oauth/authorize
          tokenUrl: https://auth.example.com/oauth/token
          refreshUrl: https://auth.example.com/oauth/refresh
          scopes:
            read: Read access to resources
            write: Write access to resources
            admin: Administrative access

    # OAuth 2.0 Client Credentials Flow
    oauth2ClientCredentials:
      type: oauth2
      description: OAuth 2.0 client credentials flow for machine-to-machine
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/oauth/token
          scopes:
            api:read: Read API resources
            api:write: Write API resources

    # OAuth 2.0 Password Flow (not recommended for new applications)
    oauth2Password:
      type: oauth2
      description: OAuth 2.0 resource owner password flow
      flows:
        password:
          tokenUrl: https://auth.example.com/oauth/token
          refreshUrl: https://auth.example.com/oauth/refresh
          scopes:
            read: Read access
            write: Write access

    # Basic Authentication (legacy)
    basicAuth:
      type: http
      scheme: basic
      description: Basic HTTP authentication (not recommended for production)

    # OpenID Connect
    openIdConnect:
      type: openIdConnect
      openIdConnectUrl: https://auth.example.com/.well-known/openid-configuration
      description: OpenID Connect authentication

paths:
  # Public endpoint - no authentication required
  /public/status:
    get:
      summary: Public health check
      description: Publicly accessible endpoint, no authentication required
      responses:
        '200':
          description: Service is healthy

  # JWT Bearer token authentication
  /users/me:
    get:
      summary: Get current user profile
      description: Requires JWT bearer token authentication
      security:
        - bearerAuth: []
      responses:
        '200':
          description: User profile
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/User'
        '401':
          $ref: '#/components/responses/Unauthorized'

  # API Key authentication
  /webhooks/events:
    post:
      summary: Webhook event handler
      description: Service-to-service endpoint using API key authentication
      security:
        - apiKeyAuth: []
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Event processed
        '401':
          $ref: '#/components/responses/Unauthorized'

  # Multiple authentication options (OR)
  /data/export:
    get:
      summary: Export data
      description: Accepts either Bearer token OR API key authentication
      security:
        - bearerAuth: []
        - apiKeyAuth: []
      responses:
        '200':
          description: Data export
        '401':
          $ref: '#/components/responses/Unauthorized'

  # Multiple authentication requirements (AND)
  /admin/config:
    put:
      summary: Update system configuration
      description: Requires BOTH bearer token AND API key (dual authentication)
      security:
        - bearerAuth: []
          apiKeyAuth: []
      responses:
        '200':
          description: Configuration updated
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'

  # OAuth 2.0 with scopes
  /resources:
    get:
      summary: List resources
      description: Requires OAuth 2.0 with 'read' scope
      security:
        - oauth2AuthCode: [read]
      responses:
        '200':
          description: List of resources
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'

    post:
      summary: Create resource
      description: Requires OAuth 2.0 with 'write' scope
      security:
        - oauth2AuthCode: [write]
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '201':
          description: Resource created
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'

  # OpenID Connect authentication
  /profile:
    get:
      summary: Get user profile
      description: Requires OpenID Connect authentication
      security:
        - openIdConnect: []
      responses:
        '200':
          description: User profile
        '401':
          $ref: '#/components/responses/Unauthorized'

  # Authentication endpoint (no security)
  /auth/login:
    post:
      summary: User login
      description: Authenticate user and receive JWT token
      security: []  # Explicitly no authentication required
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/LoginRequest'
      responses:
        '200':
          description: Login successful
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/LoginResponse'
        '401':
          $ref: '#/components/responses/Unauthorized'

  /auth/refresh:
    post:
      summary: Refresh access token
      description: Exchange refresh token for new access token
      security: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/RefreshRequest'
      responses:
        '200':
          description: Token refreshed
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/LoginResponse'
        '401':
          $ref: '#/components/responses/Unauthorized'

components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
        email:
          type: string
        name:
          type: string

    LoginRequest:
      type: object
      required:
        - email
        - password
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          format: password

    RefreshRequest:
      type: object
      required:
        - refresh_token
      properties:
        refresh_token:
          type: string

    LoginResponse:
      type: object
      required:
        - access_token
        - token_type
        - expires_in
      properties:
        access_token:
          type: string
          description: JWT access token
        refresh_token:
          type: string
          description: Refresh token for obtaining new access tokens
        token_type:
          type: string
          enum: [Bearer]
        expires_in:
          type: integer
          description: Token expiration time in seconds

    Error:
      type: object
      required:
        - error
        - message
      properties:
        error:
          type: string
        message:
          type: string

  responses:
    Unauthorized:
      description: Unauthorized - authentication required or invalid
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'

    Forbidden:
      description: Forbidden - authenticated but insufficient permissions
      content:
        application/json:
          schema:
            $ref: '#/components/schemas/Error'

# Global security requirement (applies to all endpoints unless overridden)
security:
  - bearerAuth: []
```

## Authentication Patterns

### JWT Bearer Token (Recommended)
**Best for:** User authentication in web and mobile apps
- Stateless authentication
- Short-lived access tokens (15-60 minutes)
- Long-lived refresh tokens for renewal
- Include user claims in token payload

```go
// Example JWT middleware in Go
func JWTAuthMiddleware(secret []byte) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "missing authorization header", http.StatusUnauthorized)
                return
            }

            tokenString := strings.TrimPrefix(authHeader, "Bearer ")
            token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
                return secret, nil
            })

            if err != nil || !token.Valid {
                http.Error(w, "invalid token", http.StatusUnauthorized)
                return
            }

            // Add claims to request context
            ctx := context.WithValue(r.Context(), "user", token.Claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### API Key Authentication
**Best for:** Service-to-service communication, webhooks
- Static keys for machine authentication
- Store keys in environment variables or secret management
- Use separate keys per environment
- Rotate keys periodically

```go
// Example API key middleware
func APIKeyAuthMiddleware(validKeys map[string]bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            apiKey := r.Header.Get("X-API-Key")
            if apiKey == "" || !validKeys[apiKey] {
                http.Error(w, "invalid API key", http.StatusUnauthorized)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

### OAuth 2.0 Flows

#### Authorization Code Flow
**Best for:** Web applications with backend
- Most secure flow for user authentication
- Access token never exposed to browser
- Supports refresh tokens

#### Client Credentials Flow
**Best for:** Machine-to-machine authentication
- No user involved
- Service authenticates with client ID and secret
- Short-lived access tokens

#### Password Flow (Not Recommended)
**Legacy only:** Direct username/password exchange
- Less secure than authorization code
- Use only for migrating legacy apps
- Prefer authorization code or client credentials

### OpenID Connect
**Best for:** Single Sign-On (SSO)
- Built on OAuth 2.0
- Provides user identity information
- Standard claims for user profile
- Supports multiple identity providers

## Security Best Practices

### Token Security
1. Use HTTPS only in production
2. Set appropriate token expiration times
3. Implement token refresh mechanism
4. Validate tokens on every request
5. Include audience (aud) and issuer (iss) claims

### API Key Security
1. Never commit keys to version control
2. Use environment-specific keys
3. Rotate keys regularly
4. Monitor for unauthorized usage
5. Use key prefixes for identification (e.g., `sk_prod_...`)

### Rate Limiting
Combine with authentication to prevent abuse:
```yaml
x-rate-limit:
  authenticated: 1000/hour
  unauthenticated: 100/hour
```

### CORS Configuration
Configure CORS headers properly:
```yaml
x-cors:
  allowed-origins:
    - https://app.example.com
  allowed-methods:
    - GET
    - POST
    - PUT
    - DELETE
  allowed-headers:
    - Authorization
    - Content-Type
  max-age: 3600
```

## Testing Authentication

### Test Bearer Token
```bash
# Obtain token
curl -X POST https://api.example.com/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'

# Use token
curl https://api.example.com/v1/users/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Test API Key
```bash
curl https://api.example.com/v1/data \
  -H "X-API-Key: sk_test_1234567890"
```

### Test OAuth 2.0 Client Credentials
```bash
# Get access token
curl -X POST https://auth.example.com/oauth/token \
  -u "client_id:client_secret" \
  -d "grant_type=client_credentials&scope=api:read"

# Use access token
curl https://api.example.com/v1/resources \
  -H "Authorization: Bearer <access_token>"
```
