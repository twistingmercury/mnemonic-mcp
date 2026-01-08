---
entity_name: GraphQL Federation Pattern
entity_type: GraphQL Pattern
language: go
domain: backend
description: Apollo Federation pattern for distributed GraphQL architecture with multiple subgraphs in Go microservices
tags:
  - GraphQL
  - Federation
  - Apollo
  - gqlgen
  - microservices
---

# GraphQL Federation Pattern

This pattern demonstrates implementing Apollo Federation for distributed GraphQL APIs across multiple Go services.

## Federation Architecture

```text
┌─────────────────┐
│  Apollo Gateway │  (Federation router)
└────────┬────────┘
         │
    ┌────┴────────────────────────────┐
    │                                 │
┌───▼────────┐              ┌────────▼─────┐
│  Users     │              │  Products    │
│  Subgraph  │              │  Subgraph    │
└────────────┘              └──────────────┘
```

## Users Subgraph Schema

```graphql
# users-service/schema.graphql

extend schema
  @link(
    url: "https://specs.apollo.dev/federation/v2.3"
    import: ["@key", "@shareable", "@external", "@requires", "@provides"]
  )

type Query {
  me: User @shareable
  user(id: ID!): User
  users(first: Int, after: String): UserConnection!
}

type User @key(fields: "id") {
  id: ID!
  email: String!
  name: String!
  username: String!
  avatar: String
  createdAt: DateTime!

  # Fields for other services to extend
  reviews: [Review!]! @external
  orders: [Order!]! @external
}

type UserConnection {
  edges: [UserEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type UserEdge {
  cursor: String!
  node: User!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type Mutation {
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
}

input CreateUserInput {
  email: String!
  name: String!
  username: String!
}

input UpdateUserInput {
  name: String
  username: String
  avatar: String
}

scalar DateTime
```

## Products Subgraph Schema

```graphql
# products-service/schema.graphql

extend schema
  @link(
    url: "https://specs.apollo.dev/federation/v2.3"
    import: ["@key", "@shareable", "@external", "@requires", "@provides"]
  )

# Extend User from users-service
type User @key(fields: "id") {
  id: ID! @external
  reviews: [Review!]!
}

type Query {
  product(id: ID!): Product
  products(first: Int, after: String, filter: ProductFilter): ProductConnection!
  searchProducts(query: String!): [Product!]!
}

type Product @key(fields: "id") {
  id: ID!
  name: String!
  description: String!
  price: Float!
  sku: String! @shareable
  inStock: Boolean!
  category: Category!

  # Average rating computed from reviews
  averageRating: Float

  # Reviews from reviews-service
  reviews(first: Int): ReviewConnection!

  # Relationship to user
  createdBy: User! @provides(fields: "id")
}

type Category @key(fields: "id") {
  id: ID!
  name: String!
  slug: String! @shareable
  products(first: Int): ProductConnection!
}

type Review @key(fields: "id") {
  id: ID!
  rating: Int!
  comment: String
  product: Product!
  author: User! @provides(fields: "id username")
  createdAt: DateTime!
}

type ProductConnection {
  edges: [ProductEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type ProductEdge {
  cursor: String!
  node: Product!
}

type ReviewConnection {
  edges: [ReviewEdge!]!
  pageInfo: PageInfo!
  averageRating: Float
}

type ReviewEdge {
  cursor: String!
  node: Review!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type Mutation {
  createProduct(input: CreateProductInput!): Product!
  updateProduct(id: ID!, input: UpdateProductInput!): Product!
  createReview(input: CreateReviewInput!): Review!
}

input ProductFilter {
  categoryId: ID
  minPrice: Float
  maxPrice: Float
  inStock: Boolean
}

input CreateProductInput {
  name: String!
  description: String!
  price: Float!
  sku: String!
  categoryId: ID!
}

input UpdateProductInput {
  name: String
  description: String
  price: Float
  inStock: Boolean
}

input CreateReviewInput {
  productId: ID!
  rating: Int!
  comment: String
}

scalar DateTime
```

## Orders Subgraph Schema

