---
entity_name: gRPC Implementation Pattern (Go)
entity_type: backend-implementation
language: go
domain: backend
description: Go implementation of gRPC services with interceptors, streaming, error handling, authentication, and testing
tags:
  - gRPC
  - Go
  - Protocol Buffers
  - protobuf
  - streaming
  - interceptors
  - middleware
version: Go 1.21+
related_patterns:
  - gRPC Service Definition Pattern
  - REST API Implementation Pattern (Go)
---

# gRPC Implementation Pattern (Go)

This pattern demonstrates implementing gRPC services in Go using google.golang.org/grpc, including server/client setup, interceptors, streaming, and error handling.

## Prerequisites

```bash
go get google.golang.org/grpc
go get google.golang.org/protobuf
go get google.golang.org/grpc/codes
go get google.golang.org/grpc/status
```

## Project Structure

```
project/
├── api/
│   ├── proto/v1/
│   │   └── user_service.proto
│   └── gen/go/user/v1/
│       ├── user_service.pb.go        # Generated protobuf code
│       └── user_service_grpc.pb.go   # Generated gRPC code
├── internal/
│   ├── server/
│   │   └── user_server.go            # gRPC server implementation
│   ├── interceptor/
│   │   ├── auth.go                   # Auth interceptor
│   │   └── logging.go                # Logging interceptor
│   └── service/
│       └── user_service.go           # Business logic
├── cmd/
│   └── server/
│       └── main.go                   # Server entrypoint
└── buf.gen.yaml                      # Buf configuration
```

## Server Implementation

