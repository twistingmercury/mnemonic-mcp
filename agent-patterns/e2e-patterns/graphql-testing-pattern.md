---
entity_name: GraphQL Testing Pattern
entity_type: e2e-testing-pattern
language: go
domain: testing
description: End-to-end testing pattern for GraphQL APIs with query execution, mutation testing, subscription validation, and error handling using Go testing framework
tags:
  - E2E
  - GraphQL
  - testing
  - Go
  - API-testing
---

# GraphQL Testing Pattern

## Philosophy

Use GraphQL client libraries to execute queries and mutations exactly as API consumers would. Never import GraphQL resolver or internal server packages. Test as a black box from the consumer's perspective.

## Core Approach

1. **Use GraphQL client library**:

   - `github.com/machinebox/graphql` for queries and mutations
   - Proper variable binding and type marshaling
   - Request header management (Authorization, Content-Type)

2. **Validate against GraphQL schema**:

   - All documented queries and mutations
   - All documented types and fields
   - Schema validation for requests and responses

3. **Test comprehensive scenarios**:
   - Happy path (minimal and complete payloads)
   - Validation errors (invalid types, missing required fields)
   - Authentication/authorization errors
   - Not found scenarios
   - Partial response handling
   - Error response validation

## Example Test Structure

```go
package integration

import (
    "context"
    "testing"

    "github.com/machinebox/graphql"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestCreateUser_ValidInput tests successful user creation via GraphQL.
func TestCreateUser_ValidInput(t *testing.T) {
    client := graphql.NewClient(graphqlBaseURL)

    // Setup: Prepare mutation
    mutation := `
        mutation CreateUser($input: CreateUserInput!) {
            createUser(input: $input) {
                id
                email
                firstName
                lastName
                createdAt
            }
        }
    `

    // Setup: Prepare variables
    variables := map[string]interface{}{
        "input": map[string]interface{}{
            "email":     "test@example.com",
            "firstName": "John",
            "lastName":  "Doe",
        },
    }

    // Cleanup: Delete user after test
    t.Cleanup(func() {
        deleteUserByEmail(t, "test@example.com")
    })

    // Execute: Run mutation
    req := graphql.NewRequest(mutation)
    for k, v := range variables {
        req.Var(k, v)
    }
    req.Header.Set("Authorization", "Bearer "+testToken)

    var response struct {
        CreateUser struct {
            ID        string `json:"id"`
            Email     string `json:"email"`
            FirstName string `json:"firstName"`
            LastName  string `json:"lastName"`
            CreatedAt string `json:"createdAt"`
        } `json:"createUser"`
    }

    ctx := context.Background()
    err := client.Run(ctx, req, &response)

    // Assert: Mutation should succeed
    require.NoError(t, err, "mutation should succeed")

    // Assert: Response should contain created user
    assert.NotEmpty(t, response.CreateUser.ID, "user ID should be generated")
    assert.Equal(t, "test@example.com", response.CreateUser.Email)
    assert.Equal(t, "John", response.CreateUser.FirstName)
    assert.Equal(t, "Doe", response.CreateUser.LastName)
    assert.NotEmpty(t, response.CreateUser.CreatedAt)
}

// TestQueryUser_ByID tests retrieving a user by ID.
func TestQueryUser_ByID(t *testing.T) {
    client := graphql.NewClient(graphqlBaseURL)

    // Setup: Create a test user first
    userID := createTestUser(t, "query-test@example.com", "Jane", "Smith")

    // Cleanup: Delete user after test
    t.Cleanup(func() {
        deleteUserByID(t, userID)
    })

    // Execute: Query user by ID
    query := `
        query GetUser($id: ID!) {
            user(id: $id) {
                id
                email
                firstName
                lastName
            }
        }
    `

    req := graphql.NewRequest(query)
    req.Var("id", userID)
    req.Header.Set("Authorization", "Bearer "+testToken)

    var response struct {
        User struct {
            ID        string `json:"id"`
            Email     string `json:"email"`
            FirstName string `json:"firstName"`
            LastName  string `json:"lastName"`
        } `json:"user"`
    }

    ctx := context.Background()
    err := client.Run(ctx, req, &response)

    // Assert: Query should succeed
    require.NoError(t, err)

    // Assert: Response should match created user
    assert.Equal(t, userID, response.User.ID)
    assert.Equal(t, "query-test@example.com", response.User.Email)
    assert.Equal(t, "Jane", response.User.FirstName)
    assert.Equal(t, "Smith", response.User.LastName)
}

// TestCreateUser_MissingRequiredField tests validation error handling.
func TestCreateUser_MissingRequiredField(t *testing.T) {
    client := graphql.NewClient(graphqlBaseURL)

    // Setup: Mutation with missing required field (email)
    mutation := `
        mutation CreateUser($input: CreateUserInput!) {
            createUser(input: $input) {
                id
                email
            }
        }
    `

    variables := map[string]interface{}{
        "input": map[string]interface{}{
            "firstName": "John",
            "lastName":  "Doe",
            // Missing email field
        },
    }

    // Execute: Run mutation
    req := graphql.NewRequest(mutation)
    for k, v := range variables {
        req.Var(k, v)
    }
    req.Header.Set("Authorization", "Bearer "+testToken)

    var response struct {
        CreateUser struct {
            ID    string `json:"id"`
            Email string `json:"email"`
        } `json:"createUser"`
    }

    ctx := context.Background()
    err := client.Run(ctx, req, &response)

    // Assert: Should return validation error
    require.Error(t, err, "mutation should fail")
    assert.Contains(t, err.Error(), "email")
    assert.Contains(t, err.Error(), "required")
}

// TestQueryUser_NotFound tests not found scenario.
func TestQueryUser_NotFound(t *testing.T) {
    client := graphql.NewClient(graphqlBaseURL)

    nonExistentID := "00000000-0000-0000-0000-000000000000"

    // Execute: Query non-existent user
    query := `
        query GetUser($id: ID!) {
            user(id: $id) {
                id
                email
            }
        }
    `

    req := graphql.NewRequest(query)
    req.Var("id", nonExistentID)
    req.Header.Set("Authorization", "Bearer "+testToken)

    var response struct {
        User *struct {
            ID    string `json:"id"`
            Email string `json:"email"`
        } `json:"user"`
    }

    ctx := context.Background()
    err := client.Run(ctx, req, &response)

    // Assert: Should return null for user (or error, depending on schema)
    if err != nil {
        assert.Contains(t, err.Error(), "not found")
    } else {
        assert.Nil(t, response.User, "user should be null when not found")
    }
}

// TestListUsers_WithPagination tests pagination parameters.
func TestListUsers_WithPagination(t *testing.T) {
    client := graphql.NewClient(graphqlBaseURL)

    // Execute: Query with pagination
    query := `
        query ListUsers($first: Int!, $after: String) {
            users(first: $first, after: $after) {
                edges {
                    node {
                        id
                        email
                    }
                    cursor
                }
                pageInfo {
                    hasNextPage
                    endCursor
                }
            }
        }
    `

    req := graphql.NewRequest(query)
    req.Var("first", 10)
    req.Header.Set("Authorization", "Bearer "+testToken)

    var response struct {
        Users struct {
            Edges []struct {
                Node struct {
                    ID    string `json:"id"`
                    Email string `json:"email"`
                } `json:"node"`
                Cursor string `json:"cursor"`
            } `json:"edges"`
            PageInfo struct {
                HasNextPage bool   `json:"hasNextPage"`
                EndCursor   string `json:"endCursor"`
            } `json:"pageInfo"`
        } `json:"users"`
    }

    ctx := context.Background()
    err := client.Run(ctx, req, &response)

    // Assert: Query should succeed
    require.NoError(t, err)

    // Assert: Response should respect pagination
    assert.LessOrEqual(t, len(response.Users.Edges), 10,
        "should return at most 10 users")

    if len(response.Users.Edges) > 0 {
        assert.NotEmpty(t, response.Users.Edges[0].Cursor,
            "each edge should have a cursor")
    }

    if response.Users.PageInfo.HasNextPage {
        assert.NotEmpty(t, response.Users.PageInfo.EndCursor,
            "endCursor should be provided when hasNextPage is true")
    }
}

// TestUpdateUser_PartialUpdate tests partial field updates.
func TestUpdateUser_PartialUpdate(t *testing.T) {
    client := graphql.NewClient(graphqlBaseURL)

    // Setup: Create a test user
    userID := createTestUser(t, "update-test@example.com", "Jane", "Smith")

    // Cleanup: Delete user after test
    t.Cleanup(func() {
        deleteUserByID(t, userID)
    })

    // Execute: Update only firstName
    mutation := `
        mutation UpdateUser($id: ID!, $input: UpdateUserInput!) {
            updateUser(id: $id, input: $input) {
                id
                firstName
                lastName
            }
        }
    `

    req := graphql.NewRequest(mutation)
    req.Var("id", userID)
    req.Var("input", map[string]interface{}{
        "firstName": "Janet",
        // Not updating lastName
    })
    req.Header.Set("Authorization", "Bearer "+testToken)

    var response struct {
        UpdateUser struct {
            ID        string `json:"id"`
            FirstName string `json:"firstName"`
            LastName  string `json:"lastName"`
        } `json:"updateUser"`
    }

    ctx := context.Background()
    err := client.Run(ctx, req, &response)

    // Assert: Mutation should succeed
    require.NoError(t, err)

    // Assert: Only firstName should be updated
    assert.Equal(t, userID, response.UpdateUser.ID)
    assert.Equal(t, "Janet", response.UpdateUser.FirstName)
    assert.Equal(t, "Smith", response.UpdateUser.LastName, "lastName should remain unchanged")
}
```

