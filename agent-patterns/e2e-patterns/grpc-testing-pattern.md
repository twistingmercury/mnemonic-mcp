---
entity_name: gRPC Testing Pattern
entity_type: e2e-testing-pattern
language: go
domain: testing
description: End-to-end testing pattern for gRPC services with unary and streaming RPC testing, error handling, and client setup using Go testing framework
tags:
  - E2E
  - gRPC
  - testing
  - Go
  - RPC-testing
---

# gRPC Testing Pattern

## Philosophy

Use gRPC client with generated protobuf code to make RPC calls exactly as API consumers would. Never import gRPC server implementation packages. Test as a black box from the consumer's perspective.

## Core Approach

1. **Use gRPC client**:

   - Standard `google.golang.org/grpc` client
   - Generated protobuf client stubs
   - Proper connection management and credentials
   - Context handling for deadlines and cancellation

2. **Validate against .proto definitions**:

   - All documented RPC methods
   - All documented status codes
   - Request/response message validation
   - Streaming patterns (unary, server-stream, client-stream, bidirectional)

3. **Test comprehensive scenarios**:
   - Happy path (minimal and complete payloads)
   - Validation errors (invalid types, missing required fields)
   - Authentication/authorization (UNAUTHENTICATED, PERMISSION_DENIED)
   - Not found scenarios (NOT_FOUND)
   - Deadline exceeded (DEADLINE_EXCEEDED)
   - Resource exhausted (RESOURCE_EXHAUSTED)
   - Server errors (INTERNAL, UNAVAILABLE)

## Example Test Structure

