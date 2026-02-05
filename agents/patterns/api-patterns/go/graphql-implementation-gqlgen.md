---
entity_name: GraphQL Implementation Pattern (Go)
entity_type: backend-implementation
language: go
domain: backend
description: Go implementation of GraphQL API using gqlgen with resolvers, dataloaders, authentication directives, and subscriptions
tags:
  - GraphQL
  - Go
  - gqlgen
  - resolvers
  - dataloaders
  - subscriptions
version: Go 1.21+
related_patterns:
  - GraphQL Schema Pattern
  - REST API Implementation Pattern (Go)
---

# GraphQL Implementation Pattern (Go)

This pattern demonstrates implementing a GraphQL API in Go using gqlgen (github.com/99designs/gqlgen), including resolvers, dataloaders for N+1 prevention, custom directives, and subscriptions.

## Prerequisites

```bash
go get github.com/99designs/gqlgen
go get github.com/99designs/gqlgen/graphql/handler
go get github.com/99designs/gqlgen/graphql/playground
```

## Project Structure

```
project/
├── graph/
│   ├── schema.graphqls          # GraphQL schema
│   ├── generated.go             # Generated code (don't edit)
│   ├── model/
│   │   └── models_gen.go        # Generated models
│   ├── resolver.go              # Root resolver
│   └── schema.resolvers.go      # Resolver implementations
├── internal/
│   ├── dataloader/              # DataLoaders for batching
│   │   └── loader.go
│   ├── directives/              # Custom directives
│   │   └── auth.go
│   └── service/                 # Business logic
│       └── user_service.go
├── gqlgen.yml                   # gqlgen configuration
└── server.go                    # Server setup
```

## Configuration (gqlgen.yml)

```yaml
schema:
  - graph/schema.graphqls

exec:
  filename: graph/generated.go
  package: graph

model:
  filename: graph/model/models_gen.go
  package: model

resolver:
  layout: follow-schema
  dir: graph
  package: graph
  filename_template: "{name}.resolvers.go"

# Custom scalar mappings
models:
  DateTime:
    model: time.Time
  UUID:
    model: github.com/google/uuid.UUID
  Email:
    model: string

# Skip generation for these types (provide custom implementations)
autobind:
  - "yourapp/internal/model"
```

## Root Resolver

```go
// graph/resolver.go
package graph

import (
    "yourapp/internal/dataloader"
    "yourapp/internal/service"
)

type Resolver struct {
    userService *service.UserService
    postService *service.PostService
    loaders     *dataloader.Loaders
}

func NewResolver(
    userService *service.UserService,
    postService *service.PostService,
) *Resolver {
    return &Resolver{
        userService: userService,
        postService: postService,
    }
}
```

## Query Resolvers

```go
// graph/query.resolvers.go
package graph

import (
    "context"
    "yourapp/graph/model"
)

func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
    userID := ctx.Value("user_id").(string)
    return r.userService.GetUser(ctx, userID)
}

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.userService.GetUser(ctx, id)
}

func (r *queryResolver) Users(
    ctx context.Context,
    first *int,
    after *string,
    status *model.UserStatus,
    orderBy *model.UserSortField,
    order *model.SortOrder,
) (*model.UserConnection, error) {
    limit := 20
    if first != nil {
        limit = *first
    }

    return r.userService.ListUsers(ctx, limit, after, status, orderBy, order)
}

func (r *queryResolver) SearchUsers(
    ctx context.Context,
    query string,
    first *int,
    after *string,
) (*model.UserConnection, error) {
    limit := 10
    if first != nil {
        limit = *first
    }

    return r.userService.SearchUsers(ctx, query, limit, after)
}
```

## Mutation Resolvers

```go
// graph/mutation.resolvers.go
package graph

import (
    "context"
    "yourapp/graph/model"
)

func (r *mutationResolver) CreateUser(
    ctx context.Context,
    input model.CreateUserInput,
) (model.CreateUserResult, error) {
    user, err := r.userService.CreateUser(ctx, input)
    if err != nil {
        // Return typed error
        return &model.CreateUserError{
            Message: err.Error(),
            Code:    "CREATE_FAILED",
        }, nil
    }

    return &model.CreateUserSuccess{
        User: user,
    }, nil
}

func (r *mutationResolver) UpdateUser(
    ctx context.Context,
    id string,
    input model.UpdateUserInput,
) (model.UpdateUserResult, error) {
    user, err := r.userService.UpdateUser(ctx, id, input)
    if err != nil {
        return &model.UpdateUserError{
            Message: err.Error(),
            Code:    "UPDATE_FAILED",
        }, nil
    }

    return &model.UpdateUserSuccess{
        User: user,
    }, nil
}

func (r *mutationResolver) DeleteUser(
    ctx context.Context,
    id string,
) (bool, error) {
    err := r.userService.DeleteUser(ctx, id)
    return err == nil, err
}
```

