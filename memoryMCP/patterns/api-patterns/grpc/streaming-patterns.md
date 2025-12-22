---
entity_name: gRPC Streaming Patterns
entity_type: gRPC Pattern
language: go
domain: backend
description: Advanced gRPC streaming patterns including server streaming, client streaming, and bidirectional streaming with practical Go examples
tags:
  - gRPC
  - streaming
  - server-streaming
  - client-streaming
  - bidirectional
  - Go
---

# gRPC Streaming Patterns

This pattern demonstrates all gRPC streaming types with practical use cases and Go implementations.

## Stream Types Overview

```
Unary RPC:             Client → Server → Client
Server Streaming:      Client → Server → → → Client
Client Streaming:      Client → → → Server → Client
Bidirectional Stream:  Client ⇄ ⇄ ⇄ Server
```

## Complete Streaming Service

```protobuf
// api/proto/v1/streaming_service.proto
syntax = "proto3";

package streaming.v1;

option go_package = "github.com/yourorg/yourproject/api/gen/go/streaming/v1;streamingv1";

import "google/protobuf/timestamp.proto";

// StreamingService demonstrates all streaming patterns
service StreamingService {
  // Server Streaming: Download large file in chunks
  rpc DownloadFile(DownloadFileRequest) returns (stream FileChunk);

  // Server Streaming: Real-time metrics
  rpc StreamMetrics(StreamMetricsRequest) returns (stream Metric);

  // Server Streaming: Watch resource changes
  rpc WatchResources(WatchResourcesRequest) returns (stream ResourceEvent);

  // Client Streaming: Upload large file
  rpc UploadFile(stream FileChunk) returns (UploadFileResponse);

  // Client Streaming: Batch process records
  rpc ProcessBatch(stream Record) returns (BatchProcessResponse);

  // Bidirectional Streaming: Chat application
  rpc Chat(stream ChatMessage) returns (stream ChatMessage);

  // Bidirectional Streaming: Real-time data sync
  rpc SyncData(stream SyncRequest) returns (stream SyncResponse);

  // Bidirectional Streaming: Interactive query
  rpc InteractiveQuery(stream QueryRequest) returns (stream QueryResponse);
}

// File streaming messages
message DownloadFileRequest {
  string file_id = 1;
  optional int64 offset = 2;
  optional int64 limit = 3;
}

message FileChunk {
  bytes data = 1;
  int64 offset = 2;
  int64 total_size = 3;
  string checksum = 4;
}

message UploadFileResponse {
  string file_id = 1;
  int64 size = 2;
  string checksum = 3;
}

// Metrics streaming messages
message StreamMetricsRequest {
  repeated string metric_names = 1;
  int32 interval_seconds = 2;
}

message Metric {
  string name = 1;
  double value = 2;
  map<string, string> labels = 3;
  google.protobuf.Timestamp timestamp = 4;
}

// Resource watching messages
message WatchResourcesRequest {
  string resource_type = 1;
  repeated string resource_ids = 2;
}

message ResourceEvent {
  EventType type = 1;
  string resource_id = 2;
  bytes data = 3;
  google.protobuf.Timestamp timestamp = 4;
}

enum EventType {
  EVENT_TYPE_UNSPECIFIED = 0;
  EVENT_TYPE_CREATED = 1;
  EVENT_TYPE_UPDATED = 2;
  EVENT_TYPE_DELETED = 3;
}

// Batch processing messages
message Record {
  string id = 1;
  bytes data = 2;
}

message BatchProcessResponse {
  int32 processed_count = 1;
  int32 failed_count = 2;
  repeated RecordError errors = 3;
}

message RecordError {
  string record_id = 1;
  string error = 2;
}

// Chat messages
message ChatMessage {
  string user_id = 1;
  string room_id = 2;
  string content = 3;
  google.protobuf.Timestamp timestamp = 4;
}

// Data sync messages
message SyncRequest {
  SyncOperation operation = 1;
  string entity_type = 2;
  string entity_id = 3;
  bytes data = 4;
}

enum SyncOperation {
  SYNC_OPERATION_UNSPECIFIED = 0;
  SYNC_OPERATION_CREATE = 1;
  SYNC_OPERATION_UPDATE = 2;
  SYNC_OPERATION_DELETE = 3;
}

message SyncResponse {
  SyncStatus status = 1;
  string entity_id = 2;
  optional string error = 3;
}

enum SyncStatus {
  SYNC_STATUS_UNSPECIFIED = 0;
  SYNC_STATUS_SUCCESS = 1;
  SYNC_STATUS_ERROR = 2;
}

// Interactive query messages
message QueryRequest {
  string query = 1;
  map<string, string> parameters = 2;
}

message QueryResponse {
  repeated Row rows = 1;
  bool has_more = 2;
  optional string error = 3;
}

message Row {
  map<string, string> columns = 1;
}
```

