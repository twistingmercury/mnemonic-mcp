---
entity_name: GraphQL Schema Pattern
entity_type: GraphQL Pattern
language: go
domain: backend
description: Comprehensive GraphQL schema pattern with queries, mutations, subscriptions, interfaces, and error handling for Go services
tags:
  - GraphQL
  - gqlgen
  - schema
  - resolvers
  - subscriptions
---

# GraphQL Schema Pattern

This pattern provides a complete GraphQL schema template for Go-based GraphQL APIs using gqlgen.

## Complete GraphQL Schema

```graphql
# schema.graphql

# Scalar Types
scalar DateTime
scalar UUID
scalar Email
scalar JSON

# Directives
directive @auth(requires: Role = USER) on OBJECT | FIELD_DEFINITION
directive @deprecated(reason: String) on FIELD_DEFINITION | ENUM_VALUE
directive @rateLimit(limit: Int!, duration: Int!) on FIELD_DEFINITION

# Enums
enum Role {
  ADMIN
  USER
  GUEST
}

enum UserStatus {
  ACTIVE
  INACTIVE
  SUSPENDED
}

enum SortOrder {
  ASC
  DESC
}

enum UserSortField {
  CREATED_AT
  UPDATED_AT
  NAME
  EMAIL
}

# Interfaces
interface Node {
  id: ID!
  createdAt: DateTime!
  updatedAt: DateTime!
}

interface Error {
  message: String!
  code: String!
}

# Types
type User implements Node {
  id: ID!
  email: Email!
  name: String!
  status: UserStatus!
  role: Role!
  profile: UserProfile
  posts(
    first: Int
    after: String
    orderBy: PostSortField
    order: SortOrder
  ): PostConnection!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type UserProfile {
  bio: String
  avatar: String
  location: String
  website: String
}

type Post implements Node {
  id: ID!
  title: String!
  content: String!
  published: Boolean!
  author: User!
  tags: [String!]!
  comments(first: Int, after: String): CommentConnection!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type Comment implements Node {
  id: ID!
  content: String!
  author: User!
  post: Post!
  createdAt: DateTime!
  updatedAt: DateTime!
}

# Pagination (Relay Cursor Connections)
type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type UserEdge {
  cursor: String!
  node: User!
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type PostEdge {
  cursor: String!
  node: Post!
}

type PostConnection {
  edges: [PostEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type CommentEdge {
  cursor: String!
  node: Comment!
}

type CommentConnection {
  edges: [CommentEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

# Input Types
input CreateUserInput {
  email: Email!
  name: String!
  password: String!
  profile: UserProfileInput
}

input UpdateUserInput {
  email: Email
  name: String
  status: UserStatus
  profile: UserProfileInput
}

input UserProfileInput {
  bio: String
  avatar: String
  location: String
  website: String
}

input CreatePostInput {
  title: String!
  content: String!
  published: Boolean
  tags: [String!]
}

input UpdatePostInput {
  title: String
  content: String
  published: Boolean
  tags: [String!]
}

input CreateCommentInput {
  postId: ID!
  content: String!
}

input UserFilterInput {
  status: UserStatus
  role: Role
  search: String
}

input PostFilterInput {
  published: Boolean
  authorId: ID
  tags: [String!]
  search: String
}

# Error Types
type ValidationError implements Error {
  message: String!
  code: String!
  field: String!
}

type NotFoundError implements Error {
  message: String!
  code: String!
  resourceType: String!
  resourceId: ID!
}

type AuthenticationError implements Error {
  message: String!
  code: String!
}

type AuthorizationError implements Error {
  message: String!
  code: String!
  requiredRole: Role!
}

# Union Types for Results
union CreateUserResult = User | ValidationError | AuthenticationError
union UpdateUserResult = User | ValidationError | NotFoundError | AuthorizationError
union DeleteUserResult = SuccessResult | NotFoundError | AuthorizationError

type SuccessResult {
  success: Boolean!
  message: String
}

# Query Root
type Query {
  # Node interface
  node(id: ID!): Node

  # User queries
  me: User @auth
  user(id: ID!): User
  users(
    first: Int = 20
    after: String
    filter: UserFilterInput
    orderBy: UserSortField = CREATED_AT
    order: SortOrder = DESC
  ): UserConnection!

  # Post queries
  post(id: ID!): Post
  posts(
    first: Int = 20
    after: String
    filter: PostFilterInput
  ): PostConnection!

  # Search
  search(query: String!, first: Int = 20): SearchResult!

  # Health check
  health: HealthStatus!
}

# Mutation Root
type Mutation {
  # User mutations
  createUser(input: CreateUserInput!): CreateUserResult! @rateLimit(limit: 5, duration: 60)
  updateUser(id: ID!, input: UpdateUserInput!): UpdateUserResult! @auth
  deleteUser(id: ID!): DeleteUserResult! @auth(requires: ADMIN)

  # Post mutations
  createPost(input: CreatePostInput!): Post! @auth
  updatePost(id: ID!, input: UpdatePostInput!): Post! @auth
  deletePost(id: ID!): SuccessResult! @auth
  publishPost(id: ID!): Post! @auth

  # Comment mutations
  createComment(input: CreateCommentInput!): Comment! @auth
  deleteComment(id: ID!): SuccessResult! @auth

  # Authentication
  login(email: Email!, password: String!): AuthPayload!
  refreshToken(refreshToken: String!): AuthPayload!
  logout: SuccessResult! @auth
}

# Subscription Root
type Subscription {
  # Post subscriptions
  postCreated: Post!
  postUpdated(id: ID!): Post!

  # Comment subscriptions
  commentAdded(postId: ID!): Comment!

  # User subscriptions
  userStatusChanged(userId: ID!): User! @auth
}

# Search Result
union SearchResultItem = User | Post | Comment

type SearchResult {
  items: [SearchResultItem!]!
  totalCount: Int!
}

# Authentication
type AuthPayload {
  accessToken: String!
  refreshToken: String!
  user: User!
  expiresIn: Int!
}

# Health Check
type HealthStatus {
  status: String!
  timestamp: DateTime!
  version: String!
}
```