## Field Resolvers

```go
// graph/user.resolvers.go
package graph

import (
    "context"
    "yourapp/graph/model"
    "yourapp/internal/dataloader"
)

func (r *userResolver) Posts(
    ctx context.Context,
    obj *model.User,
    first *int,
    after *string,
    orderBy *model.PostSortField,
    order *model.SortOrder,
) (*model.PostConnection, error) {
    limit := 20
    if first != nil {
        limit = *first
    }

    return r.postService.ListPostsByAuthor(ctx, obj.ID, limit, after, orderBy, order)
}

// Using DataLoader to prevent N+1
func (r *postResolver) Author(
    ctx context.Context,
    obj *model.Post,
) (*model.User, error) {
    loaders := dataloader.For(ctx)
    return loaders.UserLoader.Load(ctx, obj.AuthorID)
}
```

## DataLoaders (N+1 Prevention)

```go
// internal/dataloader/loader.go
package dataloader

import (
    "context"
    "time"

    "github.com/graph-gophers/dataloader/v7"
    "yourapp/graph/model"
    "yourapp/internal/service"
)

type ctxKey string

const loadersKey = ctxKey("dataloaders")

type Loaders struct {
    UserLoader *dataloader.Loader[string, *model.User]
    PostLoader *dataloader.Loader[string, *model.Post]
}

func NewLoaders(
    userService *service.UserService,
    postService *service.PostService,
) *Loaders {
    return &Loaders{
        UserLoader: dataloader.NewBatchedLoader(
            userBatchFunc(userService),
            dataloader.WithWait[string, *model.User](time.Millisecond),
        ),
        PostLoader: dataloader.NewBatchedLoader(
            postBatchFunc(postService),
            dataloader.WithWait[string, *model.Post](time.Millisecond),
        ),
    }
}

func userBatchFunc(service *service.UserService) dataloader.BatchFunc[string, *model.User] {
    return func(ctx context.Context, keys []string) []*dataloader.Result[*model.User] {
        users, err := service.GetUsersByIDs(ctx, keys)
        if err != nil {
            results := make([]*dataloader.Result[*model.User], len(keys))
            for i := range results {
                results[i] = &dataloader.Result[*model.User]{Error: err}
            }
            return results
        }

        userMap := make(map[string]*model.User)
        for _, user := range users {
            userMap[user.ID] = user
        }

        results := make([]*dataloader.Result[*model.User], len(keys))
        for i, key := range keys {
            if user, ok := userMap[key]; ok {
                results[i] = &dataloader.Result[*model.User]{Data: user}
            } else {
                results[i] = &dataloader.Result[*model.User]{
                    Error: fmt.Errorf("user not found: %s", key),
                }
            }
        }

        return results
    }
}

// Middleware to attach loaders to context
func Middleware(loaders *Loaders) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ctx := context.WithValue(r.Context(), loadersKey, loaders)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func For(ctx context.Context) *Loaders {
    return ctx.Value(loadersKey).(*Loaders)
}
```

## Custom Directives

```go
// internal/directives/auth.go
package directives

import (
    "context"
    "fmt"

    "github.com/99designs/gqlgen/graphql"
    "yourapp/graph/model"
)

func Auth(ctx context.Context, obj interface{}, next graphql.Resolver, requires *model.Role) (interface{}, error) {
    // Check if user is authenticated
    userID := ctx.Value("user_id")
    if userID == nil {
        return nil, fmt.Errorf("unauthorized")
    }

    // If role is required, check it
    if requires != nil {
        userRole := ctx.Value("user_role")
        if userRole == nil {
            return nil, fmt.Errorf("forbidden: role required")
        }

        role := userRole.(model.Role)
        if role != *requires && role != model.RoleAdmin {
            return nil, fmt.Errorf("forbidden: insufficient permissions")
        }
    }

    return next(ctx)
}
```

## Subscriptions

```go
// graph/subscription.resolvers.go
package graph

import (
    "context"
    "time"

    "yourapp/graph/model"
)

func (r *subscriptionResolver) UserUpdated(
    ctx context.Context,
    userID string,
) (<-chan *model.User, error) {
    ch := make(chan *model.User, 1)

    go func() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        defer close(ch)

        for {
            select {
            case <-ticker.C:
                user, err := r.userService.GetUser(ctx, userID)
                if err != nil {
                    return
                }
                ch <- user
            case <-ctx.Done():
                return
            }
        }
    }()

    return ch, nil
}

func (r *subscriptionResolver) PostCreated(
    ctx context.Context,
    authorID *string,
) (<-chan *model.Post, error) {
    ch := make(chan *model.Post, 1)

    // Subscribe to post creation events
    r.postService.SubscribeToCreated(ctx, authorID, ch)

    return ch, nil
}
```

## Server Setup

