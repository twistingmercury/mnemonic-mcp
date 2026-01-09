---
entity_name: AsyncAPI Implementation Pattern (Go)
entity_type: backend-implementation
language: go
domain: backend
description: Go implementation of AsyncAPI event-driven patterns using Kafka (sarama), MQTT (paho), AMQP (amqp091), and WebSocket (gorilla) with message handling, error recovery, and graceful shutdown
tags:
  - AsyncAPI
  - event-driven
  - Kafka
  - sarama
  - MQTT
  - paho
  - AMQP
  - RabbitMQ
  - WebSocket
  - gorilla
  - messaging
version: Go 1.23+
related_patterns:
  - AsyncAPI Specification Pattern
  - REST API Implementation Pattern (Go)
  - gRPC Implementation Pattern (Go)
---

# AsyncAPI Implementation Pattern (Go)

This pattern demonstrates how to implement AsyncAPI event-driven patterns in Go using popular messaging libraries.

## Libraries

### Kafka

```bash
go get github.com/IBM/sarama
```

### MQTT

```bash
go get github.com/eclipse/paho.mqtt.golang
```

### AMQP (RabbitMQ)

```bash
go get github.com/rabbitmq/amqp091-go
```

### WebSocket

```bash
go get github.com/gorilla/websocket
```

## Project Structure

```text
.
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── events/
│   │   ├── publisher.go      # Event publishing interface
│   │   ├── subscriber.go     # Event subscription interface
│   │   └── handler.go        # Message handler interface
│   ├── kafka/
│   │   ├── producer.go       # Kafka producer implementation
│   │   ├── consumer.go       # Kafka consumer implementation
│   │   └── config.go         # Kafka configuration
│   ├── mqtt/
│   │   ├── publisher.go      # MQTT publisher implementation
│   │   ├── subscriber.go     # MQTT subscriber implementation
│   │   └── config.go         # MQTT configuration
│   ├── amqp/
│   │   ├── publisher.go      # AMQP publisher implementation
│   │   ├── consumer.go       # AMQP consumer implementation
│   │   └── config.go         # AMQP configuration
│   ├── websocket/
│   │   ├── server.go         # WebSocket server implementation
│   │   ├── client.go         # WebSocket client implementation
│   │   └── hub.go            # WebSocket connection hub
│   └── models/
│       └── events.go         # Event message models
└── go.mod
```

## Core Event Models

```go
// internal/models/events.go
package models

import (
    "encoding/json"
    "time"
    "github.com/google/uuid"
)

// EventEnvelope wraps all events with common metadata
type EventEnvelope struct {
    ID            string                 `json:"id"`
    Type          string                 `json:"type"`
    Timestamp     time.Time              `json:"timestamp"`
    Version       string                 `json:"version"`
    Data          json.RawMessage        `json:"data"`
    Metadata      map[string]interface{} `json:"metadata,omitempty"`
    CorrelationID string                 `json:"correlationId,omitempty"`
}

// NewEventEnvelope creates a new event envelope
func NewEventEnvelope(eventType string, version string, data interface{}) (*EventEnvelope, error) {
    dataBytes, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    return &EventEnvelope{
        ID:        uuid.New().String(),
        Type:      eventType,
        Timestamp: time.Now().UTC(),
        Version:   version,
        Data:      dataBytes,
        Metadata:  make(map[string]interface{}),
    }, nil
}

// UserSignedUpPayload represents a user signup event
type UserSignedUpPayload struct {
    UserID    string                 `json:"userId"`
    Email     string                 `json:"email"`
    Name      string                 `json:"name"`
    Timestamp time.Time              `json:"timestamp"`
    Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// UserUpdatedPayload represents a user update event
type UserUpdatedPayload struct {
    UserID        string                 `json:"userId"`
    UpdatedFields map[string]interface{} `json:"updatedFields"`
    Timestamp     time.Time              `json:"timestamp"`
}

// UserDeletedPayload represents a user deletion event
type UserDeletedPayload struct {
    UserID    string    `json:"userId"`
    DeletedBy string    `json:"deletedBy,omitempty"`
    Reason    string    `json:"reason,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}
```

## Event Publisher/Subscriber Interfaces

```go
// internal/events/publisher.go
package events

import (
    "context"
)

// Publisher defines the interface for publishing events
type Publisher interface {
    Publish(ctx context.Context, channel string, message []byte) error
    Close() error
}