```go
// internal/server/user_server.go
package server

import (
    "context"
    "fmt"
    "io"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/emptypb"
    "google.golang.org/protobuf/types/known/fieldmaskpb"

    userv1 "yourapp/api/gen/go/user/v1"
    "yourapp/internal/service"
)

type UserServer struct {
    userv1.UnimplementedUserServiceServer
    userService *service.UserService
}

func NewUserServer(userService *service.UserService) *UserServer {
    return &UserServer{
        userService: userService,
    }
}

// Unary RPC: GetUser
func (s *UserServer) GetUser(
    ctx context.Context,
    req *userv1.GetUserRequest,
) (*userv1.GetUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    user, err := s.userService.GetUser(ctx, req.Id)
    if err != nil {
        return nil, toGRPCError(err)
    }

    return &userv1.GetUserResponse{User: user}, nil
}

// Unary RPC: ListUsers with pagination
func (s *UserServer) ListUsers(
    ctx context.Context,
    req *userv1.ListUsersRequest,
) (*userv1.ListUsersResponse, error) {
    pageSize := req.PageSize
    if pageSize == 0 {
        pageSize = 20
    }
    if pageSize > 100 {
        pageSize = 100
    }

    users, nextToken, total, err := s.userService.ListUsers(
        ctx,
        int(pageSize),
        req.PageToken,
        req.Status,
    )
    if err != nil {
        return nil, toGRPCError(err)
    }

    return &userv1.ListUsersResponse{
        Users:         users,
        NextPageToken: nextToken,
        TotalSize:     int32(total),
    }, nil
}

// Unary RPC: CreateUser
func (s *UserServer) CreateUser(
    ctx context.Context,
    req *userv1.CreateUserRequest,
) (*userv1.CreateUserResponse, error) {
    if err := validateCreateUserRequest(req); err != nil {
        return nil, err
    }

    user, err := s.userService.CreateUser(ctx, req)
    if err != nil {
        return nil, toGRPCError(err)
    }

    return &userv1.CreateUserResponse{User: user}, nil
}

// Unary RPC: UpdateUser with field mask
func (s *UserServer) UpdateUser(
    ctx context.Context,
    req *userv1.UpdateUserRequest,
) (*userv1.UpdateUserResponse, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    // Apply field mask to only update specified fields
    fieldsToUpdate := getFieldsFromMask(req.UpdateMask)

    user, err := s.userService.UpdateUser(ctx, req.Id, req.User, fieldsToUpdate)
    if err != nil {
        return nil, toGRPCError(err)
    }

    return &userv1.UpdateUserResponse{User: user}, nil
}

// Unary RPC: DeleteUser
func (s *UserServer) DeleteUser(
    ctx context.Context,
    req *userv1.DeleteUserRequest,
) (*emptypb.Empty, error) {
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "user ID is required")
    }

    if err := s.userService.DeleteUser(ctx, req.Id); err != nil {
        return nil, toGRPCError(err)
    }

    return &emptypb.Empty{}, nil
}

// Server Streaming: StreamUserEvents
func (s *UserServer) StreamUserEvents(
    req *userv1.StreamUserEventsRequest,
    stream userv1.UserService_StreamUserEventsServer,
) error {
    ctx := stream.Context()

    // Subscribe to events
    events, err := s.userService.SubscribeToEvents(ctx, req.UserIds)
    if err != nil {
        return toGRPCError(err)
    }

    // Stream events to client
    for {
        select {
        case event := <-events:
            if err := stream.Send(event); err != nil {
                return err
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

// Client Streaming: BatchCreateUsers
func (s *UserServer) BatchCreateUsers(
    stream userv1.UserService_BatchCreateUsersServer,
) error {
    var users []*userv1.User
    var errors []*userv1.Error

    for {
        req, err := stream.Recv()
        if err == io.EOF {
            // Client finished sending
            return stream.SendAndClose(&userv1.BatchCreateUsersResponse{
                Users:  users,
                Errors: errors,
            })
        }
        if err != nil {
            return err
        }

        // Process each user
        user, err := s.userService.CreateUser(stream.Context(), req)
        if err != nil {
            errors = append(errors, &userv1.Error{
                Code:    userv1.Error_CODE_INTERNAL,
                Message: err.Error(),
            })
        } else {
            users = append(users, user)
        }
    }
}

// Bidirectional Streaming: SyncUsers
func (s *UserServer) SyncUsers(
    stream userv1.UserService_SyncUsersServer,
) error {
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return err
        }

        var response *userv1.UserSyncResponse

        switch r := req.Request.(type) {
        case *userv1.UserSyncRequest_UserUpdate:
            _, err := s.userService.UpdateUser(stream.Context(), r.UserUpdate.Id, r.UserUpdate, nil)
            if err != nil {
                response = &userv1.UserSyncResponse{
                    Success: false,
                    Error: &userv1.Error{
                        Code:    userv1.Error_CODE_INTERNAL,
                        Message: err.Error(),
                    },
                }
            } else {
                response = &userv1.UserSyncResponse{Success: true}
            }

        case *userv1.UserSyncRequest_UserIdDelete:
            err := s.userService.DeleteUser(stream.Context(), r.UserIdDelete)
            if err != nil {
                response = &userv1.UserSyncResponse{
                    Success: false,
                    Error: &userv1.Error{
                        Code:    userv1.Error_CODE_INTERNAL,
                        Message: err.Error(),
                    },
                }
            } else {
                response = &userv1.UserSyncResponse{Success: true}
            }
        }

        if err := stream.Send(response); err != nil {
            return err
        }
    }
}

// Helper functions
func validateCreateUserRequest(req *userv1.CreateUserRequest) error {
    if req.Email == "" {
        return status.Error(codes.InvalidArgument, "email is required")
    }
    if req.Name == "" {
        return status.Error(codes.InvalidArgument, "name is required")
    }
    return nil
}

func getFieldsFromMask(mask *fieldmaskpb.FieldMask) []string {
    if mask == nil {
        return nil
    }
    return mask.Paths
}

func toGRPCError(err error) error {
    // Map service errors to gRPC status codes
    switch {
    case errors.Is(err, service.ErrNotFound):
        return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, service.ErrAlreadyExists):
        return status.Error(codes.AlreadyExists, err.Error())
    case errors.Is(err, service.ErrInvalidArgument):
        return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, service.ErrPermissionDenied):
        return status.Error(codes.PermissionDenied, err.Error())
    default:
        return status.Error(codes.Internal, "internal server error")
    }
}
```

