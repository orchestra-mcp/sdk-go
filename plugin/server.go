package plugin

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToolHandler is the function signature for handling tool invocations.
type ToolHandler func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error)

// StreamingToolHandler is the function signature for streaming tool invocations.
// The handler sends chunks via the provided channel and returns when done.
// Each []byte chunk is sent as a StreamChunk to the client. The content_type
// parameter in StreamStart determines how chunks are interpreted.
type StreamingToolHandler func(ctx context.Context, req *pluginv1.StreamStart, chunks chan<- []byte) error

// EventHandler is called when an event is delivered to a subscriber.
type EventHandler func(ctx context.Context, event *pluginv1.EventDelivery)

// PromptHandler is the function signature for handling prompt get requests.
type PromptHandler func(ctx context.Context, req *pluginv1.PromptGetRequest) (*pluginv1.PromptGetResponse, error)

// StorageHandler defines the interface that storage plugins must implement to
// handle storage requests dispatched by the orchestrator.
type StorageHandler interface {
	Read(ctx context.Context, req *pluginv1.StorageReadRequest) (*pluginv1.StorageReadResponse, error)
	Write(ctx context.Context, req *pluginv1.StorageWriteRequest) (*pluginv1.StorageWriteResponse, error)
	Delete(ctx context.Context, req *pluginv1.StorageDeleteRequest) (*pluginv1.StorageDeleteResponse, error)
	List(ctx context.Context, req *pluginv1.StorageListRequest) (*pluginv1.StorageListResponse, error)
}

// toolEntry holds a registered tool definition alongside its handler.
type toolEntry struct {
	definition *pluginv1.ToolDefinition
	handler    ToolHandler
}

// streamingToolEntry holds a registered streaming tool definition alongside its handler.
type streamingToolEntry struct {
	definition *pluginv1.ToolDefinition
	handler    StreamingToolHandler
}

// promptEntry holds a registered prompt definition alongside its handler.
type promptEntry struct {
	definition *pluginv1.PromptDefinition
	handler    PromptHandler
}

// subscription holds state for an event subscription.
type subscription struct {
	id      string
	topic   string
	filters map[string]string
	handler EventHandler
}

// Server is a QUIC server that accepts plugin protocol requests (tool calls,
// list_tools, health, boot, shutdown, storage) from an orchestrator or other
// clients.
type Server struct {
	addr      string
	tlsConfig *tls.Config
	lifecycle LifecycleHooks

	mu             sync.RWMutex
	tools          map[string]*toolEntry
	streamingTools map[string]*streamingToolEntry
	prompts        map[string]*promptEntry
	storageHandler StorageHandler
	subscriptions  map[string]*subscription // subscription_id → subscription
	activeStreams   map[string]context.CancelFunc // stream_id → cancel

	// actualAddr is the address the server is actually listening on, populated
	// after ListenAndServe binds the socket. This is useful when addr is
	// "localhost:0" and the OS assigns a random port.
	actualAddr string
	readyCh    chan struct{}
}

// NewServer creates a new QUIC server bound to the given address.
func NewServer(addr string, tlsConfig *tls.Config) *Server {
	return &Server{
		addr:           addr,
		tlsConfig:      tlsConfig,
		lifecycle:      noopLifecycle{},
		tools:          make(map[string]*toolEntry),
		streamingTools: make(map[string]*streamingToolEntry),
		prompts:        make(map[string]*promptEntry),
		subscriptions:  make(map[string]*subscription),
		activeStreams:   make(map[string]context.CancelFunc),
		readyCh:        make(chan struct{}),
	}
}

// SetLifecycleHooks sets the lifecycle hook implementation for boot/shutdown.
func (s *Server) SetLifecycleHooks(hooks LifecycleHooks) {
	s.lifecycle = hooks
}

// SetStorageHandler sets the storage handler for dispatching storage requests.
func (s *Server) SetStorageHandler(h StorageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storageHandler = h
}

// RegisterTool adds a tool to the server's registry.
func (s *Server) RegisterTool(name string, description string, schema *structpb.Struct, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tools[name] = &toolEntry{
		definition: &pluginv1.ToolDefinition{
			Name:        name,
			Description: description,
			InputSchema: schema,
		},
		handler: handler,
	}
}

