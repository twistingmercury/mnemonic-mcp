---
entity_name: AsyncAPI Specification Pattern
entity_type: api-specification
language: agnostic
domain: api-design
description: Comprehensive AsyncAPI 3.0.0 specification pattern for event-driven APIs with message channels, operations, and protocol bindings for Kafka, MQTT, AMQP, and WebSocket
tags:
  - AsyncAPI
  - event-driven
  - messaging
  - Kafka
  - MQTT
  - AMQP
  - WebSocket
  - pub-sub
version: AsyncAPI 3.0.0
related_patterns:
  - AsyncAPI Implementation Pattern (Go)
  - REST API Specification Pattern
  - gRPC Service Definition Pattern
---

# AsyncAPI Specification Pattern

This pattern provides a comprehensive AsyncAPI 3.0.0 specification for event-driven APIs. AsyncAPI is used to document message-based and event-driven architectures, similar to how OpenAPI documents REST APIs.

## When to Use AsyncAPI

- **Event-driven architectures**: Services communicate through events, not direct calls
- **Message queues**: Kafka, RabbitMQ, MQTT, AMQP, Redis Streams
- **Pub/Sub patterns**: Publishers send messages, subscribers receive them
- **Decoupled services**: Producers and consumers don't know about each other
- **Real-time data streaming**: Continuous data flows (logs, metrics, IoT)
- **WebSocket APIs**: Bidirectional communication channels

## Core AsyncAPI Concepts

### Channels
Communication pathways (topics, queues, routing keys) where messages flow.

### Operations
Actions that applications perform:
- `send`: Application sends messages to the channel
- `receive`: Application receives messages from the channel

### Messages
Data units exchanged with:
- **Payload**: The actual data (JSON, Avro, Protobuf)
- **Headers**: Metadata for routing, correlation, authentication

### Servers
Message brokers or streaming platforms (Kafka, MQTT broker, RabbitMQ, etc.)

### Protocols
- Kafka
- MQTT
- AMQP
- WebSocket
- HTTP/SSE
- Redis Streams

## Basic AsyncAPI Specification