## Server Setup

```go
// cmd/server/main.go
package main

import (
    "log"
    "net"

    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"

    userv1 "yourapp/api/gen/go/user/v1"
    "yourapp/internal/interceptor"
    "yourapp/internal/server"
    "yourapp/internal/service"
)

func main() {
    // Create listener
    lis, err := net.Listen("tcp", ":50051")
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    // Create gRPC server with interceptors
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            interceptor.LoggingUnaryInterceptor(),
            interceptor.AuthUnaryInterceptor(),
        ),
        grpc.ChainStreamInterceptor(
            interceptor.LoggingStreamInterceptor(),
            interceptor.AuthStreamInterceptor(),
        ),
    )

    // Register services
    userService := service.NewUserService()
    userServer := server.NewUserServer(userService)
    userv1.RegisterUserServiceServer(grpcServer, userServer)

    // Enable reflection for grpcurl
    reflection.Register(grpcServer)

    log.Printf("gRPC server listening on :50051")
    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

## Interceptors (Middleware)

### Logging Interceptor

```go
// internal/interceptor/logging.go
package interceptor

import (
    "context"
    "log"
    "time"

    "google.golang.org/grpc"
)

func LoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()

        resp, err := handler(ctx, req)

        log.Printf(
            "method=%s duration=%s error=%v",
            info.FullMethod,
            time.Since(start),
            err,
        )

        return resp, err
    }
}

func LoggingStreamInterceptor() grpc.StreamServerInterceptor {
    return func(
        srv interface{},
        stream grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        start := time.Now()

        err := handler(srv, stream)

        log.Printf(
            "method=%s duration=%s error=%v",
            info.FullMethod,
            time.Since(start),
            err,
        )

        return err
    }
}
```

### Auth Interceptor

```go
// internal/interceptor/auth.go
package interceptor

import (
    "context"
    "strings"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"
)

func AuthUnaryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // Skip auth for certain methods
        if isPublicMethod(info.FullMethod) {
            return handler(ctx, req)
        }

        // Validate token
        ctx, err := authorize(ctx)
        if err != nil {
            return nil, err
        }

        return handler(ctx, req)
    }
}

func AuthStreamInterceptor() grpc.StreamServerInterceptor {
    return func(
        srv interface{},
        stream grpc.ServerStream,
        info *grpc.StreamServerInfo,
        handler grpc.StreamHandler,
    ) error {
        if isPublicMethod(info.FullMethod) {
            return handler(srv, stream)
        }

        ctx, err := authorize(stream.Context())
        if err != nil {
            return err
        }

        wrapped := &wrappedStream{stream, ctx}
        return handler(srv, wrapped)
    }
}

func authorize(ctx context.Context) (context.Context, error) {
    md, ok := metadata.FromIncomingContext(ctx)
    if !ok {
        return nil, status.Error(codes.Unauthenticated, "missing metadata")
    }

    values := md.Get("authorization")
    if len(values) == 0 {
        return nil, status.Error(codes.Unauthenticated, "missing authorization")
    }

    token := strings.TrimPrefix(values[0], "Bearer ")

    // Validate token (implement your JWT validation here)
    userID, err := validateToken(token)
    if err != nil {
        return nil, status.Error(codes.Unauthenticated, "invalid token")
    }

    // Add user info to context
    ctx = context.WithValue(ctx, "user_id", userID)
    return ctx, nil
}

func isPublicMethod(method string) bool {
    publicMethods := []string{
        "/user.v1.UserService/GetUser", // Example public method
    }

    for _, pm := range publicMethods {
        if method == pm {
            return true
        }
    }
    return false
}

