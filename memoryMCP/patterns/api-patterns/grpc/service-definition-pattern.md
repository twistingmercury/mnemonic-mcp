---
entity_name: gRPC Service Definition Pattern
entity_type: gRPC Pattern
language: go
domain: backend
description: Comprehensive Protocol Buffer service definition pattern with unary and streaming RPCs, error handling, and Go implementation
tags:
  - gRPC
  - Protocol Buffers
  - protobuf
  - Go
  - RPC
---

# gRPC Service Definition Pattern

This pattern provides complete Protocol Buffer definitions for gRPC services in Go projects.

## Complete Proto File

```protobuf
// api/proto/v1/user_service.proto
syntax = "proto3";

package user.v1;

option go_package = "github.com/yourorg/yourproject/api/gen/go/user/v1;userv1";

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/field_mask.proto";

// UserService provides user management operations
service UserService {
  // Unary RPC: Get a single user by ID
  rpc GetUser(GetUserRequest) returns (GetUserResponse);

  // Unary RPC: List users with pagination
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);

  // Unary RPC: Create a new user
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);

  // Unary RPC: Update an existing user
  rpc UpdateUser(UpdateUserRequest) returns (UpdateUserResponse);

  // Unary RPC: Delete a user
  rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty);

  // Server streaming RPC: Stream user events
  rpc StreamUserEvents(StreamUserEventsRequest) returns (stream UserEvent);

  // Client streaming RPC: Batch create users
  rpc BatchCreateUsers(stream CreateUserRequest) returns (BatchCreateUsersResponse);

  // Bidirectional streaming RPC: Real-time user updates
  rpc SyncUsers(stream UserSyncRequest) returns (stream UserSyncResponse);
}

// User message
message User {
  // Unique user identifier (UUID format)
  string id = 1;

  // User email address (required, unique)
  string email = 2;

  // User display name
  string name = 3;

  // User status
  UserStatus status = 4;

  // User role for authorization
  UserRole role = 5;

  // User profile information
  UserProfile profile = 6;

  // Creation timestamp
  google.protobuf.Timestamp created_at = 7;

  // Last update timestamp
  google.protobuf.Timestamp updated_at = 8;
}

// User profile information
message UserProfile {
  // Profile bio
  string bio = 1;

  // Avatar URL
  string avatar_url = 2;

  // User location
  string location = 3;

  // User website
  string website = 4;
}

// User status enum
enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;
  USER_STATUS_ACTIVE = 1;
  USER_STATUS_INACTIVE = 2;
  USER_STATUS_SUSPENDED = 3;
}

// User role enum
enum UserRole {
  USER_ROLE_UNSPECIFIED = 0;
  USER_ROLE_GUEST = 1;
  USER_ROLE_USER = 2;
  USER_ROLE_ADMIN = 3;
}

// GetUser request
message GetUserRequest {
  // User ID to retrieve
  string id = 1;
}

// GetUser response
message GetUserResponse {
  // Retrieved user
  User user = 1;
}

// ListUsers request with pagination
message ListUsersRequest {
  // Maximum number of users to return
  int32 page_size = 1;

  // Page token from previous ListUsers call
  string page_token = 2;

  // Filter by user status
  optional UserStatus status = 3;

  // Filter by user role
  optional UserRole role = 4;

  // Search query for name or email
  optional string query = 5;

  // Sort order
  SortOrder sort_order = 6;

  // Sort field
  string sort_by = 7;
}

// Sort order enum
enum SortOrder {
  SORT_ORDER_UNSPECIFIED = 0;
  SORT_ORDER_ASC = 1;
  SORT_ORDER_DESC = 2;
}

// ListUsers response
message ListUsersResponse {
  // List of users
  repeated User users = 1;

  // Token for retrieving next page
  string next_page_token = 2;

  // Total number of users matching filter
  int32 total_count = 3;
}

// CreateUser request
message CreateUserRequest {
  // User email (required)
  string email = 1;

  // User name (required)
  string name = 2;

  // User password (required, will be hashed)
  string password = 3;

  // User role (optional, defaults to USER)
  optional UserRole role = 4;

  // User profile (optional)
  optional UserProfile profile = 5;
}

// CreateUser response
message CreateUserResponse {
  // Created user
  User user = 1;
}

// UpdateUser request
message UpdateUserRequest {
  // User ID to update (required)
  string id = 1;

  // Fields to update
  User user = 2;

  // Field mask specifying which fields to update
  google.protobuf.FieldMask update_mask = 3;
}

// UpdateUser response
message UpdateUserResponse {
  // Updated user
  User user = 1;
}

// DeleteUser request
message DeleteUserRequest {
  // User ID to delete
  string id = 1;
}

// StreamUserEvents request
message StreamUserEventsRequest {
  // Filter by user IDs (empty = all users)
  repeated string user_ids = 1;

  // Event types to stream
  repeated UserEventType event_types = 2;
}

// User event types
enum UserEventType {
  USER_EVENT_TYPE_UNSPECIFIED = 0;
  USER_EVENT_TYPE_CREATED = 1;
  USER_EVENT_TYPE_UPDATED = 2;
  USER_EVENT_TYPE_DELETED = 3;
  USER_EVENT_TYPE_STATUS_CHANGED = 4;
}

// User event message
message UserEvent {
  // Event ID
  string id = 1;

  // Event type
  UserEventType type = 2;

  // User associated with event
  User user = 3;

  // Event timestamp
  google.protobuf.Timestamp timestamp = 4;

  // Previous user state (for updates)
  optional User previous_user = 5;
}

// BatchCreateUsers response
message BatchCreateUsersResponse {
  // Created users
  repeated User users = 1;

  // Number of users successfully created
  int32 created_count = 2;

  // Number of users that failed
  int32 failed_count = 3;

  // Error details for failed users
  repeated UserError errors = 4;
}

// UserError message
message UserError {
  // User email that failed
  string email = 1;

  // Error message
  string message = 2;

  // Error code
  string code = 3;
}

// UserSyncRequest for bidirectional streaming
message UserSyncRequest {
  // Operation type
  SyncOperation operation = 1;

  // User data
  User user = 2;
}

// SyncOperation enum
enum SyncOperation {
  SYNC_OPERATION_UNSPECIFIED = 0;
  SYNC_OPERATION_CREATE = 1;
  SYNC_OPERATION_UPDATE = 2;
  SYNC_OPERATION_DELETE = 3;
}

// UserSyncResponse for bidirectional streaming
message UserSyncResponse {
  // Operation result
  SyncResult result = 1;

  // User data (for successful operations)
  optional User user = 2;

  // Error message (for failed operations)
  optional string error = 3;
}

// SyncResult enum
enum SyncResult {
  SYNC_RESULT_UNSPECIFIED = 0;
  SYNC_RESULT_SUCCESS = 1;
  SYNC_RESULT_ERROR = 2;
}
```