## gqlgen Configuration

```yaml
# gqlgen.yml
schema:
  - schema.graphql

exec:
  filename: internal/graph/generated.go
  package: graph

model:
  filename: internal/graph/model/models_gen.go
  package: model

resolver:
  filename: internal/graph/resolver.go
  type: Resolver
  package: graph

# Custom scalar mappings
models:
  ID:
    model:
      - github.com/99designs/gqlgen/graphql.ID
  DateTime:
    model:
      - github.com/99designs/gqlgen/graphql.Time
  UUID:
    model:
      - github.com/google/uuid.UUID
  Email:
    model:
      - github.com/yourorg/yourproject/internal/types.Email
  JSON:
    model:
      - github.com/99designs/gqlgen/graphql.Map

# Skip generating models for existing types
autobind:
  - github.com/yourorg/yourproject/internal/model

# Custom directives
directives:
  auth:
    skip_runtime: false
  rateLimit:
    skip_runtime: false
```

## Resolver Implementation Example

```go
// internal/graph/resolver.go
package graph

import (
    "context"
    "github.com/yourorg/yourproject/internal/service"
    "github.com/yourorg/yourproject/internal/graph/model"
)

type Resolver struct {
    userService    *service.UserService
    postService    *service.PostService
    commentService *service.CommentService
    authService    *service.AuthService
}

// Query resolver
func (r *Resolver) Query() QueryResolver {
    return &queryResolver{r}
}

// Mutation resolver
func (r *Resolver) Mutation() MutationResolver {
    return &mutationResolver{r}
}

// Subscription resolver
func (r *Resolver) Subscription() SubscriptionResolver {
    return &subscriptionResolver{r}
}

// queryResolver implements QueryResolver
type queryResolver struct{ *Resolver }

func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
    // Get current user from context (set by auth middleware)
    userID, ok := ctx.Value("userID").(string)
    if !ok {
        return nil, &model.AuthenticationError{
            Message: "not authenticated",
            Code:    "UNAUTHENTICATED",
        }
    }

    user, err := r.userService.GetUserByID(ctx, userID)
    if err != nil {
        return nil, err
    }

    return user, nil
}

func (r *queryResolver) Users(
    ctx context.Context,
    first *int,
    after *string,
    filter *model.UserFilterInput,
    orderBy *model.UserSortField,
    order *model.SortOrder,
) (*model.UserConnection, error) {
    // Build query options
    opts := service.ListUsersOptions{
        First:   first,
        After:   after,
        Filter:  filter,
        OrderBy: orderBy,
        Order:   order,
    }

    return r.userService.ListUsers(ctx, opts)
}

// mutationResolver implements MutationResolver
type mutationResolver struct{ *Resolver }

func (r *mutationResolver) CreateUser(
    ctx context.Context,
    input model.CreateUserInput,
) (model.CreateUserResult, error) {
    user, err := r.userService.CreateUser(ctx, input)
    if err != nil {
        // Return typed errors
        switch e := err.(type) {
        case *service.ValidationError:
            return &model.ValidationError{
                Message: e.Message,
                Code:    "VALIDATION_ERROR",
                Field:   e.Field,
            }, nil
        case *service.AuthError:
            return &model.AuthenticationError{
                Message: e.Message,
                Code:    "AUTHENTICATION_ERROR",
            }, nil
        default:
            return nil, err
        }
    }

    return user, nil
}

// subscriptionResolver implements SubscriptionResolver
type subscriptionResolver struct{ *Resolver }

func (r *subscriptionResolver) PostCreated(ctx context.Context) (<-chan *model.Post, error) {
    posts := make(chan *model.Post)

    go func() {
        defer close(posts)

        // Subscribe to post creation events
        eventChan := r.postService.SubscribePostCreated(ctx)

        for {
            select {
            case <-ctx.Done():
                return
            case post := <-eventChan:
                posts <- post
            }
        }
    }()

    return posts, nil
}
```