## Server Streaming Implementation

```go
// Server streaming: Download file in chunks
func (s *StreamingServiceServer) DownloadFile(req *streamingv1.DownloadFileRequest, stream streamingv1.StreamingService_DownloadFileServer) error {
    ctx := stream.Context()

    file, err := s.fileService.GetFile(ctx, req.FileId)
    if err != nil {
        return status.Error(codes.NotFound, "file not found")
    }

    reader, err := s.fileService.OpenFile(ctx, req.FileId)
    if err != nil {
        return status.Error(codes.Internal, "failed to open file")
    }
    defer reader.Close()

    // Seek to offset if specified
    if req.Offset != nil {
        if _, err := reader.Seek(*req.Offset, io.SeekStart); err != nil {
            return status.Error(codes.InvalidArgument, "invalid offset")
        }
    }

    const chunkSize = 64 * 1024 // 64 KB chunks
    buffer := make([]byte, chunkSize)
    offset := int64(0)
    if req.Offset != nil {
        offset = *req.Offset
    }

    hasher := md5.New()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        n, err := reader.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to read file")
        }

        data := buffer[:n]
        hasher.Write(data)

        chunk := &streamingv1.FileChunk{
            Data:      data,
            Offset:    offset,
            TotalSize: file.Size,
            Checksum:  fmt.Sprintf("%x", hasher.Sum(nil)),
        }

        if err := stream.Send(chunk); err != nil {
            return status.Error(codes.Internal, "failed to send chunk")
        }

        offset += int64(n)

        // Check limit if specified
        if req.Limit != nil && offset >= *req.Offset+*req.Limit {
            break
        }
    }

    return nil
}

// Server streaming: Stream metrics
func (s *StreamingServiceServer) StreamMetrics(req *streamingv1.StreamMetricsRequest, stream streamingv1.StreamingService_StreamMetricsServer) error {
    ctx := stream.Context()
    ticker := time.NewTicker(time.Duration(req.IntervalSeconds) * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            for _, metricName := range req.MetricNames {
                metric, err := s.metricsService.GetMetric(ctx, metricName)
                if err != nil {
                    continue
                }

                if err := stream.Send(&streamingv1.Metric{
                    Name:      metric.Name,
                    Value:     metric.Value,
                    Labels:    metric.Labels,
                    Timestamp: timestamppb.Now(),
                }); err != nil {
                    return status.Error(codes.Internal, "failed to send metric")
                }
            }
        }
    }
}

// Server streaming: Watch resources
func (s *StreamingServiceServer) WatchResources(req *streamingv1.WatchResourcesRequest, stream streamingv1.StreamingService_WatchResourcesServer) error {
    ctx := stream.Context()

    eventChan, err := s.resourceService.Watch(ctx, req.ResourceType, req.ResourceIds)
    if err != nil {
        return status.Error(codes.Internal, "failed to watch resources")
    }

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case event, ok := <-eventChan:
            if !ok {
                return nil
            }

            if err := stream.Send(&streamingv1.ResourceEvent{
                Type:        event.Type,
                ResourceId:  event.ResourceID,
                Data:        event.Data,
                Timestamp:   timestamppb.New(event.Timestamp),
            }); err != nil {
                return status.Error(codes.Internal, "failed to send event")
            }
        }
    }
}
```

## Client Streaming Implementation

```go
// Client streaming: Upload file
func (s *StreamingServiceServer) UploadFile(stream streamingv1.StreamingService_UploadFileServer) error {
    var fileID string
    var totalSize int64
    hasher := md5.New()

    writer, err := s.fileService.CreateFile(stream.Context())
    if err != nil {
        return status.Error(codes.Internal, "failed to create file")
    }
    defer writer.Close()

    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            // Client finished sending
            checksum := fmt.Sprintf("%x", hasher.Sum(nil))

            fileID, err = s.fileService.FinalizeFile(stream.Context(), writer, totalSize, checksum)
            if err != nil {
                return status.Error(codes.Internal, "failed to finalize file")
            }

            return stream.SendAndClose(&streamingv1.UploadFileResponse{
                FileId:   fileID,
                Size:     totalSize,
                Checksum: checksum,
            })
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to receive chunk")
        }

        n, err := writer.Write(chunk.Data)
        if err != nil {
            return status.Error(codes.Internal, "failed to write chunk")
        }

        hasher.Write(chunk.Data)
        totalSize += int64(n)
    }
}

// Client streaming: Process batch
func (s *StreamingServiceServer) ProcessBatch(stream streamingv1.StreamingService_ProcessBatchServer) error {
    var processedCount, failedCount int32
    var errors []*streamingv1.RecordError

    for {
        record, err := stream.Recv()
        if err == io.EOF {
            return stream.SendAndClose(&streamingv1.BatchProcessResponse{
                ProcessedCount: processedCount,
                FailedCount:    failedCount,
                Errors:         errors,
            })
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to receive record")
        }

        if err := s.dataService.ProcessRecord(stream.Context(), record); err != nil {
            failedCount++
            errors = append(errors, &streamingv1.RecordError{
                RecordId: record.Id,
                Error:    err.Error(),
            })
        } else {
            processedCount++
        }
    }
}
```