## Helper Functions

```go
// executeQuery executes a GraphQL query with authentication.
func executeQuery(t *testing.T, query string, variables map[string]interface{}, target interface{}) error {
    t.Helper()

    client := graphql.NewClient(graphqlBaseURL)
    req := graphql.NewRequest(query)

    for k, v := range variables {
        req.Var(k, v)
    }

    req.Header.Set("Authorization", "Bearer "+testToken)
    req.Header.Set("Content-Type", "application/json")

    ctx := context.Background()
    return client.Run(ctx, req, target)
}

// executeMutation executes a GraphQL mutation with authentication.
func executeMutation(t *testing.T, mutation string, variables map[string]interface{}, target interface{}) error {
    t.Helper()

    // Same implementation as executeQuery (GraphQL doesn't distinguish at transport level)
    return executeQuery(t, mutation, variables, target)
}

// assertGraphQLError verifies that a GraphQL operation returned an error containing expected text.
func assertGraphQLError(t *testing.T, err error, expectedSubstring string) {
    t.Helper()

    require.Error(t, err, "expected GraphQL error")
    assert.Contains(t, err.Error(), expectedSubstring,
        "error message should contain '%s'", expectedSubstring)
}

// createTestUser creates a user for testing and returns the ID.
func createTestUser(t *testing.T, email, firstName, lastName string) string {
    t.Helper()

    mutation := `
        mutation CreateUser($input: CreateUserInput!) {
            createUser(input: $input) {
                id
            }
        }
    `

    variables := map[string]interface{}{
        "input": map[string]interface{}{
            "email":     email,
            "firstName": firstName,
            "lastName":  lastName,
        },
    }

    var response struct {
        CreateUser struct {
            ID string `json:"id"`
        } `json:"createUser"`
    }

    err := executeMutation(t, mutation, variables, &response)
    require.NoError(t, err, "failed to create test user")

    return response.CreateUser.ID
}

// deleteUserByID removes a user by ID for test cleanup.
func deleteUserByID(t *testing.T, userID string) {
    t.Helper()

    mutation := `
        mutation DeleteUser($id: ID!) {
            deleteUser(id: $id)
        }
    `

    variables := map[string]interface{}{
        "id": userID,
    }

    var response struct {
        DeleteUser bool `json:"deleteUser"`
    }

    err := executeMutation(t, mutation, variables, &response)
    if err != nil {
        t.Logf("warning: failed to delete user: %v", err)
    }
}

// deleteUserByEmail removes a user by email for test cleanup.
func deleteUserByEmail(t *testing.T, email string) {
    t.Helper()

    mutation := `
        mutation DeleteUserByEmail($email: String!) {
            deleteUserByEmail(email: $email)
        }
    `

    variables := map[string]interface{}{
        "email": email,
    }

    var response struct {
        DeleteUserByEmail bool `json:"deleteUserByEmail"`
    }

    err := executeMutation(t, mutation, variables, &response)
    if err != nil {
        t.Logf("warning: failed to delete user by email: %v", err)
    }
}
```