// Subscriber defines the interface for subscribing to events
type Subscriber interface {
    Subscribe(ctx context.Context, channel string, handler MessageHandler) error
    Close() error
}

// MessageHandler processes incoming messages
type MessageHandler interface {
    Handle(ctx context.Context, message Message) error
}

// Message represents a received message
type Message struct {
    Topic     string
    Key       []byte
    Value     []byte
    Headers   map[string]string
    Partition int32
    Offset    int64
    Timestamp time.Time
}
```

## Kafka Implementation

### Kafka Producer

```go
// internal/kafka/producer.go
package kafka

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/IBM/sarama"
)

type Producer struct {
    producer sarama.SyncProducer
    config   *Config
}

func NewProducer(config *Config) (*Producer, error) {
    saramaConfig := sarama.NewConfig()
    saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
    saramaConfig.Producer.Retry.Max = 5
    saramaConfig.Producer.Return.Successes = true
    saramaConfig.Producer.Compression = sarama.CompressionSnappy

    producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create kafka producer: %w", err)
    }

    return &Producer{
        producer: producer,
        config:   config,
    }, nil
}

func (p *Producer) Publish(ctx context.Context, topic string, message []byte) error {
    msg := &sarama.ProducerMessage{
        Topic: topic,
        Value: sarama.ByteEncoder(message),
    }

    partition, offset, err := p.producer.SendMessage(msg)
    if err != nil {
        return fmt.Errorf("failed to publish message: %w", err)
    }

    fmt.Printf("Message published to partition %d at offset %d\n", partition, offset)
    return nil
}

func (p *Producer) PublishWithKey(ctx context.Context, topic string, key string, message []byte) error {
    msg := &sarama.ProducerMessage{
        Topic: topic,
        Key:   sarama.StringEncoder(key),
        Value: sarama.ByteEncoder(message),
    }

    partition, offset, err := p.producer.SendMessage(msg)
    if err != nil {
        return fmt.Errorf("failed to publish message: %w", err)
    }

    fmt.Printf("Message with key '%s' published to partition %d at offset %d\n", key, partition, offset)
    return nil
}

func (p *Producer) Close() error {
    return p.producer.Close()
}
```

### Kafka Consumer

```go
// internal/kafka/consumer.go
package kafka

import (
    "context"
    "fmt"
    "log"
    "sync"

    "github.com/IBM/sarama"
    "myapp/internal/events"
)

type Consumer struct {
    consumerGroup sarama.ConsumerGroup
    config        *Config
    topics        []string
    handler       events.MessageHandler
}

func NewConsumer(config *Config, topics []string, handler events.MessageHandler) (*Consumer, error) {
    saramaConfig := sarama.NewConfig()
    saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()
    saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
    saramaConfig.Consumer.Return.Errors = true

    consumerGroup, err := sarama.NewConsumerGroup(config.Brokers, config.GroupID, saramaConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create consumer group: %w", err)
    }

    return &Consumer{
        consumerGroup: consumerGroup,
        config:        config,
        topics:        topics,
        handler:       handler,
    }, nil
}

func (c *Consumer) Subscribe(ctx context.Context) error {
    consumer := &consumerGroupHandler{
        handler: c.handler,
    }

    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()
        for {
            if err := c.consumerGroup.Consume(ctx, c.topics, consumer); err != nil {
                log.Printf("Error from consumer: %v", err)
            }

            if ctx.Err() != nil {
                return
            }
        }
    }()

    // Handle errors
    go func() {
        for err := range c.consumerGroup.Errors() {
            log.Printf("Consumer error: %v", err)
        }
    }()

    wg.Wait()
    return nil
}

func (c *Consumer) Close() error {
    return c.consumerGroup.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
    handler events.MessageHandler
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
    return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
    return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    for message := range claim.Messages() {
        msg := events.Message{
            Topic:     message.Topic,
            Key:       message.Key,
            Value:     message.Value,
            Partition: message.Partition,
            Offset:    message.Offset,
            Timestamp: message.Timestamp,
            Headers:   make(map[string]string),
        }

        // Extract headers
        for _, header := range message.Headers {
            msg.Headers[string(header.Key)] = string(header.Value)
        }

        if err := h.handler.Handle(session.Context(), msg); err != nil {
            log.Printf("Error handling message: %v", err)
            continue
        }

        session.MarkMessage(message, "")
    }

    return nil
}
```

### Kafka Configuration

```go
// internal/kafka/config.go
package kafka

type Config struct {
    Brokers []string
    GroupID string
}