## Bidirectional Streaming Implementation

```go
// Bidirectional streaming: Chat
func (s *StreamingServiceServer) Chat(stream streamingv1.StreamingService_ChatServer) error {
    ctx := stream.Context()

    // Get user from context (set by auth interceptor)
    userID, ok := ctx.Value("userID").(string)
    if !ok {
        return status.Error(codes.Unauthenticated, "not authenticated")
    }

    // Channel for receiving messages from other users
    messageChan := make(chan *streamingv1.ChatMessage, 10)
    defer close(messageChan)

    // Goroutine to receive messages from client and broadcast
    go func() {
        for {
            msg, err := stream.Recv()
            if err == io.EOF {
                s.chatService.LeaveRoom(ctx, userID, msg.RoomId)
                return
            }
            if err != nil {
                return
            }

            // Broadcast message to all users in room
            if err := s.chatService.BroadcastMessage(ctx, msg); err != nil {
                log.Printf("failed to broadcast message: %v", err)
            }
        }
    }()

    // Subscribe to room messages
    roomChan, err := s.chatService.SubscribeToRoom(ctx, userID)
    if err != nil {
        return status.Error(codes.Internal, "failed to subscribe to room")
    }

    // Send messages from room to client
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case msg := <-roomChan:
            if err := stream.Send(msg); err != nil {
                return status.Error(codes.Internal, "failed to send message")
            }
        }
    }
}

// Bidirectional streaming: Data sync
func (s *StreamingServiceServer) SyncData(stream streamingv1.StreamingService_SyncDataServer) error {
    ctx := stream.Context()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to receive sync request")
        }

        var response *streamingv1.SyncResponse

        switch req.Operation {
        case streamingv1.SyncOperation_SYNC_OPERATION_CREATE:
            entityID, err := s.syncService.Create(ctx, req.EntityType, req.Data)
            if err != nil {
                response = &streamingv1.SyncResponse{
                    Status:   streamingv1.SyncStatus_SYNC_STATUS_ERROR,
                    EntityId: req.EntityId,
                    Error:    ptr(err.Error()),
                }
            } else {
                response = &streamingv1.SyncResponse{
                    Status:   streamingv1.SyncStatus_SYNC_STATUS_SUCCESS,
                    EntityId: entityID,
                }
            }

        case streamingv1.SyncOperation_SYNC_OPERATION_UPDATE:
            err := s.syncService.Update(ctx, req.EntityType, req.EntityId, req.Data)
            if err != nil {
                response = &streamingv1.SyncResponse{
                    Status:   streamingv1.SyncStatus_SYNC_STATUS_ERROR,
                    EntityId: req.EntityId,
                    Error:    ptr(err.Error()),
                }
            } else {
                response = &streamingv1.SyncResponse{
                    Status:   streamingv1.SyncStatus_SYNC_STATUS_SUCCESS,
                    EntityId: req.EntityId,
                }
            }

        case streamingv1.SyncOperation_SYNC_OPERATION_DELETE:
            err := s.syncService.Delete(ctx, req.EntityType, req.EntityId)
            if err != nil {
                response = &streamingv1.SyncResponse{
                    Status:   streamingv1.SyncStatus_SYNC_STATUS_ERROR,
                    EntityId: req.EntityId,
                    Error:    ptr(err.Error()),
                }
            } else {
                response = &streamingv1.SyncResponse{
                    Status:   streamingv1.SyncStatus_SYNC_STATUS_SUCCESS,
                    EntityId: req.EntityId,
                }
            }
        }

        if err := stream.Send(response); err != nil {
            return status.Error(codes.Internal, "failed to send sync response")
        }
    }
}

// Bidirectional streaming: Interactive query
func (s *StreamingServiceServer) InteractiveQuery(stream streamingv1.StreamingService_InteractiveQueryServer) error {
    ctx := stream.Context()

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return status.Error(codes.Internal, "failed to receive query")
        }

        // Execute query and stream results
        rows, err := s.queryService.ExecuteQuery(ctx, req.Query, req.Parameters)
        if err != nil {
            if err := stream.Send(&streamingv1.QueryResponse{
                Error: ptr(err.Error()),
            }); err != nil {
                return status.Error(codes.Internal, "failed to send error")
            }
            continue
        }

        // Stream results in batches
        const batchSize = 100
        batch := make([]*streamingv1.Row, 0, batchSize)

        for rows.Next() {
            row := convertRow(rows)
            batch = append(batch, row)

            if len(batch) >= batchSize {
                if err := stream.Send(&streamingv1.QueryResponse{
                    Rows:    batch,
                    HasMore: true,
                }); err != nil {
                    return status.Error(codes.Internal, "failed to send results")
                }
                batch = batch[:0]
            }
        }

        // Send remaining rows
        if len(batch) > 0 {
            if err := stream.Send(&streamingv1.QueryResponse{
                Rows:    batch,
                HasMore: false,
            }); err != nil {
                return status.Error(codes.Internal, "failed to send results")
            }
        }
    }
}

func ptr[T any](v T) *T {
    return &v
}
```