```go
package integration

import (
    "context"
    "testing"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    pb "github.com/org/project/api/proto/v1"
)

// TestCreateUser_ValidRequest tests successful user creation via gRPC.
func TestCreateUser_ValidRequest(t *testing.T) {
    // Setup: Connect to gRPC server
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Setup: Prepare request
    req := &pb.CreateUserRequest{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
        UserType:  pb.UserType_STANDARD,
    }

    // Cleanup: Delete user after test
    t.Cleanup(func() {
        deleteUserByEmail(t, client, req.Email)
    })

    // Execute: Make gRPC call with authentication
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resp, err := client.CreateUser(ctx, req)

    // Assert: RPC should succeed
    require.NoError(t, err, "CreateUser should succeed")

    // Assert: Response should contain created user
    assert.NotEmpty(t, resp.User.Id, "user ID should be generated")
    assert.Equal(t, req.Email, resp.User.Email)
    assert.Equal(t, req.FirstName, resp.User.FirstName)
    assert.Equal(t, req.LastName, resp.User.LastName)
    assert.Equal(t, req.UserType, resp.User.UserType)
    assert.NotNil(t, resp.User.CreatedAt)
}

// TestGetUser_ExistingUser tests retrieving a user by ID.
func TestGetUser_ExistingUser(t *testing.T) {
    // Setup: Connect and create test user
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)
    userID := createTestUser(t, client, "get-test@example.com", "Jane", "Smith")

    // Cleanup: Delete user after test
    t.Cleanup(func() {
        deleteUserByID(t, client, userID)
    })

    // Execute: Get user by ID
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.GetUserRequest{
        Id: userID,
    }

    resp, err := client.GetUser(ctx, req)

    // Assert: RPC should succeed
    require.NoError(t, err)

    // Assert: Response should match created user
    assert.Equal(t, userID, resp.User.Id)
    assert.Equal(t, "get-test@example.com", resp.User.Email)
    assert.Equal(t, "Jane", resp.User.FirstName)
    assert.Equal(t, "Smith", resp.User.LastName)
}

// TestCreateUser_MissingRequiredField tests validation error.
func TestCreateUser_MissingRequiredField(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Setup: Request with missing required field (email)
    req := &pb.CreateUserRequest{
        FirstName: "John",
        LastName:  "Doe",
        UserType:  pb.UserType_STANDARD,
        // Missing Email field
    }

    // Execute: Make gRPC call
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    resp, err := client.CreateUser(ctx, req)

    // Assert: Should return INVALID_ARGUMENT error
    require.Error(t, err, "should return validation error")
    assert.Nil(t, resp)

    st, ok := status.FromError(err)
    require.True(t, ok, "error should be gRPC status")
    assert.Equal(t, codes.InvalidArgument, st.Code())
    assert.Contains(t, st.Message(), "email")
}

// TestGetUser_NotFound tests not found scenario.
func TestGetUser_NotFound(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Execute: Get non-existent user
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.GetUserRequest{
        Id: "00000000-0000-0000-0000-000000000000",
    }

    resp, err := client.GetUser(ctx, req)

    // Assert: Should return NOT_FOUND error
    require.Error(t, err)
    assert.Nil(t, resp)

    st, ok := status.FromError(err)
    require.True(t, ok)
    assert.Equal(t, codes.NotFound, st.Code())
}

// TestCreateUser_Unauthenticated tests missing authentication.
func TestCreateUser_Unauthenticated(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    req := &pb.CreateUserRequest{
        Email:     "test@example.com",
        FirstName: "John",
        LastName:  "Doe",
        UserType:  pb.UserType_STANDARD,
    }

    // Execute: Make gRPC call without authentication
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    resp, err := client.CreateUser(ctx, req)

    // Assert: Should return UNAUTHENTICATED error
    require.Error(t, err)
    assert.Nil(t, resp)

    st, ok := status.FromError(err)
    require.True(t, ok)
    assert.Equal(t, codes.Unauthenticated, st.Code())
}

// TestListUsers_WithPagination tests pagination parameters.
func TestListUsers_WithPagination(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Execute: Request with pagination
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.ListUsersRequest{
        PageSize:  10,
        PageToken: "",
    }

    resp, err := client.ListUsers(ctx, req)

    // Assert: RPC should succeed
    require.NoError(t, err)

    // Assert: Response should respect pagination
    assert.LessOrEqual(t, len(resp.Users), 10,
        "should return at most 10 users")

    if len(resp.Users) > 0 && resp.NextPageToken != "" {
        assert.NotEmpty(t, resp.NextPageToken,
            "nextPageToken should be provided when more results exist")
    }
}

// TestStreamUsers_ServerSideStreaming tests server-side streaming RPC.
func TestStreamUsers_ServerSideStreaming(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Execute: Start streaming RPC
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    req := &pb.StreamUsersRequest{
        PageSize: 5,
    }

    stream, err := client.StreamUsers(ctx, req)
    require.NoError(t, err, "should start streaming")

    // Assert: Receive streamed users
    usersReceived := 0
    for {
        resp, err := stream.Recv()
        if err != nil {
            // Stream ended (io.EOF is expected)
            break
        }

        assert.NotNil(t, resp.User, "each response should contain a user")
        assert.NotEmpty(t, resp.User.Id)
        usersReceived++

        // Prevent infinite loop in case of bugs
        if usersReceived > 100 {
            t.Fatal("received too many users, possible infinite stream")
        }
    }

    assert.Greater(t, usersReceived, 0, "should receive at least one user")
}

// TestBatchCreateUsers_ClientSideStreaming tests client-side streaming RPC.
func TestBatchCreateUsers_ClientSideStreaming(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Execute: Start streaming RPC
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    stream, err := client.BatchCreateUsers(ctx)
    require.NoError(t, err, "should start client streaming")

    // Prepare test users
    testUsers := []struct {
        email     string
        firstName string
        lastName  string
    }{
        {"batch1@example.com", "User", "One"},
        {"batch2@example.com", "User", "Two"},
        {"batch3@example.com", "User", "Three"},
    }

    // Cleanup: Delete all test users
    for _, u := range testUsers {
        email := u.email
        t.Cleanup(func() {
            deleteUserByEmail(t, client, email)
        })
    }

    // Send: Stream multiple user creation requests
    for _, u := range testUsers {
        req := &pb.CreateUserRequest{
            Email:     u.email,
            FirstName: u.firstName,
            LastName:  u.lastName,
            UserType:  pb.UserType_STANDARD,
        }

        err := stream.Send(req)
        require.NoError(t, err, "should send request")
    }

    // Receive: Get batch response
    resp, err := stream.CloseAndRecv()
    require.NoError(t, err, "should receive batch response")

    // Assert: All users should be created
    assert.Equal(t, len(testUsers), int(resp.UsersCreated),
        "all users should be created")
}

// TestDeadlineExceeded tests timeout handling.
func TestDeadlineExceeded(t *testing.T) {
    conn := newGRPCConnection(t)
    defer conn.Close()

    client := pb.NewUserServiceClient(conn)

    // Execute: Make call with very short deadline
    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
    defer cancel()

    // Wait to ensure deadline passes
    time.Sleep(10 * time.Millisecond)

    req := &pb.ListUsersRequest{
        PageSize: 10,
    }

    resp, err := client.ListUsers(ctx, req)

    // Assert: Should return DEADLINE_EXCEEDED error
    require.Error(t, err)
    assert.Nil(t, resp)

    st, ok := status.FromError(err)
    require.True(t, ok)
    assert.Equal(t, codes.DeadlineExceeded, st.Code())
}
```