## Go Server Implementation

```go
// internal/grpc/server/user_service.go
package server

import (
    "context"
    "io"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/emptypb"

    userv1 "github.com/yourorg/yourproject/api/gen/go/user/v1"
    "github.com/yourorg/yourproject/internal/service"
)

type UserServiceServer struct {
    userv1.UnimplementedUserServiceServer
    userService *service.UserService
}

func NewUserServiceServer(userService *service.UserService) *UserServiceServer {
    return &UserServiceServer{
        userService: userService,
    }
}

// GetUser implements unary RPC
func (s *UserServiceServer) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    user, err := s.userService.GetUserByID(ctx, req.Id)
    if err != nil {
        if errors.Is(err, service.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        return nil, status.Error(codes.Internal, "internal server error")
    }

    return &userv1.GetUserResponse{
        User: toProtoUser(user),
    }, nil
}

// ListUsers implements unary RPC with pagination
func (s *UserServiceServer) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
    opts := service.ListUsersOptions{
        PageSize:  req.PageSize,
        PageToken: req.PageToken,
        Status:    req.Status,
        Role:      req.Role,
        Query:     req.Query,
        SortBy:    req.SortBy,
        SortOrder: req.SortOrder,
    }

    result, err := s.userService.ListUsers(ctx, opts)
    if err != nil {
        return nil, status.Error(codes.Internal, "failed to list users")
    }

    users := make([]*userv1.User, len(result.Users))
    for i, u := range result.Users {
        users[i] = toProtoUser(u)
    }

    return &userv1.ListUsersResponse{
        Users:         users,
        NextPageToken: result.NextPageToken,
        TotalCount:    result.TotalCount,
    }, nil
}

// CreateUser implements unary RPC
func (s *UserServiceServer) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
    if req.Email == "" || req.Name == "" || req.Password == "" {
        return nil, status.Error(codes.InvalidArgument, "email, name, and password are required")
    }

    user, err := s.userService.CreateUser(ctx, service.CreateUserInput{
        Email:    req.Email,
        Name:     req.Name,
        Password: req.Password,
        Role:     req.Role,
        Profile:  fromProtoProfile(req.Profile),
    })
    if err != nil {
        if errors.Is(err, service.ErrAlreadyExists) {
            return nil, status.Error(codes.AlreadyExists, "user already exists")
        }
        return nil, status.Error(codes.Internal, "failed to create user")
    }

    return &userv1.CreateUserResponse{
        User: toProtoUser(user),
    }, nil
}

// UpdateUser implements unary RPC with field mask
func (s *UserServiceServer) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.UpdateUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    user, err := s.userService.UpdateUser(ctx, req.Id, service.UpdateUserInput{
        User:       fromProtoUser(req.User),
        UpdateMask: req.UpdateMask,
    })
    if err != nil {
        if errors.Is(err, service.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        return nil, status.Error(codes.Internal, "failed to update user")
    }

    return &userv1.UpdateUserResponse{
        User: toProtoUser(user),
    }, nil
}

// DeleteUser implements unary RPC
func (s *UserServiceServer) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*emptypb.Empty, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    if err := s.userService.DeleteUser(ctx, req.Id); err != nil {
        if errors.Is(err, service.ErrNotFound) {
            return nil, status.Error(codes.NotFound, "user not found")
        }
        return nil, status.Error(codes.Internal, "failed to delete user")
    }

    return &emptypb.Empty{}, nil
}

// StreamUserEvents implements server streaming RPC
func (s *UserServiceServer) StreamUserEvents(req *userv1.StreamUserEventsRequest, stream userv1.UserService_StreamUserEventsServer) error {
    ctx := stream.Context()

    eventChan, err := s.userService.SubscribeUserEvents(ctx, service.EventSubscription{
        UserIDs:    req.UserIds,
        EventTypes: req.EventTypes,
    })
    if err != nil {
        return status.Error(codes.Internal, "failed to subscribe to events")
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case event, ok := <-eventChan:
            if !ok {
                return nil
            }
            if err := stream.Send(toProtoEvent(event)); err != nil {
                return status.Error(codes.Internal, "failed to send event")
            }
        }
    }
}

// BatchCreateUsers implements client streaming RPC
func (s *UserServiceServer) BatchCreateUsers(stream userv1.UserService_BatchCreateUsersServer) error {
    var created []*userv1.User
    var errors []*userv1.UserError
    createdCount := 0
    failedCount := 0

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&userv1.BatchCreateUsersResponse{
                Users:        created,
                CreatedCount: int32(createdCount),
                FailedCount:  int32(failedCount),
                Errors:       errors,
            })
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to receive user")
        }

        user, err := s.userService.CreateUser(stream.Context(), service.CreateUserInput{
            Email:    req.Email,
            Name:     req.Name,
            Password: req.Password,
            Role:     req.Role,
            Profile:  fromProtoProfile(req.Profile),
        })

        if err != nil {
            failedCount++
            errors = append(errors, &userv1.UserError{
                Email:   req.Email,
                Message: err.Error(),
                Code:    "CREATE_FAILED",
            })
        } else {
            createdCount++
            created = append(created, toProtoUser(user))
        }
    }
}

// SyncUsers implements bidirectional streaming RPC
func (s *UserServiceServer) SyncUsers(stream userv1.UserService_SyncUsersServer) error {
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to receive sync request")
        }

        var result *userv1.UserSyncResponse

        switch req.Operation {
        case userv1.SyncOperation_SYNC_OPERATION_CREATE:
            user, err := s.userService.CreateUser(stream.Context(), fromProtoCreateInput(req.User))
            if err != nil {
                result = &userv1.UserSyncResponse{
                    Result: userv1.SyncResult_SYNC_RESULT_ERROR,
                    Error:  ptr(err.Error()),
                }
            } else {
                result = &userv1.UserSyncResponse{
                    Result: userv1.SyncResult_SYNC_RESULT_SUCCESS,
                    User:   toProtoUser(user),
                }
            }

        case userv1.SyncOperation_SYNC_OPERATION_UPDATE:
            user, err := s.userService.UpdateUser(stream.Context(), req.User.Id, fromProtoUpdateInput(req.User))
            if err != nil {
                result = &userv1.UserSyncResponse{
                    Result: userv1.SyncResult_SYNC_RESULT_ERROR,
                    Error:  ptr(err.Error()),
                }
            } else {
                result = &userv1.UserSyncResponse{
                    Result: userv1.SyncResult_SYNC_RESULT_SUCCESS,
                    User:   toProtoUser(user),
                }
            }

        case userv1.SyncOperation_SYNC_OPERATION_DELETE:
            err := s.userService.DeleteUser(stream.Context(), req.User.Id)
            if err != nil {
                result = &userv1.UserSyncResponse{
                    Result: userv1.SyncResult_SYNC_RESULT_ERROR,
                    Error:  ptr(err.Error()),
                }
            } else {
                result = &userv1.UserSyncResponse{
                    Result: userv1.SyncResult_SYNC_RESULT_SUCCESS,
                }
            }
        }

        if err := stream.Send(result); err != nil {
            return status.Error(codes.Internal, "failed to send sync response")
        }
    }
}

// Helper conversion functions
func toProtoUser(u *model.User) *userv1.User {
    return &userv1.User{
        Id:        u.ID,
        Email:     u.Email,
        Name:      u.Name,
        Status:    toProtoStatus(u.Status),
        Role:      toProtoRole(u.Role),
        Profile:   toProtoProfile(u.Profile),
        CreatedAt: timestamppb.New(u.CreatedAt),
        UpdatedAt: timestamppb.New(u.UpdatedAt),
    }
}

func ptr[T any](v T) *T {
    return &v
}
```