```graphql
# orders-service/schema.graphql

extend schema
  @link(
    url: "https://specs.apollo.dev/federation/v2.3"
    import: ["@key", "@shareable", "@external", "@requires", "@provides"]
  )

# Extend User from users-service
type User @key(fields: "id") {
  id: ID! @external
  orders: [Order!]!
}

# Extend Product from products-service
type Product @key(fields: "id") {
  id: ID! @external
  name: String! @external
  price: Float! @external
}

type Query {
  order(id: ID!): Order
  orders(userId: ID, first: Int, after: String): OrderConnection!
}

type Order @key(fields: "id") {
  id: ID!
  user: User!
  items: [OrderItem!]!
  status: OrderStatus!
  total: Float!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type OrderItem {
  id: ID!
  product: Product!
  quantity: Int!
  price: Float!
  subtotal: Float!
}

enum OrderStatus {
  PENDING
  PROCESSING
  SHIPPED
  DELIVERED
  CANCELLED
}

type OrderConnection {
  edges: [OrderEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type OrderEdge {
  cursor: String!
  node: Order!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type Mutation {
  createOrder(input: CreateOrderInput!): Order!
  updateOrderStatus(orderId: ID!, status: OrderStatus!): Order!
}

input CreateOrderInput {
  items: [OrderItemInput!]!
}

input OrderItemInput {
  productId: ID!
  quantity: Int!
}

scalar DateTime
```

## Federation Resolver Implementation

```go
// users-service/internal/graph/entity.go
package graph

import (
    "context"
    "github.com/99designs/gqlgen/graphql"
)

func (r *Resolver) Entity() EntityResolver {
    return &entityResolver{r}
}

type entityResolver struct{ *Resolver }

// ResolveEntity resolves federated entities
func (r *entityResolver) FindUserByID(ctx context.Context, id string) (*model.User, error) {
    return r.userService.GetUserByID(ctx, id)
}
```

```go
// products-service/internal/graph/entity.go
package graph

import (
    "context"
    "github.com/yourorg/products-service/internal/graph/model"
)

func (r *Resolver) Entity() EntityResolver {
    return &entityResolver{r}
}

type entityResolver struct{ *Resolver }

// FindProductByID resolves Product entities
func (r *entityResolver) FindProductByID(ctx context.Context, id string) (*model.Product, error) {
    return r.productService.GetProductByID(ctx, id)
}

// FindCategoryByID resolves Category entities
func (r *entityResolver) FindCategoryByID(ctx context.Context, id string) (*model.Category, error) {
    return r.categoryService.GetCategoryByID(ctx, id)
}

// FindUserByID resolves User stub (extended from users-service)
func (r *entityResolver) FindUserByID(ctx context.Context, id string) (*model.User, error) {
    // Return stub - actual fields resolved by users-service
    return &model.User{ID: id}, nil
}
```

## Reference Resolver (Extending Entities)

```go
// products-service/internal/graph/schema.resolvers.go
package graph

import (
    "context"
    "github.com/yourorg/products-service/internal/graph/model"
)

// Reviews resolver for User type (extending from users-service)
func (r *userResolver) Reviews(ctx context.Context, obj *model.User) ([]*model.Review, error) {
    return r.reviewService.GetReviewsByUserID(ctx, obj.ID)
}

// User type resolver
func (r *Resolver) User() UserResolver {
    return &userResolver{r}
}

type userResolver struct{ *Resolver }
```

## Federation gqlgen Configuration

```yaml
# users-service/gqlgen.yml
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

federation:
  filename: internal/graph/federation.go
  package: graph
  version: 2

autobind:
  - github.com/yourorg/users-service/internal/model
```

## Apollo Gateway Configuration

```typescript
// gateway/index.ts
import { ApolloGateway, IntrospectAndCompose } from "@apollo/gateway";
import { ApolloServer } from "@apollo/server";
import { startStandaloneServer } from "@apollo/server/standalone";

const gateway = new ApolloGateway({
  supergraphSdl: new IntrospectAndCompose({
    subgraphs: [
      { name: "users", url: "http://users-service:8080/graphql" },
      { name: "products", url: "http://products-service:8080/graphql" },
      { name: "orders", url: "http://orders-service:8080/graphql" },
    ],
    pollIntervalInMs: 10000, // Poll for schema changes every 10s
  }),
});

const server = new ApolloServer({
  gateway,
});

const { url } = await startStandaloneServer(server, {
  listen: { port: 4000 },
});

console.log(`Gateway ready at ${url}`);
```