## Required Packages

```go
import (
    "context"
    "testing"

    "github.com/machinebox/graphql"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)
```

## GraphQL-Specific Testing Patterns

### Query Testing

- **Fields Selection**: Test with minimal fields and all available fields
- **Nested Objects**: Verify nested object resolution
- **Aliases**: Test field aliasing when needed
- **Fragments**: Test with inline and named fragments

### Mutation Testing

- **Input Validation**: Test required fields, type validation, format validation
- **Partial Updates**: Verify only specified fields are updated
- **Cascading Operations**: Test mutations that affect related entities
- **Optimistic Responses**: Test mutation response structure

### Error Handling

- **GraphQL Errors**: Errors returned in `errors` array with `message`, `path`, `extensions`
- **Validation Errors**: Schema-level validation failures
- **Authorization Errors**: Field-level permission checks
- **Custom Errors**: Application-specific error codes and messages

### Pagination Patterns

- **Cursor-based**: Test `first`, `after`, `last`, `before` arguments
- **Offset-based**: Test `limit` and `offset` arguments
- **Page Info**: Verify `hasNextPage`, `hasPreviousPage`, `startCursor`, `endCursor`
- **Connection Pattern**: Test `edges`, `node`, `cursor` structure

## Key Patterns

1. **Use GraphQL Client**: Use `github.com/machinebox/graphql`, not raw HTTP
2. **Type-Safe Responses**: Define response structs matching GraphQL schema
3. **Variable Binding**: Use `req.Var()` for query/mutation variables
4. **Context Usage**: Always pass `context.Context` to `client.Run()`
5. **Header Management**: Set Authorization and Content-Type headers
6. **Test Cleanup**: Use `t.Cleanup()` with deletion mutations
7. **No Internal Imports**: Never import GraphQL resolver packages