func NewConfig(brokers []string, groupID string) *Config {
    return &Config{
        Brokers: brokers,
        GroupID: groupID,
    }
}
```

## MQTT Implementation

### MQTT Publisher

```go
// internal/mqtt/publisher.go
package mqtt

import (
    "context"
    "fmt"
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Publisher struct {
    client mqtt.Client
    config *Config
}

func NewPublisher(config *Config) (*Publisher, error) {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(config.Broker)
    opts.SetClientID(config.ClientID)
    opts.SetUsername(config.Username)
    opts.SetPassword(config.Password)
    opts.SetKeepAlive(60 * time.Second)
    opts.SetPingTimeout(10 * time.Second)
    opts.SetAutoReconnect(true)

    if config.LastWill != nil {
        opts.SetWill(config.LastWill.Topic, config.LastWill.Message, byte(config.LastWill.QoS), config.LastWill.Retained)
    }

    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
    }

    return &Publisher{
        client: client,
        config: config,
    }, nil
}

func (p *Publisher) Publish(ctx context.Context, topic string, message []byte) error {
    return p.PublishWithQoS(ctx, topic, message, 1, false)
}

func (p *Publisher) PublishWithQoS(ctx context.Context, topic string, message []byte, qos byte, retained bool) error {
    token := p.client.Publish(topic, qos, retained, message)

    // Wait for publish with timeout
    timeout := 5 * time.Second
    if !token.WaitTimeout(timeout) {
        return fmt.Errorf("publish timeout after %v", timeout)
    }

    if token.Error() != nil {
        return fmt.Errorf("failed to publish message: %w", token.Error())
    }

    return nil
}

func (p *Publisher) Close() error {
    p.client.Disconnect(250)
    return nil
}
```

### MQTT Subscriber

```go
// internal/mqtt/subscriber.go
package mqtt

import (
    "context"
    "fmt"
    "log"
    "time"

    mqtt "github.com/eclipse/paho.mqtt.golang"
    "myapp/internal/events"
)

type Subscriber struct {
    client  mqtt.Client
    config  *Config
    handler events.MessageHandler
}

func NewSubscriber(config *Config, handler events.MessageHandler) (*Subscriber, error) {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(config.Broker)
    opts.SetClientID(config.ClientID)
    opts.SetUsername(config.Username)
    opts.SetPassword(config.Password)
    opts.SetKeepAlive(60 * time.Second)
    opts.SetPingTimeout(10 * time.Second)
    opts.SetAutoReconnect(true)

    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
    }

    return &Subscriber{
        client:  client,
        config:  config,
        handler: handler,
    }, nil
}

func (s *Subscriber) Subscribe(ctx context.Context, topic string, qos byte) error {
    messageHandler := func(client mqtt.Client, msg mqtt.Message) {
        message := events.Message{
            Topic:     msg.Topic(),
            Value:     msg.Payload(),
            Headers:   make(map[string]string),
            Timestamp: time.Now(),
        }

        if err := s.handler.Handle(ctx, message); err != nil {
            log.Printf("Error handling message: %v", err)
        }
    }

    token := s.client.Subscribe(topic, qos, messageHandler)
    if token.Wait() && token.Error() != nil {
        return fmt.Errorf("failed to subscribe to topic %s: %w", topic, token.Error())
    }

    log.Printf("Subscribed to topic: %s (QoS %d)", topic, qos)
    return nil
}

func (s *Subscriber) Close() error {
    s.client.Disconnect(250)
    return nil
}
```

### MQTT Configuration

```go
// internal/mqtt/config.go
package mqtt

type Config struct {
    Broker   string
    ClientID string
    Username string
    Password string
    LastWill *LastWill
}

type LastWill struct {
    Topic    string
    Message  string
    QoS      int
    Retained bool
}

func NewConfig(broker, clientID, username, password string) *Config {
    return &Config{
        Broker:   broker,
        ClientID: clientID,
        Username: username,
        Password: password,
    }
}
```

## AMQP (RabbitMQ) Implementation

### AMQP Publisher

```go
// internal/amqp/publisher.go
package amqp

import (
    "context"
    "fmt"

    amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    config  *Config
}

func NewPublisher(config *Config) (*Publisher, error) {
    conn, err := amqp.Dial(config.URL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
    }

    channel, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("failed to open channel: %w", err)
    }

    return &Publisher{
        conn:    conn,
        channel: channel,
        config:  config,
    }, nil
}

