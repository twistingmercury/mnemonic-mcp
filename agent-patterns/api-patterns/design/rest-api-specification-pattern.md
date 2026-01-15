---
entity_name: REST API Specification Pattern
entity_type: api-specification
language: agnostic
domain: api-design
description: Comprehensive OpenAPI 3.1 specification pattern for RESTful APIs with standard CRUD operations, pagination, filtering, and versioning
tags:
  - REST
  - OpenAPI
  - CRUD
  - pagination
  - versioning
  - API design
version: OpenAPI 3.1
related_patterns:
  - REST API Implementation Pattern (Go)
  - REST API Authentication Patterns
---

# REST API Specification Pattern

This pattern provides a complete OpenAPI 3.1 specification template for RESTful APIs. The specification is language-agnostic and can be implemented in any backend language or framework.

## Complete OpenAPI 3.1 Specification

```yaml
openapi: 3.1.0
info:
  title: Example API
  version: 1.0.0
  description: |
    Example RESTful API for managing resources.

    ## Features
    - CRUD operations on resources
    - Pagination and filtering
    - API versioning
    - Comprehensive error handling
  contact:
    name: API Support
    email: api-support@example.com
  license:
    name: MIT
    url: https://opensource.org/licenses/MIT

servers:
  - url: https://api.example.com/v1
    description: Production
  - url: https://api-staging.example.com/v1
    description: Staging
  - url: http://localhost:8080/v1
    description: Local development

tags:
  - name: users
    description: User management operations
  - name: items
    description: Item management operations

paths:
  /users:
    get:
      summary: List users
      description: Retrieve a paginated list of users with optional filtering
      operationId: listUsers
      tags:
        - users
      parameters:
        - $ref: "#/components/parameters/PageParam"
        - $ref: "#/components/parameters/PageSizeParam"
        - name: email
          in: query
          description: Filter by email address
          schema:
            type: string
            format: email
        - name: status
          in: query
          description: Filter by user status
          schema:
            type: string
            enum: [active, inactive, suspended]
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserListResponse"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "500":
          $ref: "#/components/responses/InternalServerError"
      security:
        - bearerAuth: []

    post:
      summary: Create user
      description: Create a new user account
      operationId: createUser
      tags:
        - users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateUserRequest"
      responses:
        "201":
          description: User created successfully
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
          headers:
            Location:
              description: URI of the created user
              schema:
                type: string
                format: uri
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "409":
          $ref: "#/components/responses/Conflict"
        "500":
          $ref: "#/components/responses/InternalServerError"
      security:
        - bearerAuth: []

  /users/{userId}:
    parameters:
      - $ref: "#/components/parameters/UserIdParam"

    get:
      summary: Get user
      description: Retrieve a specific user by ID
      operationId: getUser
      tags:
        - users
      responses:
        "200":
          description: Successful response
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "500":
          $ref: "#/components/responses/InternalServerError"
      security:
        - bearerAuth: []

    put:
      summary: Update user
      description: Update an existing user (full update)
      operationId: updateUser
      tags:
        - users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UpdateUserRequest"
      responses:
        "200":
          description: User updated successfully
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "500":
          $ref: "#/components/responses/InternalServerError"
      security:
        - bearerAuth: []

    patch:
      summary: Partially update user
      description: Partially update user fields
      operationId: patchUser
      tags:
        - users
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/PatchUserRequest"
      responses:
        "200":
          description: User updated successfully
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "400":
          $ref: "#/components/responses/BadRequest"
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "500":
          $ref: "#/components/responses/InternalServerError"
      security:
        - bearerAuth: []

    delete:
      summary: Delete user
      description: Delete a user account
      operationId: deleteUser
      tags:
        - users
      responses:
        "204":
          description: User deleted successfully
        "401":
          $ref: "#/components/responses/Unauthorized"
        "404":
          $ref: "#/components/responses/NotFound"
        "500":
          $ref: "#/components/responses/InternalServerError"
      security:
        - bearerAuth: []

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT token for authentication

  parameters:
    PageParam:
      name: page
      in: query
      description: Page number for pagination (1-indexed)
      schema:
        type: integer
        minimum: 1
        default: 1

    PageSizeParam:
      name: page_size
      in: query
      description: Number of items per page
      schema:
        type: integer
        minimum: 1
        maximum: 100
        default: 20

    UserIdParam:
      name: userId
      in: path
      required: true
      description: Unique user identifier
      schema:
        type: string
        format: uuid

  schemas:
    User:
      type: object
      required:
        - id
        - email
        - name
        - created_at
        - updated_at
      properties:
        id:
          type: string
          format: uuid
          description: Unique user identifier
        email:
          type: string
          format: email
          description: User email address
        name:
          type: string
          minLength: 1
          maxLength: 100
          description: User full name
        status:
          type: string
          enum: [active, inactive, suspended]
          description: User account status
        created_at:
          type: string
          format: date-time
          description: Timestamp when user was created
        updated_at:
          type: string
          format: date-time
          description: Timestamp when user was last updated
      example:
        id: "123e4567-e89b-12d3-a456-426614174000"
        email: "user@example.com"
        name: "John Doe"
        status: "active"
        created_at: "2024-01-15T10:30:00Z"
        updated_at: "2024-01-15T10:30:00Z"

    CreateUserRequest:
      type: object
      required:
        - email
        - name
      properties:
        email:
          type: string
          format: email
        name:
          type: string
          minLength: 1
          maxLength: 100
      example:
        email: "newuser@example.com"
        name: "Jane Smith"

    UpdateUserRequest:
      type: object
      required:
        - email
        - name
        - status
      properties:
        email:
          type: string
          format: email
        name:
          type: string
          minLength: 1
          maxLength: 100
        status:
          type: string
          enum: [active, inactive, suspended]

    PatchUserRequest:
      type: object
      properties:
        email:
          type: string
          format: email
        name:
          type: string
          minLength: 1
          maxLength: 100
        status:
          type: string
          enum: [active, inactive, suspended]
      minProperties: 1

    UserListResponse:
      type: object
      required:
        - data
        - pagination
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/User"
        pagination:
          $ref: "#/components/schemas/PaginationInfo"

    PaginationInfo:
      type: object
      required:
        - page
        - page_size
        - total_items
        - total_pages
      properties:
        page:
          type: integer
          description: Current page number
        page_size:
          type: integer
          description: Items per page
        total_items:
          type: integer
          description: Total number of items
        total_pages:
          type: integer
          description: Total number of pages
      example:
        page: 1
        page_size: 20
        total_items: 150
        total_pages: 8

    Error:
      type: object
      required:
        - error
        - message
      properties:
        error:
          type: string
          description: Error code
        message:
          type: string
          description: Human-readable error message
        details:
          type: array
          description: Additional error details
          items:
            type: object
            properties:
              field:
                type: string
              message:
                type: string
      example:
        error: "validation_error"
        message: "Invalid input data"
        details:
          - field: "email"
            message: "Invalid email format"

  responses:
    BadRequest:
      description: Bad request - invalid input
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    Unauthorized:
      description: Unauthorized - authentication required
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    Conflict:
      description: Conflict - resource already exists
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    InternalServerError:
      description: Internal server error
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
```

