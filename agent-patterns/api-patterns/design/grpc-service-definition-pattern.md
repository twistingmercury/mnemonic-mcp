---
entity_name: gRPC Service Definition Pattern
entity_type: api-specification
language: agnostic
domain: api-design
description: Comprehensive Protocol Buffer service definition pattern with unary, streaming RPCs, pagination, error handling, and versioning
tags:
  - gRPC
  - Protocol Buffers
  - protobuf
  - API design
  - streaming
  - RPC
version: proto3
related_patterns:
  - gRPC Implementation Pattern (Go)
  - REST API Specification Pattern
---

# gRPC Service Definition Pattern

This pattern provides complete Protocol Buffer (protobuf) definitions for gRPC services. Proto definitions are language-agnostic and can be compiled to any supported language (Go, Python, Java, C++, etc.).

## Complete Proto File

```protobuf
// api/proto/v1/user_service.proto
syntax = "proto3";

package user.v1;

// Language-specific package options
option go_package = "github.com/yourorg/yourproject/api/gen/go/user/v1;userv1";
option java_package = "com.yourorg.yourproject.user.v1";
option java_multiple_files = true;
option csharp_namespace = "YourOrg.YourProject.User.V1";

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

// User represents a user entity
message User {
  string id = 1;
  string email = 2;
  string name = 3;
  UserStatus status = 4;
  google.protobuf.Timestamp created_at = 5;
  google.protobuf.Timestamp updated_at = 6;
}

// UserStatus enum
enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;
  USER_STATUS_ACTIVE = 1;
  USER_STATUS_INACTIVE = 2;
  USER_STATUS_SUSPENDED = 3;
}

// Request messages
message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

message ListUsersRequest {
  int32 page_size = 1;    // Number of users to return
  string page_token = 2;   // Token from previous response
  UserStatus status = 3;   // Filter by status
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;
  int32 total_size = 3;
}

message CreateUserRequest {
  string email = 1;
  string name = 2;
  string password = 3;
}

message CreateUserResponse {
  User user = 1;
}

message UpdateUserRequest {
  string id = 1;
  User user = 2;
  google.protobuf.FieldMask update_mask = 3;  // Fields to update
}

message UpdateUserResponse {
  User user = 1;
}

message DeleteUserRequest {
  string id = 1;
}

// Streaming messages
message StreamUserEventsRequest {
  repeated string user_ids = 1;  // Filter by user IDs (empty = all)
}

message UserEvent {
  enum EventType {
    EVENT_TYPE_UNSPECIFIED = 0;
    EVENT_TYPE_CREATED = 1;
    EVENT_TYPE_UPDATED = 2;
    EVENT_TYPE_DELETED = 3;
  }

  EventType type = 1;
  User user = 2;
  google.protobuf.Timestamp timestamp = 3;
}

message BatchCreateUsersResponse {
  repeated User users = 1;
  repeated Error errors = 2;
}

message UserSyncRequest {
  oneof request {
    User user_update = 1;
    string user_id_delete = 2;
  }
}

message UserSyncResponse {
  bool success = 1;
  Error error = 2;
}

// Error message
message Error {
  enum Code {
    CODE_UNSPECIFIED = 0;
    CODE_NOT_FOUND = 1;
    CODE_ALREADY_EXISTS = 2;
    CODE_INVALID_ARGUMENT = 3;
    CODE_PERMISSION_DENIED = 4;
    CODE_INTERNAL = 5;
  }

  Code code = 1;
  string message = 2;
  map<string, string> details = 3;
}
```

## Key Patterns

### 1. RPC Types

#### Unary RPC (Request-Response)

**Use for:** Standard CRUD operations

```protobuf
rpc GetUser(GetUserRequest) returns (GetUserResponse);
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
```

**Characteristics:**
- Client sends single request
- Server sends single response
- Similar to REST API call

#### Server Streaming RPC

**Use for:** Server sending multiple responses

```protobuf
rpc StreamUserEvents(StreamUserEventsRequest) returns (stream UserEvent);
```

**Characteristics:**
- Client sends single request
- Server streams multiple responses
- Good for real-time updates, logs, notifications

#### Client Streaming RPC

**Use for:** Client sending multiple requests

```protobuf
rpc BatchCreateUsers(stream CreateUserRequest) returns (BatchCreateUsersResponse);
```

**Characteristics:**
- Client streams multiple requests
- Server sends single response
- Good for bulk uploads, batch processing

#### Bidirectional Streaming RPC

**Use for:** Both sides streaming

```protobuf
rpc SyncUsers(stream UserSyncRequest) returns (stream UserSyncResponse);
```

**Characteristics:**
- Client and server both stream
- Order independent
- Good for chat, real-time collaboration

### 2. Pagination

**Token-Based Pagination:**

```protobuf
message ListUsersRequest {
  int32 page_size = 1;     // Items per page
  string page_token = 2;   // Opaque continuation token
}

message ListUsersResponse {
  repeated User users = 1;
  string next_page_token = 2;  // Token for next page (empty if last page)
  int32 total_size = 3;        // Total number of items
}
```

**Benefits:**
- Stable across data changes
- Efficient for large datasets
- Standard pattern across Google APIs

### 3. Partial Updates (FieldMask)

**Update only specified fields:**