## Common Pitfalls

- **Ignoring GraphQL errors**: Check both `err` and response `errors` array
- **Not testing partial responses**: GraphQL can return partial data with errors
- **Hardcoded URLs**: Use `graphqlBaseURL` variable
- **Not testing pagination**: Test cursor handling and page boundaries
- **Skipping field-level auth**: Test both query-level and field-level permissions
- **Not testing null values**: GraphQL distinguishes between null and missing fields
- **Forgetting context**: Always provide `context.Background()` or deadline context

## Test Coverage Requirements

All GraphQL operations must be tested for:

### Query Testing

- Successful retrieval with various field selections
- Not found scenarios (null vs error)
- Pagination parameters (cursors, limits)
- Filtering and sorting arguments
- Nested object resolution

### Mutation Testing

- Successful creation/update/deletion
- Input validation errors
- Required field enforcement
- Type validation
- Conflict scenarios
- Partial updates

### Error Scenarios

- GraphQL validation errors (schema violations)
- Business logic errors (custom error codes)
- Authentication errors (401 equivalent)
- Authorization errors (403 equivalent, field-level)
- Not found errors
- Rate limiting errors

### Edge Cases

- Empty result sets
- Large result sets (pagination)
- Null vs missing fields
- Complex nested input types
- Union and interface types
- Fragment usage
