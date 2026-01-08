---
entity_name: REST API Testing Pattern
entity_type: e2e-testing-pattern
language: go
domain: testing
description: End-to-end testing pattern for REST APIs with HTTP request/response validation, authentication, error handling, and test fixtures using Go testing framework
tags:
  - E2E
  - REST
  - testing
  - Go
  - HTTP-testing
---

# REST API Testing Pattern

## Philosophy

Use standard Go HTTP client to make requests exactly as API consumers would. Never import API handler or service packages. Test as a black box from the consumer's perspective.

## Core Approach

1. **Use `net/http` for requests**:

   - Standard library HTTP client
   - Proper header management (Content-Type, Authorization)
   - JSON marshaling/unmarshaling for payloads

2. **Validate against OpenAPI specs**:

   - All documented endpoints
   - All documented response codes (2xx, 4xx, 5xx)
   - Request/response schema validation

3. **Test comprehensive scenarios**:
   - Happy path (minimal and complete payloads)
   - Validation errors (format, required fields)
   - Authentication/authorization (401, 403)
   - Not found scenarios (404)
   - Conflict scenarios (409)
   - Server errors (500)
   - Network errors (timeouts, connection refused)

## Example Test Structure

```go
package integration

import (
    "bytes"
    "encoding/json"
    "net/http"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestCreateUser_ValidPayload tests successful user creation.
func TestCreateUser_ValidPayload(t *testing.T) {
    // Setup: Prepare request payload
    user := User{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
        UserType:  "standard",
    }

    payload, err := json.Marshal(user)
    require.NoError(t, err)

    // Cleanup: Delete user after test
    t.Cleanup(func() {
        deleteUserByEmail(t, user.Email)
    })

    // Execute: Make HTTP POST request
    resp, err := http.Post(
        apiBaseURL+"/api/v1/users",
        "application/json",
        bytes.NewBuffer(payload),
    )
    require.NoError(t, err)
    defer resp.Body.Close()

    // Assert: Verify status code
    assert.Equal(t, http.StatusCreated, resp.StatusCode,
        "should return 201 Created")

    // Assert: Verify response headers
    assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

    // Assert: Verify response body
    var result User
    err = json.NewDecoder(resp.Body).Decode(&result)
    require.NoError(t, err)

    assert.NotEmpty(t, result.ID, "user ID should be generated")
    assert.Equal(t, user.Email, result.Email)
    assert.Equal(t, user.FirstName, result.FirstName)
}

// TestCreateUser_InvalidEmail tests validation error.
func TestCreateUser_InvalidEmail(t *testing.T) {
    // Setup: Prepare invalid payload
    user := User{
        Email:     "not-an-email",
        FirstName: "John",
        LastName:  "Doe",
    }

    payload, _ := json.Marshal(user)

    // Execute: Make HTTP POST request
    resp, err := http.Post(
        apiBaseURL+"/api/v1/users",
        "application/json",
        bytes.NewBuffer(payload),
    )
    require.NoError(t, err)
    defer resp.Body.Close()

    // Assert: Should return 400 Bad Request
    assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

    // Assert: Error message should be clear
    var errResp ErrorResponse
    json.NewDecoder(resp.Body).Decode(&errResp)
    assert.Contains(t, errResp.Message, "invalid email format")
}

// TestGetUser_NotFound tests 404 scenario.
func TestGetUser_NotFound(t *testing.T) {
    nonExistentID := "00000000-0000-0000-0000-000000000000"

    // Execute: Make GET request
    resp, err := http.Get(
        apiBaseURL + "/api/v1/users/" + nonExistentID,
    )
    require.NoError(t, err)
    defer resp.Body.Close()

    // Assert: Should return 404 Not Found
    assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// TestListUsers_Pagination tests pagination parameters.
func TestListUsers_Pagination(t *testing.T) {
    // Execute: Request second page with limit
    resp, err := http.Get(
        apiBaseURL + "/api/v1/users?page=2&limit=10",
    )
    require.NoError(t, err)
    defer resp.Body.Close()

    // Assert: Should return 200 OK
    assert.Equal(t, http.StatusOK, resp.StatusCode)

    // Assert: Response should include pagination metadata
    var result PaginatedResponse
    json.NewDecoder(resp.Body).Decode(&result)

    assert.Equal(t, 2, result.Page)
    assert.Equal(t, 10, result.Limit)
    assert.LessOrEqual(t, len(result.Data), 10)
}
```

