---
entity_name: REST API Authentication Patterns
entity_type: api-specification
language: agnostic
domain: api-design
description: Comprehensive authentication and security patterns for OpenAPI specifications including JWT, OAuth2, API keys, and OpenID Connect
tags:
  - REST
  - OpenAPI
  - authentication
  - security
  - JWT
  - OAuth2
  - API-keys
version: OpenAPI 3.1
related_patterns:
  - REST API Specification Pattern
  - REST API Authentication Implementation (Go)
---

# REST API Authentication Patterns

This pattern demonstrates various authentication schemes for RESTful APIs in OpenAPI 3.1 specifications. These patterns are language-agnostic and can be implemented in any backend framework.

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

**Characteristics:**
- Stateless authentication
- Short-lived access tokens (15-60 minutes)
- Long-lived refresh tokens for renewal
- Include user claims in token payload

**Token Structure:**
```json
{
  "header": {
    "alg": "HS256",
    "typ": "JWT"
  },
  "payload": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "exp": 1735689600,
    "iat": 1735686000,
    "iss": "api.example.com",
    "aud": "api.example.com"
  }
}
```

**Required Claims:**
- `exp` - Expiration time
- `iat` - Issued at time
- `iss` - Issuer
- `aud` - Audience
- Custom claims (user_id, email, roles, etc.)

### API Key Authentication

**Best for:** Service-to-service communication, webhooks

**Characteristics:**
- Static keys for machine authentication
- Store keys in environment variables or secret management
- Use separate keys per environment
- Rotate keys periodically

**Key Format Examples:**
```
X-API-Key: sk_prod_1234567890abcdef
X-API-Key: pk_test_abcdef1234567890
```

**Best Practices:**
- Use prefixes to identify key type (sk_, pk_, etc.)
- Include environment in prefix (prod, test, dev)
- Never log API keys
- Support key rotation without downtime

### OAuth 2.0 Flows

#### Authorization Code Flow

**Best for:** Web applications with backend

**Characteristics:**
- Most secure flow for user authentication
- Access token never exposed to browser
- Supports refresh tokens
- PKCE extension for additional security

**Flow:**
1. Client redirects user to authorization server
2. User authenticates and grants permissions
3. Authorization server redirects back with code
4. Client exchanges code for access token
5. Client uses access token for API requests

#### Client Credentials Flow

**Best for:** Machine-to-machine authentication

**Characteristics:**
- No user involved
- Service authenticates with client ID and secret
- Short-lived access tokens
- Direct token request to token endpoint

**Use Cases:**
- Microservice communication
- Scheduled jobs
- Background processes
- CLI tools

#### Password Flow (Not Recommended)

**Legacy only:** Direct username/password exchange

**Why avoid:**
- Less secure than authorization code
- Client handles user credentials directly
- No consent screen
- Use only for migrating legacy apps

**Prefer:** Authorization code or client credentials

### OpenID Connect

**Best for:** Single Sign-On (SSO)

**Characteristics:**
- Built on OAuth 2.0
- Provides user identity information
- Standard claims for user profile
- Supports multiple identity providers

**Standard Claims:**
- `sub` - Subject (user ID)
- `name` - Full name
- `email` - Email address
- `picture` - Profile picture URL
- `email_verified` - Email verification status

## Security Best Practices

### Token Security

1. **Use HTTPS only** in production
2. **Set appropriate expiration times**
   - Access tokens: 15-60 minutes
   - Refresh tokens: 7-30 days
3. **Implement token refresh mechanism**
4. **Validate tokens on every request**
5. **Include required claims** (aud, iss, exp)
6. **Use strong signing algorithms** (RS256, ES256)
7. **Never store tokens in localStorage** (use httpOnly cookies)

### API Key Security

1. **Never commit keys to version control**
2. **Use environment-specific keys**
3. **Rotate keys regularly** (quarterly minimum)
4. **Monitor for unauthorized usage**
5. **Use key prefixes** for identification
6. **Implement rate limiting** per key
7. **Support key revocation**

### OAuth Security

1. **Use PKCE** for public clients
2. **Validate redirect URIs**
3. **Use state parameter** to prevent CSRF
4. **Validate scopes** on every request
5. **Implement proper consent screens**
6. **Use short-lived access tokens**
7. **Refresh tokens should be single-use**

### General Security

1. **Implement rate limiting**
   ```yaml
   x-rate-limit:
     authenticated: 1000/hour
     unauthenticated: 100/hour
   ```

2. **Configure CORS properly**
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

3. **Set security headers**
   - `Strict-Transport-Security`
   - `X-Content-Type-Options`
   - `X-Frame-Options`
   - `Content-Security-Policy`

4. **Log authentication events**
   - Login attempts
   - Token generation
   - Failed authentication
   - Suspicious activity

## Testing Authentication

### Test Bearer Token

```bash
# Obtain token
curl -X POST https://api.example.com/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"secret"}'

# Response
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}

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

# Response
{
  "access_token": "2YotnFZFEjr1zCsicMWpAA",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "api:read"
}

# Use access token
curl https://api.example.com/v1/resources \
  -H "Authorization: Bearer 2YotnFZFEjr1zCsicMWpAA"
```

## Common Patterns

### Public vs Protected Endpoints

```yaml
paths:
  /public/health:
    get:
      security: []  # No authentication

  /protected/data:
    get:
      security:
        - bearerAuth: []  # Authentication required
```

### Multiple Auth Options (OR)

```yaml
paths:
  /flexible/endpoint:
    get:
      security:
        - bearerAuth: []
        - apiKeyAuth: []
      # Either Bearer token OR API key works
```

### Multiple Auth Requirements (AND)

```yaml
paths:
  /highly-secure/endpoint:
    put:
      security:
        - bearerAuth: []
          apiKeyAuth: []
      # Both Bearer token AND API key required
```

### Scope-Based Authorization

```yaml
paths:
  /resources:
    get:
      security:
        - oauth2AuthCode: [read]  # Requires 'read' scope

    post:
      security:
        - oauth2AuthCode: [write]  # Requires 'write' scope

    delete:
      security:
        - oauth2AuthCode: [admin]  # Requires 'admin' scope
```

## Error Responses

### 401 Unauthorized

Missing or invalid authentication:
```json
{
  "error": "unauthorized",
  "message": "Missing or invalid authentication token"
}
```

### 403 Forbidden

Authenticated but insufficient permissions:
```json
{
  "error": "forbidden",
  "message": "Insufficient permissions to access this resource"
}
```

### 429 Too Many Requests

Rate limit exceeded:
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please try again later.",
  "retry_after": 3600
}
```