## Directive Implementation

```go
// internal/graph/directives/auth.go
package directives

import (
    "context"
    "github.com/99designs/gqlgen/graphql"
    "github.com/yourorg/yourproject/internal/graph/model"
)

func AuthDirective(ctx context.Context, obj interface{}, next graphql.Resolver, requires model.Role) (interface{}, error) {
    // Check if user is authenticated
    userID, ok := ctx.Value("userID").(string)
    if !ok {
        return nil, &model.AuthenticationError{
            Message: "authentication required",
            Code:    "UNAUTHENTICATED",
        }
    }

    // Check if user has required role
    userRole := ctx.Value("userRole").(model.Role)
    if userRole < requires {
        return nil, &model.AuthorizationError{
            Message:      "insufficient permissions",
            Code:         "FORBIDDEN",
            RequiredRole: requires,
        }
    }

    return next(ctx)
}
```

## DataLoader Pattern for N+1 Problem

```go
// internal/graph/dataloader/dataloader.go
package dataloader

import (
    "context"
    "time"
    "github.com/graph-gophers/dataloader"
    "github.com/yourorg/yourproject/internal/model"
    "github.com/yourorg/yourproject/internal/service"
)

type Loaders struct {
    UserLoader *dataloader.Loader
    PostLoader *dataloader.Loader
}

func NewLoaders(userService *service.UserService, postService *service.PostService) *Loaders {
    return &Loaders{
        UserLoader: dataloader.NewBatchedLoader(
            userBatchFunc(userService),
            dataloader.WithWait(time.Millisecond),
        ),
        PostLoader: dataloader.NewBatchedLoader(
            postBatchFunc(postService),
            dataloader.WithWait(time.Millisecond),
        ),
    }
}

func userBatchFunc(service *service.UserService) dataloader.BatchFunc {
    return func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
        userIDs := make([]string, len(keys))
        for i, key := range keys {
            userIDs[i] = key.String()
        }

        users, err := service.GetUsersByIDs(ctx, userIDs)
        if err != nil {
            return []*dataloader.Result{{Error: err}}
        }

        userMap := make(map[string]*model.User)
        for _, user := range users {
            userMap[user.ID] = user
        }

        results := make([]*dataloader.Result, len(keys))
        for i, key := range keys {
            user, ok := userMap[key.String()]
            if ok {
                results[i] = &dataloader.Result{Data: user}
            } else {
                results[i] = &dataloader.Result{Error: fmt.Errorf("user not found")}
            }
        }

        return results
    }
}
```

## Key Patterns

### Pagination (Relay Cursor Connections)
- Use cursor-based pagination for stability
- Include `PageInfo` with `hasNextPage` and `hasPreviousPage`
- Return `totalCount` for UI display
- Cursors should be opaque (base64 encoded)

### Error Handling
- Use union types for mutation results
- Implement `Error` interface for typed errors
- Include error codes for client handling
- Return errors as data, not in `errors` field (when appropriate)

### Authentication & Authorization
- Use custom directives for declarative auth
- Store user context in GraphQL context
- Implement role-based access control
- Support field-level authorization

### Subscriptions
- Use channels for real-time updates
- Clean up on context cancellation
- Consider using Redis pub/sub for distributed systems
- Rate limit subscription connections

### DataLoader
- Prevent N+1 queries
- Batch requests per query
- Configure appropriate wait times
- Cache results within request context

## Code Generation

Generate GraphQL code:
```bash
go run github.com/99designs/gqlgen generate
```

Generate with custom config:
```bash
go run github.com/99designs/gqlgen generate --config gqlgen.yml
```

## Testing

```go
// internal/graph/resolver_test.go
func TestQueryMe(t *testing.T) {
    resolver := &Resolver{
        userService: mockUserService,
    }

    ctx := context.WithValue(context.Background(), "userID", "user-123")

    user, err := resolver.Query().Me(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "user-123", user.ID)
}
```