```yaml
asyncapi: 3.0.0

info:
  title: User Management Events API
  version: 1.0.0
  description: |
    Event-driven API for user management system.
    Publishes events when users are created, updated, or deleted.
  contact:
    name: API Support
    email: support@example.com
  license:
    name: Apache 2.0
    url: https://www.apache.org/licenses/LICENSE-2.0.html

servers:
  production:
    host: kafka.example.com:9092
    protocol: kafka
    description: Production Kafka cluster
    security:
      - saslScram: []

  development:
    host: localhost:9092
    protocol: kafka
    description: Local Kafka for development

channels:
  userSignup:
    address: 'user.signup.v1'
    messages:
      userSignedUp:
        $ref: '#/components/messages/UserSignedUp'
    description: Channel for user signup events

  userUpdate:
    address: 'user.update.v1'
    messages:
      userUpdated:
        $ref: '#/components/messages/UserUpdated'
    description: Channel for user update events

  userDelete:
    address: 'user.delete.v1'
    messages:
      userDeleted:
        $ref: '#/components/messages/UserDeleted'
    description: Channel for user deletion events

operations:
  onUserSignup:
    action: receive
    channel:
      $ref: '#/channels/userSignup'
    summary: Receive user signup events
    description: |
      Subscribe to this operation to receive notifications when new users sign up.
    messages:
      - $ref: '#/channels/userSignup/messages/userSignedUp'

  publishUserUpdate:
    action: send
    channel:
      $ref: '#/channels/userUpdate'
    summary: Publish user update events
    description: |
      Send user update events when user information changes.
    messages:
      - $ref: '#/channels/userUpdate/messages/userUpdated'

  onUserDelete:
    action: receive
    channel:
      $ref: '#/channels/userDelete'
    summary: Receive user deletion events
    description: |
      Subscribe to receive notifications when users are deleted.
    messages:
      - $ref: '#/channels/userDelete/messages/userDeleted'

components:
  messages:
    UserSignedUp:
      name: UserSignedUp
      title: User Signed Up Event
      summary: Event published when a new user signs up
      contentType: application/json
      payload:
        $ref: '#/components/schemas/UserSignedUpPayload'
      examples:
        - payload:
            userId: "123e4567-e89b-12d3-a456-426614174000"
            email: "user@example.com"
            name: "John Doe"
            timestamp: "2024-01-15T10:30:00Z"
            metadata:
              source: "web-app"
              ipAddress: "192.168.1.1"

    UserUpdated:
      name: UserUpdated
      title: User Updated Event
      summary: Event published when user information is updated
      contentType: application/json
      payload:
        $ref: '#/components/schemas/UserUpdatedPayload'
      examples:
        - payload:
            userId: "123e4567-e89b-12d3-a456-426614174000"
            updatedFields:
              name: "John Smith"
              phone: "+1-555-0123"
            timestamp: "2024-01-15T11:00:00Z"

    UserDeleted:
      name: UserDeleted
      title: User Deleted Event
      summary: Event published when a user is deleted
      contentType: application/json
      payload:
        $ref: '#/components/schemas/UserDeletedPayload'
      examples:
        - payload:
            userId: "123e4567-e89b-12d3-a456-426614174000"
            deletedBy: "admin@example.com"
            reason: "Account closure requested"
            timestamp: "2024-01-15T12:00:00Z"

  schemas:
    UserSignedUpPayload:
      type: object
      required:
        - userId
        - email
        - name
        - timestamp
      properties:
        userId:
          type: string
          format: uuid
          description: Unique identifier for the user
        email:
          type: string
          format: email
          description: User's email address
        name:
          type: string
          description: User's full name
          minLength: 1
          maxLength: 255
        timestamp:
          type: string
          format: date-time
          description: When the signup occurred
        metadata:
          type: object
          description: Additional metadata about the signup
          properties:
            source:
              type: string
              description: Source of the signup (web-app, mobile-app, api)
              enum: [web-app, mobile-app, api]
            ipAddress:
              type: string
              description: IP address of the signup request

    UserUpdatedPayload:
      type: object
      required:
        - userId
        - updatedFields
        - timestamp
      properties:
        userId:
          type: string
          format: uuid
          description: Unique identifier for the user
        updatedFields:
          type: object
          description: Fields that were updated
          additionalProperties: true
        timestamp:
          type: string
          format: date-time
          description: When the update occurred

    UserDeletedPayload:
      type: object
      required:
        - userId
        - timestamp
      properties:
        userId:
          type: string
          format: uuid
          description: Unique identifier for the deleted user
        deletedBy:
          type: string
          format: email
          description: Email of the admin who deleted the user
        reason:
          type: string
          description: Reason for deletion
          maxLength: 500
        timestamp:
          type: string
          format: date-time
          description: When the deletion occurred

  securitySchemes:
    saslScram:
      type: scramSha256
      description: SASL/SCRAM-SHA-256 authentication for Kafka

    apiKey:
      type: apiKey
      in: user
      description: API key for broker authentication

    oauth2:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/oauth/token
          scopes:
            events:read: Read event streams
            events:write: Write to event streams
```

## Protocol-Specific Examples

### Kafka Bindings

```yaml
asyncapi: 3.0.0

info:
  title: Order Processing Events (Kafka)
  version: 1.0.0

servers:
  production:
    host: kafka.example.com:9092
    protocol: kafka
    bindings:
      kafka:
        schemaRegistryUrl: https://schema-registry.example.com
        schemaRegistryVendor: confluent

channels:
  orderCreated:
    address: 'orders.created.v1'
    bindings:
      kafka:
        topic: orders.created.v1
        partitions: 10
        replicas: 3
        configs:
          retention.ms: 604800000  # 7 days
          cleanup.policy: delete
    messages:
      orderCreated:
        $ref: '#/components/messages/OrderCreated'

operations:
  onOrderCreated:
    action: receive
    channel:
      $ref: '#/channels/orderCreated'
    bindings:
      kafka:
        groupId: order-processing-service
        clientId: order-processor-1

components:
  messages:
    OrderCreated:
      name: OrderCreated
      contentType: application/json
      bindings:
        kafka:
          key:
            type: string
            description: Order ID used as partition key
          schemaIdLocation: payload
          schemaIdPayloadEncoding: confluent
      payload:
        $ref: '#/components/schemas/OrderCreatedPayload'

  schemas:
    OrderCreatedPayload:
      type: object
      required: [orderId, customerId, items, total, timestamp]
      properties:
        orderId:
          type: string
          format: uuid
        customerId:
          type: string
          format: uuid
        items:
          type: array
          items:
            type: object
            required: [productId, quantity, price]
            properties:
              productId:
                type: string
              quantity:
                type: integer
                minimum: 1
              price:
                type: number
                format: double
        total:
          type: number
          format: double
        timestamp:
          type: string
          format: date-time
```

### MQTT Bindings