// RegisterStreamingTool adds a streaming tool to the server's registry.
func (s *Server) RegisterStreamingTool(name string, description string, schema *structpb.Struct, handler StreamingToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streamingTools[name] = &streamingToolEntry{
		definition: &pluginv1.ToolDefinition{
			Name:        name,
			Description: description,
			InputSchema: schema,
		},
		handler: handler,
	}
}

// Subscribe registers a local event handler for a topic. Returns subscription ID.
func (s *Server) Subscribe(topic string, filters map[string]string, handler EventHandler) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := uuid.New().String()
	s.subscriptions[id] = &subscription{
		id:      id,
		topic:   topic,
		filters: filters,
		handler: handler,
	}
	return id
}

// Unsubscribe removes a local event subscription.
func (s *Server) Unsubscribe(subscriptionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subscriptions, subscriptionID)
}

// RegisterPrompt adds a prompt to the server's registry.
func (s *Server) RegisterPrompt(name string, description string, args []*pluginv1.PromptArgument, handler PromptHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prompts[name] = &promptEntry{
		definition: &pluginv1.PromptDefinition{
			Name:        name,
			Description: description,
			Arguments:   args,
		},
		handler: handler,
	}
}

// ListenAndServe starts the QUIC listener and accepts connections until the
// context is cancelled. Each connection is handled in its own goroutine.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := quic.ListenAddr(s.addr, s.tlsConfig, &quic.Config{
		MaxIdleTimeout:  5 * time.Minute,
		KeepAlivePeriod: 15 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("quic listen %s: %w", s.addr, err)
	}
	defer listener.Close()

	// Store the actual address (important when addr is "localhost:0").
	s.actualAddr = listener.Addr().String()
	if s.readyCh != nil {
		close(s.readyCh)
	}

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // graceful shutdown
			}
			return fmt.Errorf("accept connection: %w", err)
		}
		go s.handleConnection(ctx, conn)
	}
}

// ActualAddr returns the address the server is listening on. This blocks until
// the server has started listening. Returns empty string if the server hasn't
// started.
func (s *Server) ActualAddr() string {
	if s.readyCh != nil {
		<-s.readyCh
	}
	return s.actualAddr
}

// handleConnection accepts streams on a single QUIC connection.
func (s *Server) handleConnection(ctx context.Context, conn quic.Connection) {
	defer conn.CloseWithError(0, "")
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			return // connection closed
		}
		go s.handleStream(ctx, stream)
	}
}

// ListenAndServeTCP starts a plain TCP listener on the given address and serves
// the same length-delimited Protobuf protocol as ListenAndServe (QUIC).
// This is used by clients (e.g. Swift/macOS) that cannot open raw QUIC streams
// via Network.framework. Each TCP connection handles one request/response.
func (s *Server) ListenAndServeTCP(ctx context.Context, addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp listen %s: %w", addr, err)
	}
	go func() {
		<-ctx.Done()
		ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("tcp accept: %w", err)
		}
		go s.handleRWC(ctx, conn)
	}
}

// handleRWC handles a single io.ReadWriteCloser (TCP conn or QUIC stream).
func (s *Server) handleRWC(ctx context.Context, rwc io.ReadWriteCloser) {
	defer rwc.Close()

	var req pluginv1.PluginRequest
	if err := ReadMessage(rwc, &req); err != nil {
		log.Printf("read request: %v", err)
		return
	}

	if ss := req.GetStreamStart(); ss != nil {
		s.handleStreamingTool(ctx, &req, ss, rwc)
		return
	}

	if sc := req.GetStreamCancel(); sc != nil {
		s.mu.RLock()
		cancelFn, ok := s.activeStreams[sc.StreamId]
		s.mu.RUnlock()
		if ok {
			cancelFn()
		}
		return
	}

	resp := s.dispatch(ctx, &req)
	resp.RequestId = req.RequestId
	if err := WriteMessage(rwc, resp); err != nil {
		log.Printf("write response: %v", err)
	}
}

// handleStream reads a single PluginRequest from a bidirectional stream,
// dispatches it, writes the PluginResponse, and closes the stream.
// For StreamStart requests, the stream is kept open for multiple responses.
func (s *Server) handleStream(ctx context.Context, stream quic.Stream) {
	s.handleRWC(ctx, stream)
}