func (p *Publisher) DeclareExchange(name, kind string, durable, autoDelete bool) error {
    return p.channel.ExchangeDeclare(
        name,
        kind,
        durable,
        autoDelete,
        false, // internal
        false, // no-wait
        nil,   // arguments
    )
}

func (p *Publisher) Publish(ctx context.Context, exchange, routingKey string, message []byte) error {
    return p.channel.PublishWithContext(
        ctx,
        exchange,
        routingKey,
        false, // mandatory
        false, // immediate
        amqp.Publishing{
            ContentType:  "application/json",
            Body:         message,
            DeliveryMode: amqp.Persistent,
        },
    )
}

func (p *Publisher) Close() error {
    if err := p.channel.Close(); err != nil {
        return err
    }
    return p.conn.Close()
}
```

### AMQP Consumer

```go
// internal/amqp/consumer.go
package amqp

import (
    "context"
    "fmt"
    "log"
    "time"

    amqp "github.com/rabbitmq/amqp091-go"
    "myapp/internal/events"
)

type Consumer struct {
    conn    *amqp.Connection
    channel *amqp.Channel
    config  *Config
    handler events.MessageHandler
}

func NewConsumer(config *Config, handler events.MessageHandler) (*Consumer, error) {
    conn, err := amqp.Dial(config.URL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
    }

    channel, err := conn.Channel()
    if err != nil {
        conn.Close()
        return nil, fmt.Errorf("failed to open channel: %w", err)
    }

    // Set QoS (prefetch count)
    if err := channel.Qos(10, 0, false); err != nil {
        channel.Close()
        conn.Close()
        return nil, fmt.Errorf("failed to set QoS: %w", err)
    }

    return &Consumer{
        conn:    conn,
        channel: channel,
        config:  config,
        handler: handler,
    }, nil
}

func (c *Consumer) DeclareQueue(name string, durable, autoDelete bool) error {
    _, err := c.channel.QueueDeclare(
        name,
        durable,
        autoDelete,
        false, // exclusive
        false, // no-wait
        nil,   // arguments
    )
    return err
}

func (c *Consumer) BindQueue(queue, exchange, routingKey string) error {
    return c.channel.QueueBind(
        queue,
        routingKey,
        exchange,
        false, // no-wait
        nil,   // arguments
    )
}

func (c *Consumer) Consume(ctx context.Context, queue string) error {
    msgs, err := c.channel.Consume(
        queue,
        "",    // consumer tag
        false, // auto-ack
        false, // exclusive
        false, // no-local
        false, // no-wait
        nil,   // args
    )
    if err != nil {
        return fmt.Errorf("failed to start consuming: %w", err)
    }

    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            case msg, ok := <-msgs:
                if !ok {
                    return
                }

                message := events.Message{
                    Topic:     msg.RoutingKey,
                    Value:     msg.Body,
                    Headers:   make(map[string]string),
                    Timestamp: time.Now(),
                }

                // Extract headers
                for key, value := range msg.Headers {
                    if str, ok := value.(string); ok {
                        message.Headers[key] = str
                    }
                }

                if err := c.handler.Handle(ctx, message); err != nil {
                    log.Printf("Error handling message: %v", err)
                    msg.Nack(false, true) // requeue
                    continue
                }

                msg.Ack(false)
            }
        }
    }()

    log.Printf("Started consuming from queue: %s", queue)
    return nil
}

func (c *Consumer) Close() error {
    if err := c.channel.Close(); err != nil {
        return err
    }
    return c.conn.Close()
}
```

### AMQP Configuration

```go
// internal/amqp/config.go
package amqp

type Config struct {
    URL string
}

func NewConfig(url string) *Config {
    return &Config{
        URL: url,
    }
}
```

## WebSocket Implementation

### WebSocket Server

```go
// internal/websocket/server.go
package websocket

import (
    "context"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Configure properly in production
    },
}

type Server struct {
    hub *Hub
}

func NewServer() *Server {
    hub := NewHub()
    go hub.Run()

    return &Server{
        hub: hub,
    }
}

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Printf("WebSocket upgrade error: %v", err)
        return
    }

    client := &Client{
        hub:  s.hub,
        conn: conn,
        send: make(chan []byte, 256),
    }

    client.hub.register <- client

    go client.writePump()
    go client.readPump()
}