```yaml
asyncapi: 3.0.0

info:
  title: IoT Sensor Data API (MQTT)
  version: 1.0.0

servers:
  production:
    host: mqtt.example.com:8883
    protocol: mqtt
    description: Production MQTT broker
    security:
      - userPassword: []
    bindings:
      mqtt:
        clientId: sensor-gateway
        cleanSession: true
        keepAlive: 60
        lastWill:
          topic: devices/status
          qos: 1
          retain: true
          message: "Gateway disconnected"

channels:
  temperatureSensor:
    address: 'sensors/temperature/{deviceId}'
    parameters:
      deviceId:
        description: Unique device identifier
        schema:
          type: string
    bindings:
      mqtt:
        qos: 1
        retain: false
    messages:
      temperatureReading:
        $ref: '#/components/messages/TemperatureReading'

operations:
  publishTemperature:
    action: send
    channel:
      $ref: '#/channels/temperatureSensor'
    bindings:
      mqtt:
        qos: 1
        retain: false

components:
  messages:
    TemperatureReading:
      name: TemperatureReading
      contentType: application/json
      payload:
        $ref: '#/components/schemas/TemperatureReadingPayload'

  schemas:
    TemperatureReadingPayload:
      type: object
      required: [deviceId, temperature, timestamp]
      properties:
        deviceId:
          type: string
        temperature:
          type: number
          format: float
          description: Temperature in Celsius
        humidity:
          type: number
          format: float
          description: Relative humidity percentage
        timestamp:
          type: string
          format: date-time

  securitySchemes:
    userPassword:
      type: userPassword
      description: Username and password for MQTT broker
```

### WebSocket Example

```yaml
asyncapi: 3.0.0

info:
  title: Real-Time Notifications API (WebSocket)
  version: 1.0.0

servers:
  production:
    host: ws.example.com
    protocol: ws
    description: WebSocket server for real-time notifications
    security:
      - bearer: []

channels:
  notifications:
    address: '/notifications'
    messages:
      notification:
        $ref: '#/components/messages/Notification'

operations:
  receiveNotifications:
    action: receive
    channel:
      $ref: '#/channels/notifications'
    summary: Receive real-time notifications

components:
  messages:
    Notification:
      name: Notification
      contentType: application/json
      payload:
        $ref: '#/components/schemas/NotificationPayload'

  schemas:
    NotificationPayload:
      type: object
      required: [id, type, title, timestamp]
      properties:
        id:
          type: string
          format: uuid
        type:
          type: string
          enum: [info, warning, error, success]
        title:
          type: string
        message:
          type: string
        timestamp:
          type: string
          format: date-time
        data:
          type: object
          description: Additional notification data
          additionalProperties: true

  securitySchemes:
    bearer:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

## Request-Reply Pattern (Correlation)

For request-reply patterns over async protocols:

```yaml
asyncapi: 3.0.0

info:
  title: User Validation Service (Request-Reply)
  version: 1.0.0

servers:
  production:
    host: kafka.example.com:9092
    protocol: kafka

channels:
  userValidationRequest:
    address: 'user.validation.request.v1'
    messages:
      validateUser:
        $ref: '#/components/messages/ValidateUserRequest'

  userValidationResponse:
    address: 'user.validation.response.v1'
    messages:
      validationResult:
        $ref: '#/components/messages/ValidateUserResponse'

operations:
  requestValidation:
    action: send
    channel:
      $ref: '#/channels/userValidationRequest'
    reply:
      channel:
        $ref: '#/channels/userValidationResponse'

  respondToValidation:
    action: receive
    channel:
      $ref: '#/channels/userValidationResponse'

components:
  messages:
    ValidateUserRequest:
      name: ValidateUserRequest
      contentType: application/json
      correlationId:
        location: '$message.header#/correlationId'
      headers:
        type: object
        required: [correlationId, replyTo]
        properties:
          correlationId:
            type: string
            format: uuid
            description: Unique ID to correlate request and response
          replyTo:
            type: string
            description: Channel to send the response to
      payload:
        $ref: '#/components/schemas/ValidateUserRequestPayload'

    ValidateUserResponse:
      name: ValidateUserResponse
      contentType: application/json
      correlationId:
        location: '$message.header#/correlationId'
      headers:
        type: object
        required: [correlationId]
        properties:
          correlationId:
            type: string
            format: uuid
            description: Same ID from the request
      payload:
        $ref: '#/components/schemas/ValidateUserResponsePayload'

  schemas:
    ValidateUserRequestPayload:
      type: object
      required: [userId]
      properties:
        userId:
          type: string
          format: uuid

    ValidateUserResponsePayload:
      type: object
      required: [userId, isValid]
      properties:
        userId:
          type: string
          format: uuid
        isValid:
          type: boolean
        errors:
          type: array
          items:
            type: string