// dispatch routes a PluginRequest to the appropriate handler based on the
// oneof request type.
func (s *Server) dispatch(ctx context.Context, req *pluginv1.PluginRequest) *pluginv1.PluginResponse {
	switch r := req.Request.(type) {

	case *pluginv1.PluginRequest_Register:
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_Register{
				Register: &pluginv1.RegistrationResult{
					Accepted: true,
				},
			},
		}

	case *pluginv1.PluginRequest_Boot:
		var config map[string]string
		if r.Boot != nil {
			config = r.Boot.Config
		}
		err := s.lifecycle.OnBoot(config)
		result := &pluginv1.BootResult{Ready: err == nil}
		if err != nil {
			result.Error = err.Error()
		}
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_Boot{Boot: result},
		}

	case *pluginv1.PluginRequest_Shutdown:
		err := s.lifecycle.OnShutdown()
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_Shutdown{
				Shutdown: &pluginv1.ShutdownResult{Clean: err == nil},
			},
		}

	case *pluginv1.PluginRequest_Health:
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_Health{
				Health: &pluginv1.HealthResult{
					Status:  pluginv1.HealthResult_STATUS_HEALTHY,
					Message: "ok",
				},
			},
		}

	case *pluginv1.PluginRequest_ListTools:
		return s.handleListTools()

	case *pluginv1.PluginRequest_ToolCall:
		return s.handleToolCall(ctx, r.ToolCall)

	case *pluginv1.PluginRequest_ListPrompts:
		return s.handleListPrompts()

	case *pluginv1.PluginRequest_PromptGet:
		return s.handlePromptGet(ctx, r.PromptGet)

	case *pluginv1.PluginRequest_StorageRead:
		return s.handleStorageRead(ctx, r.StorageRead)

	case *pluginv1.PluginRequest_StorageWrite:
		return s.handleStorageWrite(ctx, r.StorageWrite)

	case *pluginv1.PluginRequest_StorageDelete:
		return s.handleStorageDelete(ctx, r.StorageDelete)

	case *pluginv1.PluginRequest_StorageList:
		return s.handleStorageList(ctx, r.StorageList)

	case *pluginv1.PluginRequest_Subscribe:
		// Subscribe is handled by the orchestrator; plugins that receive it
		// just acknowledge. Local subscriptions are managed via Server.Subscribe().
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_EventDelivery{
				EventDelivery: &pluginv1.EventDelivery{
					SubscriptionId: r.Subscribe.SubscriptionId,
					Topic:          r.Subscribe.Topic,
				},
			},
		}

	case *pluginv1.PluginRequest_Unsubscribe:
		s.Unsubscribe(r.Unsubscribe.SubscriptionId)
		return &pluginv1.PluginResponse{}

	case *pluginv1.PluginRequest_Publish:
		// Publish is typically sent TO the orchestrator, not received.
		// If we receive it, deliver to local subscriptions.
		s.handleEventDelivery(ctx, &pluginv1.EventDelivery{
			Topic:        r.Publish.Topic,
			EventType:    r.Publish.EventType,
			Payload:      r.Publish.Payload,
			SourcePlugin: r.Publish.SourcePlugin,
		})
		return &pluginv1.PluginResponse{}

	default:
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_ToolCall{
				ToolCall: &pluginv1.ToolResponse{
					Success:      false,
					ErrorCode:    "unknown_request",
					ErrorMessage: "unrecognized request type",
				},
			},
		}
	}
}

// handleListTools returns all registered tool definitions (both regular and streaming).
func (s *Server) handleListTools() *pluginv1.PluginResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]*pluginv1.ToolDefinition, 0, len(s.tools)+len(s.streamingTools))
	for _, entry := range s.tools {
		tools = append(tools, entry.definition)
	}
	for _, entry := range s.streamingTools {
		tools = append(tools, entry.definition)
	}
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_ListTools{
			ListTools: &pluginv1.ListToolsResponse{Tools: tools},
		},
	}
}