## Helper Functions

```go
// makeRequest is a helper for making authenticated HTTP requests.
func makeRequest(t *testing.T, method, path string, body interface{}) (*http.Response, error) {
    t.Helper()

    var reqBody []byte
    var err error

    if body != nil {
        reqBody, err = json.Marshal(body)
        require.NoError(t, err)
    }

    req, err := http.NewRequest(method, apiBaseURL+path, bytes.NewBuffer(reqBody))
    require.NoError(t, err)

    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+testToken)

    client := &http.Client{Timeout: 10 * time.Second}
    return client.Do(req)
}

// assertStatusCode verifies HTTP status code.
func assertStatusCode(t *testing.T, resp *http.Response, expected int) {
    t.Helper()

    if resp.StatusCode != expected {
        body, _ := io.ReadAll(resp.Body)
        t.Errorf("expected status %d, got %d\nResponse body: %s",
            expected, resp.StatusCode, string(body))
    }
}

// parseJSONResponse decodes JSON response into target struct.
func parseJSONResponse(t *testing.T, resp *http.Response, target interface{}) {
    t.Helper()

    err := json.NewDecoder(resp.Body).Decode(target)
    require.NoError(t, err, "failed to decode JSON response")
}

// deleteUserByEmail removes a user by email for test cleanup.
func deleteUserByEmail(t *testing.T, email string) {
    t.Helper()

    req, _ := http.NewRequest("DELETE", apiBaseURL+"/api/v1/users/by-email/"+email, nil)
    req.Header.Set("Authorization", "Bearer "+adminToken)

    client := &http.Client{Timeout: 5 * time.Second}
    resp, err := client.Do(req)
    if err != nil {
        t.Logf("warning: failed to delete user: %v", err)
        return
    }
    defer resp.Body.Close()
}
```

## Required Packages

```go
import (
    "bytes"
    "encoding/json"
    "io"
    "net/http"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

## HTTP Status Codes to Test

### Success (2xx)

- **200 OK**: Successful GET, PUT, PATCH
- **201 Created**: Successful POST creating new resource
- **204 No Content**: Successful DELETE

### Client Errors (4xx)

- **400 Bad Request**: Validation failures
- **401 Unauthorized**: Missing or invalid authentication
- **403 Forbidden**: Authenticated but insufficient permissions
- **404 Not Found**: Resource doesn't exist
- **409 Conflict**: Duplicate/constraint violation
- **422 Unprocessable Entity**: Semantic validation errors

### Server Errors (5xx)

- **500 Internal Server Error**: Unexpected server failures
- **503 Service Unavailable**: Service temporarily down

## Key Patterns

1. **Standard HTTP Client**: Use `net/http` package, not custom clients
2. **JSON Handling**: Marshal requests, unmarshal responses
3. **Header Management**: Set Content-Type, Authorization properly
4. **Test Cleanup**: Delete created resources in `t.Cleanup()`
5. **Error Response Validation**: Check error messages, not just status codes
6. **No Internal Imports**: Never import API handler packages

## Common Pitfalls

- **Ignoring response bodies**: Always check error messages, not just status codes
- **Not setting headers**: Forget Content-Type or Authorization
- **Not closing response bodies**: Always `defer resp.Body.Close()`
- **Hardcoded URLs**: Use baseURL + path pattern
- **Not testing pagination**: Test limit, offset, page parameters
- **Skipping authentication tests**: Test 401 and 403 scenarios
- **Not testing edge cases**: Empty lists, large payloads, special characters

## Test Coverage Requirements

All REST endpoints must be tested for:

### Happy Path

- Successful creation (POST 201)
- Successful retrieval (GET 200)
- Successful update (PUT/PATCH 200)
- Successful deletion (DELETE 204)
- List operations with pagination

### Validation Errors

- Invalid JSON structure (400)
- Missing required fields (400)
- Invalid field formats (400)
- Invalid field values (422)

### Authentication/Authorization

- Missing authentication (401)
- Invalid token (401)
- Insufficient permissions (403)

### Resource Errors

- Resource not found (404)
- Duplicate resource (409)
- Constraint violations (409)

### Network/Infrastructure

- Connection timeouts
- Request timeouts
- Large payload handling
- Rate limiting (429)