```

## Channel Naming Best Practices

### Hierarchical Naming

Use dot or slash notation:
- `domain.entity.action` → `user.signup.completed`
- `domain/entity/action` → `user/signup/completed`

### Examples

**User domain:**
- `user.signup.v1`
- `user.update.v1`
- `user.delete.v1`

**Order domain:**
- `orders.created.v1`
- `orders.updated.v1`
- `orders.cancelled.v1`
- `orders.shipped.v1`

**Payment domain:**
- `payments.initiated.v1`
- `payments.completed.v1`
- `payments.failed.v1`
- `payments.refunded.v1`

## Message Schema Best Practices

### Common Fields

Include in all messages:
- `id`: Unique message identifier (UUID)
- `timestamp`: ISO 8601 datetime when event occurred
- `version`: Schema version for evolution

### Envelope Pattern

```yaml
schemas:
  EventEnvelope:
    type: object
    required: [id, type, timestamp, version, data]
    properties:
      id:
        type: string
        format: uuid
      type:
        type: string
        description: Event type identifier
      timestamp:
        type: string
        format: date-time
      version:
        type: string
        description: Schema version
      data:
        type: object
        description: Event-specific payload
      metadata:
        type: object
        description: Optional metadata
        properties:
          correlationId:
            type: string
          causationId:
            type: string
          userId:
            type: string
```

## Error Handling

### Dead Letter Queues

```yaml
channels:
  orderProcessingDLQ:
    address: 'orders.processing.dlq'
    description: Dead letter queue for failed order processing
    messages:
      failedOrder:
        $ref: '#/components/messages/FailedOrderMessage'

components:
  messages:
    FailedOrderMessage:
      contentType: application/json
      payload:
        type: object
        required: [originalMessage, error, failureCount, timestamp]
        properties:
          originalMessage:
            type: object
            description: The original message that failed
          error:
            type: object
            required: [code, message]
            properties:
              code:
                type: string
              message:
                type: string
              stackTrace:
                type: string
          failureCount:
            type: integer
            description: Number of times processing has failed
          timestamp:
            type: string
            format: date-time
```

## Security Examples

### SASL/SCRAM for Kafka

```yaml
servers:
  production:
    protocol: kafka
    security:
      - saslScram: []

components:
  securitySchemes:
    saslScram:
      type: scramSha256
      description: SASL/SCRAM-SHA-256 authentication
```

### TLS Client Certificates

```yaml
servers:
  production:
    protocol: kafka
    security:
      - tlsClientCert: []

components:
  securitySchemes:
    tlsClientCert:
      type: X509
      description: Client certificate authentication
```

### OAuth 2.0

```yaml
servers:
  production:
    protocol: kafka
    security:
      - oauth2: [events:read, events:write]

components:
  securitySchemes:
    oauth2:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://auth.example.com/oauth/token
          scopes:
            events:read: Read from event streams
            events:write: Write to event streams
```

## Versioning Strategy

### Channel Versioning

Include version in channel address:
```yaml
channels:
  userSignupV1:
    address: 'user.signup.v1'

  userSignupV2:
    address: 'user.signup.v2'
```

### Schema Evolution

Use JSON Schema for backward compatibility:

```yaml
schemas:
  UserEventV2:
    type: object
    required: [userId, email]  # Keep required fields minimal
    properties:
      userId:
        type: string
      email:
        type: string
      # New optional fields can be added without breaking consumers
      phoneNumber:
        type: string
      # Deprecated fields should remain for compatibility
      legacyId:
        type: string
        deprecated: true
        description: "Use userId instead"
```

## Documentation Best Practices

1. **Provide examples**: Include real message examples in each message definition
2. **Document operations**: Explain when and why operations are triggered
3. **Describe channels**: Document the purpose and usage of each channel
4. **Define correlation**: Explain how request-reply patterns work
5. **Specify retention**: Document message retention policies
6. **Security requirements**: Clearly state authentication and authorization
7. **Error scenarios**: Document what happens when processing fails
8. **Rate limits**: Specify any throughput limitations

## Tools and Code Generation

AsyncAPI specifications can be used with:
- **AsyncAPI Generator**: Generate code, documentation, and diagrams
- **AsyncAPI Studio**: Visual editor for specifications
- **AsyncAPI CLI**: Command-line tools for validation and generation
- **Language-specific libraries**: Various implementations for Go, Python, Java, etc.

## Related Patterns

- **REST API Specification Pattern**: For synchronous request-response APIs
- **gRPC Service Definition Pattern**: For high-performance RPC communication
- **GraphQL Schema Pattern**: For flexible query-based APIs
- **AsyncAPI Implementation Pattern (Go)**: Language-specific implementation using Go