```protobuf
import "google/protobuf/field_mask.proto";

message UpdateUserRequest {
  string id = 1;
  User user = 2;
  google.protobuf.FieldMask update_mask = 3;
}
```

**Example Usage:**
```json
{
  "id": "user-123",
  "user": {
    "name": "Updated Name",
    "email": "new@example.com"
  },
  "update_mask": {
    "paths": ["name", "email"]
  }
}
```

**Benefits:**
- Explicit about what changes
- Prevents accidental overwrites
- Bandwidth efficient

### 4. Error Handling

**Structured Errors:**

```protobuf
message Error {
  enum Code {
    CODE_UNSPECIFIED = 0;
    CODE_NOT_FOUND = 1;
    CODE_ALREADY_EXISTS = 2;
    CODE_INVALID_ARGUMENT = 3;
    CODE_PERMISSION_DENIED = 4;
    CODE_INTERNAL = 5;
  }

  Code code = 1;
  string message = 2;
  map<string, string> details = 3;  // Additional context
}
```

**Use in responses:**
```protobuf
message CreateUserResponse {
  oneof result {
    User user = 1;
    Error error = 2;
  }
}
```

### 5. Versioning

**Package Versioning:**

```protobuf
// v1/user_service.proto
syntax = "proto3";
package user.v1;

// v2/user_service.proto
syntax = "proto3";
package user.v2;
```

**Benefits:**
- Multiple versions can coexist
- Clear separation of versions
- Gradual migration path

### 6. Enum Best Practices

**Always include UNSPECIFIED:**

```protobuf
enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;  // Default value
  USER_STATUS_ACTIVE = 1;
  USER_STATUS_INACTIVE = 2;
  USER_STATUS_SUSPENDED = 3;
}
```

**Naming:**
- Prefix with message name (`USER_STATUS_`)
- 0 value is always `*_UNSPECIFIED`
- Use UPPER_SNAKE_CASE

### 7. Oneof for Variants

**Mutually exclusive fields:**

```protobuf
message UserSyncRequest {
  oneof request {
    User user_update = 1;
    string user_id_delete = 2;
  }
}
```

**Benefits:**
- Type-safe variants
- Only one field can be set
- Efficient encoding

## Design Best Practices

### Naming Conventions

1. **Services:** PascalCase with "Service" suffix - `UserService`
2. **RPCs:** PascalCase verbs - `GetUser`, `ListUsers`
3. **Messages:** PascalCase - `GetUserRequest`, `User`
4. **Fields:** snake_case - `user_id`, `created_at`
5. **Enums:** UPPER_SNAKE_CASE - `USER_STATUS_ACTIVE`

### Request/Response Naming

```protobuf
// Pattern: {RpcName}Request / {RpcName}Response
rpc GetUser(GetUserRequest) returns (GetUserResponse);
rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
```

### Reserved Fields

**Reserve deprecated fields:**

```protobuf
message User {
  reserved 2, 15, 9 to 11;
  reserved "old_field_name", "deprecated_field";

  string id = 1;
  // field 2 is reserved
  string name = 3;
}
```

**Benefits:**
- Prevents field number reuse
- Documents removed fields
- Avoids compatibility issues

### Comments and Documentation

```protobuf
// UserService provides user management operations.
//
// This service handles CRUD operations for users and supports
// real-time updates via streaming RPCs.
service UserService {
  // GetUser retrieves a user by ID.
  //
  // Returns NOT_FOUND error if user doesn't exist.
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
}

// User represents a user entity.
message User {
  // Unique identifier for the user.
  string id = 1;

  // User's email address (must be unique).
  string email = 2;
}
```

## Code Generation

### Generate for Go

```bash
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  api/proto/v1/user_service.proto
```

### Generate for Python

```bash
python -m grpc_tools.protoc -I. \
  --python_out=. \
  --grpc_python_out=. \
  api/proto/v1/user_service.proto
```

### Generate for Multiple Languages

```bash
# Using buf (recommended)
buf generate
```

**buf.gen.yaml:**
```yaml
version: v1
plugins:
  - name: go
    out: gen/go
    opt: paths=source_relative
  - name: go-grpc
    out: gen/go
    opt: paths=source_relative
  - name: python
    out: gen/python
  - name: python-grpc
    out: gen/python
```

## Validation with buf

**buf.yaml:**
```yaml
version: v1
lint:
  use:
    - DEFAULT
  except:
    - PACKAGE_VERSION_SUFFIX  # If not using version suffixes
breaking:
  use:
    - FILE
```

```bash
# Lint proto files
buf lint

# Check for breaking changes
buf breaking --against '.git#branch=main'

# Format proto files
buf format -w
```

## Proto Organization

```
api/
  proto/
    v1/
      user_service.proto
      post_service.proto
      common/
        pagination.proto
        errors.proto
        types.proto
    v2/
      user_service.proto
```

**Import common definitions:**
```protobuf
import "api/proto/v1/common/pagination.proto";
import "api/proto/v1/common/errors.proto";
```

## Well-Known Types

Use Google's well-known types:

```protobuf
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/empty.proto";
import "google/protobuf/field_mask.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/wrappers.proto";

message User {
  google.protobuf.Timestamp created_at = 1;
  google.protobuf.Duration timeout = 2;
}
```

**Benefits:**
- Standard serialization across languages
- Built-in validation
- Well-documented behavior