// handleToolCall dispatches a tool invocation to the registered handler.
func (s *Server) handleToolCall(ctx context.Context, req *pluginv1.ToolRequest) *pluginv1.PluginResponse {
	s.mu.RLock()
	entry, ok := s.tools[req.ToolName]
	s.mu.RUnlock()

	if !ok {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_ToolCall{
				ToolCall: &pluginv1.ToolResponse{
					Success:      false,
					ErrorCode:    "tool_not_found",
					ErrorMessage: fmt.Sprintf("tool %q not found", req.ToolName),
				},
			},
		}
	}

	result, err := entry.handler(ctx, req)
	if err != nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_ToolCall{
				ToolCall: &pluginv1.ToolResponse{
					Success:      false,
					ErrorCode:    "handler_error",
					ErrorMessage: err.Error(),
				},
			},
		}
	}

	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_ToolCall{ToolCall: result},
	}
}

// handleListPrompts returns all registered prompt definitions.
func (s *Server) handleListPrompts() *pluginv1.PluginResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := make([]*pluginv1.PromptDefinition, 0, len(s.prompts))
	for _, entry := range s.prompts {
		prompts = append(prompts, entry.definition)
	}
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_ListPrompts{
			ListPrompts: &pluginv1.ListPromptsResponse{Prompts: prompts},
		},
	}
}

// handlePromptGet dispatches a prompt get request to the registered handler.
func (s *Server) handlePromptGet(ctx context.Context, req *pluginv1.PromptGetRequest) *pluginv1.PluginResponse {
	s.mu.RLock()
	entry, ok := s.prompts[req.PromptName]
	s.mu.RUnlock()

	if !ok {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_PromptGet{
				PromptGet: &pluginv1.PromptGetResponse{
					Description: fmt.Sprintf("prompt %q not found", req.PromptName),
				},
			},
		}
	}

	result, err := entry.handler(ctx, req)
	if err != nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_PromptGet{
				PromptGet: &pluginv1.PromptGetResponse{
					Description: fmt.Sprintf("prompt handler error: %v", err),
				},
			},
		}
	}

	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_PromptGet{PromptGet: result},
	}
}

// storageErrorResponse returns a generic error response for storage requests
// when no handler is registered.
func storageNoHandlerResponse() *pluginv1.PluginResponse {
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_StorageRead{
			StorageRead: &pluginv1.StorageReadResponse{},
		},
	}
}

// handleStorageRead dispatches a storage read request to the registered handler.
func (s *Server) handleStorageRead(ctx context.Context, req *pluginv1.StorageReadRequest) *pluginv1.PluginResponse {
	s.mu.RLock()
	handler := s.storageHandler
	s.mu.RUnlock()

	if handler == nil {
		return storageNoHandlerResponse()
	}

	resp, err := handler.Read(ctx, req)
	if err != nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageRead{
				StorageRead: &pluginv1.StorageReadResponse{},
			},
		}
	}
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_StorageRead{StorageRead: resp},
	}
}

// handleStorageWrite dispatches a storage write request to the registered handler.
func (s *Server) handleStorageWrite(ctx context.Context, req *pluginv1.StorageWriteRequest) *pluginv1.PluginResponse {
	s.mu.RLock()
	handler := s.storageHandler
	s.mu.RUnlock()

	if handler == nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageWrite{
				StorageWrite: &pluginv1.StorageWriteResponse{
					Success: false,
					Error:   "no storage handler registered",
				},
			},
		}
	}

	resp, err := handler.Write(ctx, req)
	if err != nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageWrite{
				StorageWrite: &pluginv1.StorageWriteResponse{
					Success: false,
					Error:   err.Error(),
				},
			},
		}
	}
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_StorageWrite{StorageWrite: resp},
	}
}

// handleStorageDelete dispatches a storage delete request to the registered handler.
func (s *Server) handleStorageDelete(ctx context.Context, req *pluginv1.StorageDeleteRequest) *pluginv1.PluginResponse {
	s.mu.RLock()
	handler := s.storageHandler
	s.mu.RUnlock()

	if handler == nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageDelete{
				StorageDelete: &pluginv1.StorageDeleteResponse{Success: false},
			},
		}
	}

	resp, err := handler.Delete(ctx, req)
	if err != nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageDelete{
				StorageDelete: &pluginv1.StorageDeleteResponse{Success: false},
			},
		}
	}
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_StorageDelete{StorageDelete: resp},
	}
}