type wrappedStream struct {
    grpc.ServerStream
    ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
    return w.ctx
}
```

## Client Implementation

```go
// client/user_client.go
package client

import (
    "context"
    "log"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"

    userv1 "yourapp/api/gen/go/user/v1"
)

type UserClient struct {
    client userv1.UserServiceClient
    conn   *grpc.ClientConn
}

func NewUserClient(address string) (*UserClient, error) {
    conn, err := grpc.Dial(
        address,
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithChainUnaryInterceptor(
            clientLoggingInterceptor(),
        ),
    )
    if err != nil {
        return nil, err
    }

    return &UserClient{
        client: userv1.NewUserServiceClient(conn),
        conn:   conn,
    }, nil
}

func (c *UserClient) Close() error {
    return c.conn.Close()
}

func (c *UserClient) GetUser(ctx context.Context, userID, token string) (*userv1.User, error) {
    // Add auth token to metadata
    ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)

    resp, err := c.client.GetUser(ctx, &userv1.GetUserRequest{
        Id: userID,
    })
    if err != nil {
        return nil, err
    }

    return resp.User, nil
}

func (c *UserClient) ListUsers(
    ctx context.Context,
    pageSize int32,
    pageToken string,
) ([]*userv1.User, string, error) {
    resp, err := c.client.ListUsers(ctx, &userv1.ListUsersRequest{
        PageSize:  pageSize,
        PageToken: pageToken,
    })
    if err != nil {
        return nil, "", err
    }

    return resp.Users, resp.NextPageToken, nil
}

func clientLoggingInterceptor() grpc.UnaryClientInterceptor {
    return func(
        ctx context.Context,
        method string,
        req, reply interface{},
        cc *grpc.ClientConn,
        invoker grpc.UnaryInvoker,
        opts ...grpc.CallOption,
    ) error {
        log.Printf("Calling %s", method)
        err := invoker(ctx, method, req, reply, cc, opts...)
        if err != nil {
            log.Printf("Error: %v", err)
        }
        return err
    }
}
```

## Testing

```go
// internal/server/user_server_test.go
package server_test

import (
    "context"
    "testing"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    userv1 "yourapp/api/gen/go/user/v1"
    "yourapp/internal/server"
    "yourapp/internal/service/mock"
)

func TestGetUser(t *testing.T) {
    mockService := mock.NewUserService()
    srv := server.NewUserServer(mockService)

    resp, err := srv.GetUser(context.Background(), &userv1.GetUserRequest{
        Id: "user-123",
    })

    if err != nil {
        t.Fatalf("GetUser failed: %v", err)
    }

    if resp.User.Id != "user-123" {
        t.Errorf("Expected user ID user-123, got %s", resp.User.Id)
    }
}

func TestGetUser_NotFound(t *testing.T) {
    mockService := mock.NewUserService()
    srv := server.NewUserServer(mockService)

    _, err := srv.GetUser(context.Background(), &userv1.GetUserRequest{
        Id: "non-existent",
    })

    if err == nil {
        t.Fatal("Expected error, got nil")
    }

    st, ok := status.FromError(err)
    if !ok {
        t.Fatalf("Expected gRPC status error")
    }

    if st.Code() != codes.NotFound {
        t.Errorf("Expected NotFound, got %v", st.Code())
    }
}
```

## Best Practices

1. **Always use context** for cancellation and timeouts
2. **Validate all inputs** in server methods
3. **Use interceptors** for cross-cutting concerns
4. **Handle streaming errors** properly
5. **Return proper status codes** (use `google.golang.org/grpc/codes`)
6. **Enable reflection** for development (grpcurl, grpcui)
7. **Use field masks** for partial updates
8. **Implement health checks** (grpc_health_v1)
9. **Add metrics and tracing** (OpenTelemetry)
10. **Use connection pooling** for clients
