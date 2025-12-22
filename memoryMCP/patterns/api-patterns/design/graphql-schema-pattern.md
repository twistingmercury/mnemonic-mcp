---
entity_name: GraphQL Schema Pattern
entity_type: api-specification
language: agnostic
domain: api-design
description: Comprehensive GraphQL schema pattern with queries, mutations, subscriptions, interfaces, cursor-based pagination, and error handling
tags:
  - GraphQL
  - API design
  - schema
  - pagination
  - subscriptions
  - Relay
version: GraphQL 2021
related_patterns:
  - GraphQL Implementation Pattern (Go)
  - REST API Specification Pattern
---

# GraphQL Schema Pattern

This pattern provides a complete GraphQL schema template following best practices for queries, mutations, subscriptions, pagination, and error handling. The schema is language-agnostic and can be implemented using any GraphQL server framework.

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

# Main Types
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

# Pagination (Relay Cursor Connections)
type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type UserEdge {
  node: User!
  cursor: String!
}

type PostConnection {
  edges: [PostEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type PostEdge {
  node: Post!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# Input Types
input CreateUserInput {
  email: Email!
  name: String!
  password: String!
}

input UpdateUserInput {
  email: Email
  name: String
  bio: String
  avatar: String
}

input CreatePostInput {
  title: String!
  content: String!
  tags: [String!]
  published: Boolean
}

# Mutation Results (Union Types for Error Handling)
type CreateUserSuccess {
  user: User!
}

type CreateUserError implements Error {
  message: String!
  code: String!
  field: String
}

union CreateUserResult = CreateUserSuccess | CreateUserError

type UpdateUserSuccess {
  user: User!
}

type UpdateUserError implements Error {
  message: String!
  code: String!
  field: String
}

union UpdateUserResult = UpdateUserSuccess | UpdateUserError

# Query Root
type Query {
  # Get current authenticated user
  me: User @auth

  # Get user by ID
  user(id: ID!): User

  # List users with pagination and filtering
  users(
    first: Int = 20
    after: String
    status: UserStatus
    orderBy: UserSortField = CREATED_AT
    order: SortOrder = DESC
  ): UserConnection! @auth(requires: ADMIN)

  # Search users by name or email
  searchUsers(query: String!, first: Int = 10, after: String): UserConnection!

  # Get post by ID
  post(id: ID!): Post

  # List posts with pagination
  posts(first: Int = 20, after: String, published: Boolean): PostConnection!
}

# Mutation Root
type Mutation {
  # User mutations
  createUser(input: CreateUserInput!): CreateUserResult!
  updateUser(id: ID!, input: UpdateUserInput!): UpdateUserResult! @auth
  deleteUser(id: ID!): Boolean! @auth(requires: ADMIN)

  # Post mutations
  createPost(input: CreatePostInput!): Post! @auth
  updatePost(id: ID!, input: CreatePostInput!): Post! @auth
  deletePost(id: ID!): Boolean! @auth
  publishPost(id: ID!): Post! @auth
}

# Subscription Root
type Subscription {
  # Subscribe to user changes
  userUpdated(userId: ID!): User! @auth

  # Subscribe to new posts
  postCreated(authorId: ID): Post!

  # Subscribe to comment notifications
  commentAdded(postId: ID!): Comment!
}
```

## Key Patterns

### 1. Pagination - Relay Cursor Connections

**Why:** Stable pagination that works with real-time data changes

```graphql
type Query {
  users(
    first: Int = 20      # Number of items to fetch
    after: String        # Cursor to fetch after
  ): UserConnection!
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!       # Total number of items
}

type UserEdge {
  node: User!            # The actual data
  cursor: String!        # Opaque cursor for this item
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}
```

**Benefits:**
- Works with insertions/deletions during pagination
- Opaque cursors hide implementation details
- Supports bidirectional navigation
- Standard pattern recognized by GraphQL clients

### 2. Error Handling - Union Result Types

**Why:** Type-safe error handling with detailed error information

```graphql
type CreateUserSuccess {
  user: User!
}

type CreateUserError implements Error {
  message: String!
  code: String!      # Machine-readable error code
  field: String      # Field that caused the error (for validation)
}

union CreateUserResult = CreateUserSuccess | CreateUserError

type Mutation {
  createUser(input: CreateUserInput!): CreateUserResult!
}
```

**Client Query:**
```graphql
mutation CreateUser($input: CreateUserInput!) {
  createUser(input: $input) {
    ... on CreateUserSuccess {
      user {
        id
        email
        name
      }
    }
    ... on CreateUserError {
      message
      code
      field
    }
  }
}
```

**Benefits:**
- Errors are part of the schema
- Type-safe error handling
- Field-level error attribution
- Better than throwing exceptions

### 3. Authentication & Authorization

**Directive-Based Auth:**
```graphql
directive @auth(requires: Role = USER) on OBJECT | FIELD_DEFINITION

type Query {
  # Requires any authenticated user
  me: User @auth

  # Requires ADMIN role
  users: UserConnection! @auth(requires: ADMIN)

  # Public endpoint (no directive)
  posts: PostConnection!
}
```

**Benefits:**
- Declarative authorization
- Easy to audit security requirements
- Consistent across schema
- Self-documenting

### 4. Subscriptions

**Real-time Updates:**
```graphql
type Subscription {
  # Subscribe to specific user changes
  userUpdated(userId: ID!): User! @auth

  # Subscribe to all new posts
  postCreated: Post!

  # Subscribe to filtered events
  postCreated(authorId: ID, tag: String): Post!
}
```

**Client Subscription:**
```graphql
subscription WatchUser($userId: ID!) {
  userUpdated(userId: $userId) {
    id
    name
    status
  }
}
```

**Benefits:**
- Real-time data updates
- Event-driven architecture
- Reduces polling
- Scalable with WebSockets

### 5. Input Types

**Mutation Inputs:**
```graphql
input CreateUserInput {
  email: Email!
  name: String!
  password: String!
}

input UpdateUserInput {
  # All fields optional for partial updates
  email: Email
  name: String
  bio: String
}

type Mutation {
  createUser(input: CreateUserInput!): CreateUserResult!
  updateUser(id: ID!, input: UpdateUserInput!): UpdateUserResult!
}
```

**Benefits:**
- Single argument for mutations
- Easy to extend without breaking changes
- Clear separation between create and update
- Type safety for nested data

### 6. Interfaces

**Shared Fields:**
```graphql
interface Node {
  id: ID!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type User implements Node {
  id: ID!
  createdAt: DateTime!
  updatedAt: DateTime!
  # User-specific fields...
}

type Post implements Node {
  id: ID!
  createdAt: DateTime!
  updatedAt: DateTime!
  # Post-specific fields...
}
```

**Query by Interface:**
```graphql
query GetNode($id: ID!) {
  node(id: $id) {
    id
    createdAt
    ... on User {
      email
      name
    }
    ... on Post {
      title
      content
    }
  }
}
```

## Schema Design Best Practices

### Naming Conventions

1. **Types:** PascalCase - `User`, `PostConnection`
2. **Fields:** camelCase - `firstName`, `createdAt`
3. **Enums:** UPPER_CASE - `ACTIVE`, `ADMIN`
4. **Input types:** End with `Input` - `CreateUserInput`
5. **Connection types:** End with `Connection` - `UserConnection`

### Nullability

```graphql
# Non-null field (always returns value)
id: ID!

# Non-null list with non-null items
tags: [String!]!

# Nullable list with non-null items
tags: [String!]

# Non-null list with nullable items
tags: [String]!

# Everything nullable
tags: [String]
```

**Guidelines:**
- Use `!` for required fields
- Return lists as `[Type!]!` (non-null list with non-null items)
- Avoid nullable IDs and timestamps
- Make input fields nullable for updates

### Versioning

**Don't version the schema like REST** - evolve it:

```graphql
type User {
  # Deprecated field
  fullName: String @deprecated(reason: "Use 'name' instead")

  # New field
  name: String!

  # Additive change (no breaking)
  email: Email
  phone: String  # New field added
}
```

**Guidelines:**
- Add new fields instead of changing existing ones
- Use `@deprecated` for old fields
- Never remove fields (mark deprecated instead)
- Add new optional arguments to queries

### Documentation

```graphql
"""
Represents a user in the system.

Users can create posts, comment on posts, and interact with other users.
"""
type User implements Node {
  "Unique identifier for the user"
  id: ID!

  "User's email address (unique, verified)"
  email: Email!

  "User's full name"
  name: String!

  """
  User's current status.

  - ACTIVE: User can log in and perform actions
  - INACTIVE: User cannot log in
  - SUSPENDED: User is temporarily blocked
  """
  status: UserStatus!
}
```

## Validation

### Schema Linting

Use schema linting tools to enforce best practices:

```bash
# Using GraphQL Inspector
npx @graphql-inspector/cli validate schema.graphql

# Using GraphQL ESLint
npx graphql-eslint schema.graphql
```

### Common Issues to Check

- Consistent naming conventions
- Proper use of nullability
- Missing descriptions
- Circular dependencies
- N+1 query potential
- Missing pagination
- Overly nested types

## Schema Organization

For large schemas, split into modules:

```
schema/
  ├── schema.graphql          # Root schema
  ├── user/
  │   ├── user.graphql        # User types
  │   └── user-queries.graphql
  ├── post/
  │   ├── post.graphql
  │   └── post-queries.graphql
  └── common/
      ├── scalars.graphql     # Custom scalars
      ├── directives.graphql  # Custom directives
      └── pagination.graphql  # Pagination types
```

Merge at build time or use schema stitching.