// handleStorageList dispatches a storage list request to the registered handler.
func (s *Server) handleStorageList(ctx context.Context, req *pluginv1.StorageListRequest) *pluginv1.PluginResponse {
	s.mu.RLock()
	handler := s.storageHandler
	s.mu.RUnlock()

	if handler == nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageList{
				StorageList: &pluginv1.StorageListResponse{},
			},
		}
	}

	resp, err := handler.List(ctx, req)
	if err != nil {
		return &pluginv1.PluginResponse{
			Response: &pluginv1.PluginResponse_StorageList{
				StorageList: &pluginv1.StorageListResponse{},
			},
		}
	}
	return &pluginv1.PluginResponse{
		Response: &pluginv1.PluginResponse_StorageList{StorageList: resp},
	}
}

// handleStreamingTool runs a streaming tool handler, sending StreamChunks over
// the QUIC stream and a final StreamEnd when the handler completes or is cancelled.
func (s *Server) handleStreamingTool(ctx context.Context, req *pluginv1.PluginRequest, ss *pluginv1.StreamStart, stream io.ReadWriteCloser) {
	s.mu.RLock()
	entry, ok := s.streamingTools[ss.ToolName]
	s.mu.RUnlock()

	if !ok {
		resp := &pluginv1.PluginResponse{
			RequestId: req.RequestId,
			Response: &pluginv1.PluginResponse_StreamEnd{
				StreamEnd: &pluginv1.StreamEnd{
					StreamId:     ss.StreamId,
					Success:      false,
					ErrorCode:    "tool_not_found",
					ErrorMessage: fmt.Sprintf("streaming tool %q not found", ss.ToolName),
				},
			},
		}
		if err := WriteMessage(stream, resp); err != nil {
			log.Printf("write stream end (not found): %v", err)
		}
		return
	}

	// Register cancellation for this stream.
	streamCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	s.mu.Lock()
	s.activeStreams[ss.StreamId] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.activeStreams, ss.StreamId)
		s.mu.Unlock()
	}()

	// Channel for handler to send chunks.
	chunks := make(chan []byte, 16)

	// Run the handler in a goroutine.
	handlerErr := make(chan error, 1)
	go func() {
		defer close(chunks)
		handlerErr <- entry.handler(streamCtx, ss, chunks)
	}()

	// Send chunks as they arrive.
	var seq int64
	for data := range chunks {
		resp := &pluginv1.PluginResponse{
			RequestId: req.RequestId,
			Response: &pluginv1.PluginResponse_StreamChunk{
				StreamChunk: &pluginv1.StreamChunk{
					StreamId:    ss.StreamId,
					Data:        data,
					ContentType: "application/octet-stream",
					Sequence:    seq,
				},
			},
		}
		if err := WriteMessage(stream, resp); err != nil {
			log.Printf("write stream chunk: %v", err)
			cancel()
			return
		}
		seq++
	}

	// Wait for handler to finish and send StreamEnd.
	err := <-handlerErr
	endResp := &pluginv1.PluginResponse{
		RequestId: req.RequestId,
		Response: &pluginv1.PluginResponse_StreamEnd{
			StreamEnd: &pluginv1.StreamEnd{
				StreamId:    ss.StreamId,
				Success:     err == nil,
				TotalChunks: seq,
			},
		},
	}
	if err != nil {
		endResp.GetStreamEnd().ErrorCode = "handler_error"
		endResp.GetStreamEnd().ErrorMessage = err.Error()
	}
	if err := WriteMessage(stream, endResp); err != nil {
		log.Printf("write stream end: %v", err)
	}
}

// handleEventDelivery routes an incoming event to matching local subscriptions.
func (s *Server) handleEventDelivery(ctx context.Context, event *pluginv1.EventDelivery) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, sub := range s.subscriptions {
		if sub.topic != event.Topic {
			continue
		}
		// Check filters — all filter keys must match event payload fields.
		if len(sub.filters) > 0 && event.Payload != nil {
			match := true
			for k, v := range sub.filters {
				if field, ok := event.Payload.Fields[k]; !ok || field.GetStringValue() != v {
					match = false
					break
				}
			}
			if !match {
				continue
			}
		}
		go sub.handler(ctx, event)
	}
}

// newRequestID generates a new UUID for request correlation.
func newRequestID() string {
	return uuid.New().String()
}