## Helper Functions

```go
// newGRPCConnection creates a new gRPC client connection.
func newGRPCConnection(t *testing.T) *grpc.ClientConn {
    t.Helper()

    // For testing, use insecure credentials
    // In production, use proper TLS credentials
    opts := []grpc.DialOption{
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    }

    conn, err := grpc.Dial(grpcServerAddress, opts...)
    require.NoError(t, err, "failed to connect to gRPC server")

    return conn
}

// contextWithAuth adds authentication metadata to context.
func contextWithAuth(ctx context.Context, token string) context.Context {
    md := metadata.Pairs("authorization", "Bearer "+token)
    return metadata.NewOutgoingContext(ctx, md)
}

// assertGRPCStatus verifies that a gRPC error has the expected status code.
func assertGRPCStatus(t *testing.T, err error, expectedCode codes.Code) {
    t.Helper()

    require.Error(t, err, "expected gRPC error")

    st, ok := status.FromError(err)
    require.True(t, ok, "error should be gRPC status")
    assert.Equal(t, expectedCode, st.Code(),
        "expected status code %v, got %v: %s",
        expectedCode, st.Code(), st.Message())
}

// assertGRPCStatusMessage verifies status code and message content.
func assertGRPCStatusMessage(t *testing.T, err error, expectedCode codes.Code, messageSubstring string) {
    t.Helper()

    require.Error(t, err, "expected gRPC error")

    st, ok := status.FromError(err)
    require.True(t, ok, "error should be gRPC status")
    assert.Equal(t, expectedCode, st.Code())
    assert.Contains(t, st.Message(), messageSubstring)
}

// createTestUser creates a user for testing and returns the ID.
func createTestUser(t *testing.T, client pb.UserServiceClient, email, firstName, lastName string) string {
    t.Helper()

    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.CreateUserRequest{
        Email:     email,
        FirstName: firstName,
        LastName:  lastName,
        UserType:  pb.UserType_STANDARD,
    }

    resp, err := client.CreateUser(ctx, req)
    require.NoError(t, err, "failed to create test user")

    return resp.User.Id
}

// deleteUserByID removes a user by ID for test cleanup.
func deleteUserByID(t *testing.T, client pb.UserServiceClient, userID string) {
    t.Helper()

    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.DeleteUserRequest{
        Id: userID,
    }

    _, err := client.DeleteUser(ctx, req)
    if err != nil {
        t.Logf("warning: failed to delete user: %v", err)
    }
}

// deleteUserByEmail removes a user by email for test cleanup.
func deleteUserByEmail(t *testing.T, client pb.UserServiceClient, email string) {
    t.Helper()

    ctx := contextWithAuth(context.Background(), testToken)
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    req := &pb.DeleteUserByEmailRequest{
        Email: email,
    }

    _, err := client.DeleteUserByEmail(ctx, req)
    if err != nil {
        t.Logf("warning: failed to delete user by email: %v", err)
    }
}
```

## Required Packages

```go
import (
    "context"
    "io"
    "testing"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"

    // Import your generated protobuf code
    pb "github.com/org/project/api/proto/v1"
)
```

## gRPC Status Codes to Test

### Success (OK)

- **codes.OK**: Successful RPC call

### Client Errors