func (s *Server) Broadcast(message []byte) {
    s.hub.broadcast <- message
}
```

### WebSocket Hub

```go
// internal/websocket/hub.go
package websocket

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.clients[client] = true

        case client := <-h.unregister:
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }

        case message := <-h.broadcast:
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    close(client.send)
                    delete(h.clients, client)
                }
            }
        }
    }
}
```

### WebSocket Client

```go
// internal/websocket/client.go
package websocket

import (
    "log"
    "time"

    "github.com/gorilla/websocket"
)

const (
    writeWait      = 10 * time.Second
    pongWait       = 60 * time.Second
    pingPeriod     = (pongWait * 9) / 10
    maxMessageSize = 512 * 1024 // 512 KB
)

type Client struct {
    hub  *Hub
    conn *websocket.Conn
    send chan []byte
}

func (c *Client) readPump() {
    defer func() {
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    c.conn.SetReadLimit(maxMessageSize)

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }

        // Echo or process message
        c.hub.broadcast <- message
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // Add queued messages
            n := len(c.send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}
```

## Message Handler Example

```go
// internal/handlers/user_event_handler.go
package handlers

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "myapp/internal/events"
    "myapp/internal/models"
)

type UserEventHandler struct {
    // Dependencies (database, services, etc.)
}

func NewUserEventHandler() *UserEventHandler {
    return &UserEventHandler{}
}

func (h *UserEventHandler) Handle(ctx context.Context, msg events.Message) error {
    var envelope models.EventEnvelope
    if err := json.Unmarshal(msg.Value, &envelope); err != nil {
        return fmt.Errorf("failed to unmarshal event envelope: %w", err)
    }

    log.Printf("Processing event type: %s, ID: %s", envelope.Type, envelope.ID)

    switch envelope.Type {
    case "user.signup":
        return h.handleUserSignup(ctx, envelope)
    case "user.update":
        return h.handleUserUpdate(ctx, envelope)
    case "user.delete":
        return h.handleUserDelete(ctx, envelope)
    default:
        log.Printf("Unknown event type: %s", envelope.Type)
        return nil
    }
}

func (h *UserEventHandler) handleUserSignup(ctx context.Context, envelope models.EventEnvelope) error {
    var payload models.UserSignedUpPayload
    if err := json.Unmarshal(envelope.Data, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal user signup payload: %w", err)
    }

    log.Printf("User signed up: %s (%s)", payload.Name, payload.Email)

    // Process signup (send welcome email, create profile, etc.)
    return nil
}

func (h *UserEventHandler) handleUserUpdate(ctx context.Context, envelope models.EventEnvelope) error {
    var payload models.UserUpdatedPayload
    if err := json.Unmarshal(envelope.Data, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal user update payload: %w", err)
    }

    log.Printf("User updated: %s", payload.UserID)
    return nil
}

func (h *UserEventHandler) handleUserDelete(ctx context.Context, envelope models.EventEnvelope) error {
    var payload models.UserDeletedPayload
    if err := json.Unmarshal(envelope.Data, &payload); err != nil {
        return fmt.Errorf("failed to unmarshal user delete payload: %w", err)
    }

    log.Printf("User deleted: %s", payload.UserID)
    return nil
}
```

## Main Application Example

```go
// cmd/server/main.go
package main

import (
    "context"
    "encoding/json"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "myapp/internal/handlers"
    "myapp/internal/kafka"
    "myapp/internal/models"
)

func main() {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Kafka configuration
    kafkaConfig := kafka.NewConfig(
        []string{"localhost:9092"},
        "user-event-consumer-group",
    )

    // Create message handler
    handler := handlers.NewUserEventHandler()

    // Create Kafka consumer
    consumer, err := kafka.NewConsumer(
        kafkaConfig,
        []string{"user.signup.v1", "user.update.v1", "user.delete.v1"},
        handler,
    )
    if err != nil {
        log.Fatalf("Failed to create Kafka consumer: %v", err)
    }
    defer consumer.Close()

    // Start consuming
    go func() {
        if err := consumer.Subscribe(ctx); err != nil {
            log.Printf("Consumer error: %v", err)
        }
    }()

    // Example: Publish an event
    producer, err := kafka.NewProducer(kafkaConfig)
    if err != nil {
        log.Fatalf("Failed to create Kafka producer: %v", err)
    }
    defer producer.Close()

    // Publish user signup event
    signupPayload := models.UserSignedUpPayload{
        UserID:    "123e4567-e89b-12d3-a456-426614174000",
        Email:     "user@example.com",
        Name:      "John Doe",
        Timestamp: time.Now().UTC(),
        Metadata: map[string]interface{}{
            "source": "web-app",
        },
    }

    envelope, err := models.NewEventEnvelope("user.signup", "1.0", signupPayload)
    if err != nil {
        log.Fatalf("Failed to create event envelope: %v", err)
    }

    message, err := json.Marshal(envelope)
    if err != nil {
        log.Fatalf("Failed to marshal event: %v", err)
    }

    if err := producer.Publish(ctx, "user.signup.v1", message); err != nil {
        log.Printf("Failed to publish event: %v", err)
    }

    log.Println("Event-driven service started")

    // Wait for shutdown signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down gracefully...")
    cancel()
    time.Sleep(2 * time.Second)
}
```

## Error Handling and Retries

```go
// internal/kafka/retry.go
package kafka