## Buf Configuration

```yaml
# buf.yaml
version: v2
modules:
  - path: api/proto
deps:
  - buf.build/googleapis/googleapis
lint:
  use:
    - STANDARD
  except:
    - PACKAGE_VERSION_SUFFIX
breaking:
  use:
    - FILE
```

```yaml
# buf.gen.yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: github.com/yourorg/yourproject/api/gen/go
plugins:
  - remote: buf.build/protocolbuffers/go
    out: api/gen/go
    opt:
      - paths=source_relative
  - remote: buf.build/grpc/go
    out: api/gen/go
    opt:
      - paths=source_relative
```

## Code Generation

```bash
# Install buf
go install github.com/bufbuild/buf/cmd/buf@latest

# Generate code
buf generate

# Lint proto files
buf lint

# Breaking change detection
buf breaking --against '.git#branch=main'
```

## Key Patterns

### Error Handling

- Use gRPC status codes (InvalidArgument, NotFound, Internal, etc.)
- Map domain errors to appropriate gRPC codes
- Include descriptive error messages
- Use structured error details for validation errors

### Pagination

- Use page_token for cursor-based pagination
- Return next_page_token and total_count
- Implement reasonable default page_size

### Field Masks

- Use google.protobuf.FieldMask for partial updates
- Validate field paths
- Only update specified fields

### Streaming

- Server streaming: One request, multiple responses
- Client streaming: Multiple requests, one response
- Bidirectional: Multiple requests and responses
- Handle context cancellation properly

## Testing

```go
func TestGetUser(t *testing.T) {
    server := NewUserServiceServer(mockUserService)

    resp, err := server.GetUser(context.Background(), &userv1.GetUserRequest{
        Id: "user-123",
    })

    assert.NoError(t, err)
    assert.NotNil(t, resp.User)
    assert.Equal(t, "user-123", resp.User.Id)
}
```