- **codes.InvalidArgument**: Validation failures, malformed input
- **codes.Unauthenticated**: Missing or invalid authentication
- **codes.PermissionDenied**: Authenticated but insufficient permissions
- **codes.NotFound**: Resource doesn't exist
- **codes.AlreadyExists**: Duplicate/constraint violation
- **codes.FailedPrecondition**: Operation rejected by current state
- **codes.OutOfRange**: Parameter out of valid range

### Server Errors

- **codes.Internal**: Unexpected server failures
- **codes.Unavailable**: Service temporarily unavailable
- **codes.DataLoss**: Unrecoverable data loss or corruption
- **codes.Unknown**: Unknown error

### Timeout/Resource Errors

- **codes.DeadlineExceeded**: Operation timeout
- **codes.ResourceExhausted**: Rate limiting, quota exceeded
- **codes.Aborted**: Concurrent operation conflict
- **codes.Cancelled**: Client cancelled the request

## Streaming Patterns

### Server-Side Streaming

- Server sends multiple messages
- Client receives stream via `stream.Recv()`
- Loop until `io.EOF` received
- Test: message ordering, early termination, errors during stream

### Client-Side Streaming

- Client sends multiple messages
- Server receives stream and sends single response
- Client uses `stream.Send()` and `stream.CloseAndRecv()`
- Test: partial sends, errors during streaming, timeout handling

### Bidirectional Streaming

- Both client and server send multiple messages
- Full-duplex communication
- Use separate goroutines for sending and receiving
- Test: concurrent send/receive, ordering, backpressure

## Key Patterns

1. **Standard gRPC Client**: Use `google.golang.org/grpc` package
2. **Generated Protobuf Code**: Import and use generated client stubs
3. **Context Management**: Always use `context.WithTimeout()` for deadlines
4. **Metadata for Auth**: Use `metadata.NewOutgoingContext()` for authentication
5. **Status Code Checking**: Use `status.FromError()` to extract gRPC status
6. **Connection Reuse**: Create connection once, reuse for multiple calls
7. **Test Cleanup**: Delete created resources in `t.Cleanup()`
8. **No Internal Imports**: Never import gRPC server implementation packages

## Common Pitfalls

- **Forgetting context deadlines**: Always set timeouts with `context.WithTimeout()`
- **Not closing connections**: Always `defer conn.Close()`
- **Ignoring stream errors**: Check `stream.Recv()` errors properly
- **Not handling io.EOF**: This is expected for end of stream, not an error
- **Hardcoded addresses**: Use `grpcServerAddress` variable
- **Not testing streaming**: Test all RPC types (unary, server-stream, client-stream, bidirectional)
- **Skipping status details**: Check both status code and message
- **Not testing metadata**: Test authentication and custom headers
- **Goroutine leaks**: Properly cancel contexts and close streams

## Test Coverage Requirements

All gRPC services must be tested for:

### Unary RPCs

- Successful call (OK)
- Invalid input (INVALID_ARGUMENT)
- Not found (NOT_FOUND)
- Already exists (ALREADY_EXISTS)
- Unauthenticated (UNAUTHENTICATED)
- Permission denied (PERMISSION_DENIED)
- Deadline exceeded (DEADLINE_EXCEEDED)

### Server-Side Streaming RPCs

- Full stream reception
- Early stream termination
- Error during streaming
- Empty stream
- Large stream handling

### Client-Side Streaming RPCs

- Multiple message sends
- Single message send
- Error during send
- Timeout during streaming
- `CloseAndRecv()` handling

### Bidirectional Streaming RPCs

- Concurrent send/receive
- Send then receive pattern
- Receive then send pattern
- Error handling in both directions
- Stream closure from either side

### Authentication/Authorization

- Missing credentials (UNAUTHENTICATED)
- Invalid credentials (UNAUTHENTICATED)
- Expired credentials (UNAUTHENTICATED)
- Insufficient permissions (PERMISSION_DENIED)
- Role-based access control

### Resource Management

- Connection pooling
- Context cancellation
- Timeout handling
- Graceful shutdown
- Error recovery

### Edge Cases

- Very large messages
- Concurrent requests
- Network failures
- Server unavailable
- Partial failures