## Rover CLI for Schema Management

```bash
# Install Rover CLI
curl -sSL https://rover.apollo.dev/nix/latest | sh

# Check subgraph schema
rover subgraph check my-graph@main \
  --schema ./schema.graphql \
  --name users

# Publish subgraph schema
rover subgraph publish my-graph@main \
  --schema ./schema.graphql \
  --name users \
  --routing-url http://users-service:8080/graphql

# Compose supergraph locally
rover supergraph compose --config supergraph.yaml > supergraph.graphql
```

## Supergraph Configuration

```yaml
# supergraph.yaml
federation_version: 2
subgraphs:
  users:
    routing_url: http://users-service:8080/graphql
    schema:
      file: ./users-service/schema.graphql
  products:
    routing_url: http://products-service:8080/graphql
    schema:
      file: ./products-service/schema.graphql
  orders:
    routing_url: http://orders-service:8080/graphql
    schema:
      file: ./orders-service/schema.graphql
```

## Example Federated Query

```graphql
query GetUserWithOrdersAndReviews($userId: ID!) {
  user(id: $userId) {
    # From users-service
    id
    name
    email

    # From orders-service (extends User)
    orders(first: 5) {
      edges {
        node {
          id
          total
          status
          items {
            product {
              # From products-service
              id
              name
              price
            }
            quantity
            subtotal
          }
        }
      }
    }

    # From products-service (extends User)
    reviews(first: 10) {
      edges {
        node {
          id
          rating
          comment
          product {
            name
            category {
              name
            }
          }
        }
      }
    }
  }
}
```

## Key Federation Concepts

### @key Directive

Marks a type as an entity that can be resolved across subgraphs:

```graphql
type User @key(fields: "id") {
  id: ID!
  name: String!
}
```

### @external Directive

Marks a field as owned by another subgraph:

```graphql
type User @key(fields: "id") {
  id: ID! @external
  reviews: [Review!]!
}
```

### @requires Directive

Specifies fields needed from another subgraph:

```graphql
type Product @key(fields: "id") {
  id: ID!
  price: Float! @external
  discountedPrice: Float! @requires(fields: "price")
}
```

### @provides Directive

Allows a subgraph to provide fields from another subgraph:

```graphql
type Review {
  author: User! @provides(fields: "username")
}
```

### @shareable Directive

Allows multiple subgraphs to resolve the same field:

```graphql
type Product @key(fields: "id") {
  id: ID!
  sku: String! @shareable
}
```

## Best Practices

### Schema Design

1. Define clear entity boundaries
2. Use @key for entity identification
3. Avoid circular dependencies between subgraphs
4. Keep shared types minimal

### Performance

1. Implement DataLoader in each subgraph
2. Use @provides to reduce round trips
3. Consider caching at gateway level
4. Monitor query complexity

### Deployment

1. Use managed federation (Apollo Studio)
2. Implement health checks per subgraph
3. Version schemas with Rover CLI
4. Test composition locally before deploy

### Error Handling

1. Return partial data on subgraph failures
2. Implement proper error propagation
3. Use gateway-level error formatting
4. Monitor subgraph availability

## Testing Federation

```go
// Test entity resolution
func TestUserEntity(t *testing.T) {
    resolver := setupResolver()

    user, err := resolver.Entity().FindUserByID(
        context.Background(),
        "user-123",
    )

    assert.NoError(t, err)
    assert.Equal(t, "user-123", user.ID)
}
```

## Docker Compose for Local Development

```yaml
version: "3.8"
services:
  gateway:
    build: ./gateway
    ports:
      - "4000:4000"
    depends_on:
      - users-service
      - products-service
      - orders-service

  users-service:
    build: ./users-service
    ports:
      - "8081:8080"
    environment:
      - DB_HOST=users-db

  products-service:
    build: ./products-service
    ports:
      - "8082:8080"
    environment:
      - DB_HOST=products-db

  orders-service:
    build: ./orders-service
    ports:
      - "8083:8080"
    environment:
      - DB_HOST=orders-db
```