## Client Implementation Examples

```go
// Client: Server streaming
func downloadFile(client streamingv1.StreamingServiceClient, fileID string) error {
    ctx := context.Background()

    stream, err := client.DownloadFile(ctx, &streamingv1.DownloadFileRequest{
        FileId: fileID,
    })
    if err != nil {
        return err
    }

    file, err := os.Create("downloaded_file")
    if err != nil {
        return err
    }
    defer file.Close()

    for {
        chunk, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        if _, err := file.Write(chunk.Data); err != nil {
            return err
        }
    }

    return nil
}

// Client: Client streaming
func uploadFile(client streamingv1.StreamingServiceClient, filePath string) (*streamingv1.UploadFileResponse, error) {
    ctx := context.Background()

    stream, err := client.UploadFile(ctx)
    if err != nil {
        return nil, err
    }

    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    const chunkSize = 64 * 1024
    buffer := make([]byte, chunkSize)

    for {
        n, err := file.Read(buffer)
        if err == io.EOF {
            break
        }
        if err != nil {
            return nil, err
        }

        chunk := &streamingv1.FileChunk{
            Data: buffer[:n],
        }

        if err := stream.Send(chunk); err != nil {
            return nil, err
        }
    }

    return stream.CloseAndRecv()
}

// Client: Bidirectional streaming (chat)
func chat(client streamingv1.StreamingServiceClient, userID, roomID string) error {
    ctx := context.Background()

    stream, err := client.Chat(ctx)
    if err != nil {
        return err
    }

    // Goroutine to send messages
    go func() {
        scanner := bufio.NewScanner(os.Stdin)
        for scanner.Scan() {
            msg := &streamingv1.ChatMessage{
                UserId:    userID,
                RoomId:    roomID,
                Content:   scanner.Text(),
                Timestamp: timestamppb.Now(),
            }
            if err := stream.Send(msg); err != nil {
                log.Printf("failed to send message: %v", err)
                return
            }
        }
    }()

    // Receive messages
    for {
        msg, err := stream.Recv()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }

        fmt.Printf("[%s] %s: %s\n", msg.Timestamp.AsTime().Format("15:04:05"), msg.UserId, msg.Content)
    }

    return nil
}
```

## Best Practices

### Server Streaming
- Send data in reasonable chunk sizes (64KB - 1MB)
- Respect context cancellation
- Include progress information
- Handle backpressure appropriately

### Client Streaming
- Buffer writes for better performance
- Validate data before finalizing
- Return comprehensive summary
- Handle partial failures gracefully

### Bidirectional Streaming
- Use goroutines for concurrent send/receive
- Implement proper synchronization
- Handle connection lifecycle
- Gracefully handle EOF from either side

### General
- Set reasonable timeouts
- Implement heartbeats for long-lived streams
- Use interceptors for auth and logging
- Test stream cancellation scenarios
- Monitor stream memory usage

## Flow Control

```go
// Implement backpressure
type StreamController struct {
    maxBufferSize int
    buffer        chan *Message
}

func (c *StreamController) Send(msg *Message) error {
    select {
    case c.buffer <- msg:
        return nil
    default:
        return errors.New("buffer full, slow down")
    }
}
```