## Key Patterns

### API Versioning

- Use URL path versioning (`/v1/`, `/v2/`)
- Version in the base URL, not per endpoint
- Maintain backwards compatibility within major versions

### Pagination

- Use `page` and `page_size` query parameters
- Return pagination metadata in response
- Enforce maximum page size limits
- Use 1-indexed page numbers for user friendliness

### Resource Naming

- Use plural nouns for collections (`/users`, `/items`)
- Use hierarchical paths for sub-resources
- Keep URLs lowercase with hyphens for readability

### HTTP Methods

- GET: Retrieve resources (safe, idempotent)
- POST: Create resources (non-idempotent)
- PUT: Full update (idempotent)
- PATCH: Partial update (potentially non-idempotent)
- DELETE: Remove resources (idempotent)

### Response Codes

- 200: Successful GET, PUT, PATCH
- 201: Successful POST (created)
- 204: Successful DELETE (no content)
- 400: Bad request (validation errors)
- 401: Unauthorized (missing/invalid auth)
- 404: Resource not found
- 409: Conflict (duplicate resource)
- 500: Internal server error

### Error Handling

- Consistent error response structure
- Include error code and human-readable message
- Provide field-level validation details
- Use appropriate HTTP status codes

### Security

- Use Bearer JWT tokens for authentication
- Document security requirements per endpoint
- Support HTTPS only in production

## Validation

Validate OpenAPI spec using standard tools:

```bash
# Using Swagger CLI
npx @apidevtools/swagger-cli validate openapi.yaml

# Using OpenAPI Generator
docker run --rm -v "${PWD}:/local" openapitools/openapi-generator-cli validate -i /local/openapi.yaml

# Using Spectral (with custom rules)
npx @stoplight/spectral-cli lint openapi.yaml
```

## Best Practices

1. **Use reusable components** - Define schemas, parameters, and responses once
2. **Provide examples** - Include example requests and responses
3. **Document all fields** - Add descriptions for every schema property
4. **Define error responses** - Document all possible error codes
5. **Use consistent naming** - Follow naming conventions throughout
6. **Version carefully** - Plan for API evolution from the start
7. **Security by default** - Require authentication unless explicitly public