import (
    "context"
    "fmt"
    "log"
    "time"

    "myapp/internal/events"
)

type RetryHandler struct {
    maxRetries int
    backoff    time.Duration
    handler    events.MessageHandler
    dlqTopic   string
    producer   *Producer
}

func NewRetryHandler(handler events.MessageHandler, maxRetries int, backoff time.Duration, dlqTopic string, producer *Producer) *RetryHandler {
    return &RetryHandler{
        maxRetries: maxRetries,
        backoff:    backoff,
        handler:    handler,
        dlqTopic:   dlqTopic,
        producer:   producer,
    }
}

func (r *RetryHandler) Handle(ctx context.Context, msg events.Message) error {
    var lastErr error

    for attempt := 0; attempt <= r.maxRetries; attempt++ {
        if attempt > 0 {
            time.Sleep(r.backoff * time.Duration(attempt))
            log.Printf("Retry attempt %d/%d for message at offset %d", attempt, r.maxRetries, msg.Offset)
        }

        if err := r.handler.Handle(ctx, msg); err != nil {
            lastErr = err
            log.Printf("Error handling message (attempt %d): %v", attempt+1, err)
            continue
        }

        return nil // Success
    }

    // All retries exhausted, send to DLQ
    if r.dlqTopic != "" {
        if err := r.sendToDLQ(ctx, msg, lastErr); err != nil {
            log.Printf("Failed to send message to DLQ: %v", err)
        }
    }

    return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (r *RetryHandler) sendToDLQ(ctx context.Context, msg events.Message, err error) error {
    dlqMessage := map[string]interface{}{
        "originalMessage": string(msg.Value),
        "error": map[string]interface{}{
            "message": err.Error(),
        },
        "failureCount": r.maxRetries + 1,
        "timestamp":    time.Now().UTC(),
    }

    dlqBytes, _ := json.Marshal(dlqMessage)
    return r.producer.Publish(ctx, r.dlqTopic, dlqBytes)
}
```

## Testing

```go
// internal/kafka/consumer_test.go
package kafka_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "myapp/internal/events"
    "myapp/internal/kafka"
)

type MockHandler struct {
    mock.Mock
}

func (m *MockHandler) Handle(ctx context.Context, msg events.Message) error {
    args := m.Called(ctx, msg)
    return args.Error(0)
}

func TestKafkaConsumer(t *testing.T) {
    // Integration test with embedded Kafka or use testcontainers
    t.Skip("Requires Kafka instance")

    config := kafka.NewConfig([]string{"localhost:9092"}, "test-group")
    handler := new(MockHandler)

    handler.On("Handle", mock.Anything, mock.Anything).Return(nil)

    consumer, err := kafka.NewConsumer(config, []string{"test-topic"}, handler)
    assert.NoError(t, err)
    defer consumer.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    go consumer.Subscribe(ctx)

    time.Sleep(2 * time.Second)

    handler.AssertExpectations(t)
}
```

## Best Practices

1. **Graceful Shutdown**: Always close connections and consumers properly
2. **Error Handling**: Implement retry logic and dead letter queues
3. **Message Validation**: Validate message structure before processing
4. **Idempotency**: Design handlers to be idempotent (handle duplicate messages)
5. **Correlation IDs**: Use correlation IDs to trace requests across services
6. **Monitoring**: Add metrics and logging for message processing
7. **Backpressure**: Implement consumer backpressure mechanisms
8. **Schema Evolution**: Version your message schemas for compatibility

## Related Patterns

- **AsyncAPI Specification Pattern**: Language-agnostic event-driven API specifications
- **REST API Implementation Pattern (Go)**: Synchronous API implementation
- **gRPC Implementation Pattern (Go)**: High-performance RPC communication