```go
// server.go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/99designs/gqlgen/graphql/handler"
    "github.com/99designs/gqlgen/graphql/handler/transport"
    "github.com/99designs/gqlgen/graphql/playground"
    "github.com/gorilla/websocket"
    "yourapp/graph"
    "yourapp/internal/dataloader"
    "yourapp/internal/directives"
    "yourapp/internal/middleware"
    "yourapp/internal/service"
)

func main() {
    // Initialize services
    userService := service.NewUserService()
    postService := service.NewPostService()

    // Create resolver
    resolver := graph.NewResolver(userService, postService)

    // Create dataloaders
    loaders := dataloader.NewLoaders(userService, postService)

    // Configure server
    srv := handler.NewDefaultServer(
        graph.NewExecutableSchema(graph.Config{
            Resolvers: resolver,
            Directives: graph.DirectiveRoot{
                Auth: directives.Auth,
            },
        }),
    )

    // Add WebSocket support for subscriptions
    srv.AddTransport(&transport.Websocket{
        Upgrader: websocket.Upgrader{
            CheckOrigin: func(r *http.Request) bool {
                return true // Configure properly in production
            },
        },
    })

    // Setup routes
    http.Handle("/", playground.Handler("GraphQL Playground", "/query"))
    http.Handle("/query",
        middleware.AuthMiddleware(
            dataloader.Middleware(loaders)(srv),
        ),
    )

    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    log.Printf("Server running on http://localhost:%s/", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

## Authentication Middleware

```go
// internal/middleware/auth.go
package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            next.ServeHTTP(w, r)
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err == nil && token.Valid {
            claims := token.Claims.(jwt.MapClaims)
            ctx := context.WithValue(r.Context(), "user_id", claims["user_id"])
            ctx = context.WithValue(ctx, "user_role", claims["role"])
            r = r.WithContext(ctx)
        }

        next.ServeHTTP(w, r)
    })
}
```

## Code Generation

```bash
# Initialize gqlgen project
go run github.com/99designs/gqlgen init

# Generate code after schema changes
go run github.com/99designs/gqlgen generate

# Watch for changes and regenerate
go run github.com/99designs/gqlgen generate --watch
```

## Testing

```go
// graph/resolver_test.go
package graph_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "yourapp/graph"
    "yourapp/internal/service/mock"
)

func TestQueryMe(t *testing.T) {
    mockUserService := mock.NewUserService()
    resolver := graph.NewResolver(mockUserService, nil)

    ctx := context.WithValue(context.Background(), "user_id", "user-123")

    user, err := resolver.Query().Me(ctx)

    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "user-123", user.ID)
}

func TestMutationCreateUser(t *testing.T) {
    mockUserService := mock.NewUserService()
    resolver := graph.NewResolver(mockUserService, nil)

    input := model.CreateUserInput{
        Email:    "test@example.com",
        Name:     "Test User",
        Password: "password123",
    }

    result, err := resolver.Mutation().CreateUser(context.Background(), input)

    assert.NoError(t, err)

    switch r := result.(type) {
    case *model.CreateUserSuccess:
        assert.Equal(t, input.Email, r.User.Email)
    case *model.CreateUserError:
        t.Fatalf("Expected success, got error: %s", r.Message)
    }
}
```

## Best Practices

### 1. Use DataLoaders

Prevent N+1 queries by batching database requests:

```go
// BAD: N+1 query
for _, post := range posts {
    author := getAuthor(post.AuthorID) // Separate query for each post
}

// GOOD: Using DataLoader
loaders.UserLoader.Load(ctx, post.AuthorID) // Batched in single query
```

### 2. Context for Auth

Always store auth info in context:

```go
ctx := context.WithValue(r.Context(), "user_id", userID)
ctx = context.WithValue(ctx, "role", role)
```

### 3. Error Handling

Use union types for business errors:

```graphql
union CreateUserResult = CreateUserSuccess | CreateUserError
```

Return errors as data, not exceptions:

```go
if err != nil {
    return &model.CreateUserError{Code: "VALIDATION_ERROR", Message: err.Error()}, nil
}
```

### 4. Subscriptions Cleanup

Always handle context cancellation:

```go
for {
    select {
    case <-ticker.C:
        // Send update
    case <-ctx.Done():
        return // Clean up when client disconnects
    }
}
```

### 5. Input Validation

Validate inputs in resolvers:

```go
func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) {
    if !isValidEmail(input.Email) {
        return nil, fmt.Errorf("invalid email format")
    }
    // Process...
}
```

## Performance Tips

1. **Use DataLoaders** for all foreign key relationships
2. **Limit query depth** to prevent expensive nested queries
3. **Implement query cost analysis** to prevent abuse
4. **Cache frequently accessed data** at resolver level
5. **Use database indexes** for filter and sort fields
6. **Configure appropriate timeouts** for resolvers
7. **Monitor resolver performance** with tracing
