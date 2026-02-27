package plugin

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"sync"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/google/uuid"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/types/known/structpb"
)

// ToolHandler is the function signature for handling tool invocations.
type ToolHandler func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error)

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

// Server is a QUIC server that accepts plugin protocol requests (tool calls,
// list_tools, health, boot, shutdown, storage) from an orchestrator or other
// clients.
type Server struct {
	addr      string
	tlsConfig *tls.Config
	lifecycle LifecycleHooks

	mu             sync.RWMutex
	tools          map[string]*toolEntry
	storageHandler StorageHandler

	// actualAddr is the address the server is actually listening on, populated
	// after ListenAndServe binds the socket. This is useful when addr is
	// "localhost:0" and the OS assigns a random port.
	actualAddr string
	readyCh    chan struct{}
}

// NewServer creates a new QUIC server bound to the given address.
func NewServer(addr string, tlsConfig *tls.Config) *Server {
	return &Server{
		addr:      addr,
		tlsConfig: tlsConfig,
		lifecycle: noopLifecycle{},
		tools:     make(map[string]*toolEntry),
		readyCh:   make(chan struct{}),
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

// ListenAndServe starts the QUIC listener and accepts connections until the
// context is cancelled. Each connection is handled in its own goroutine.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := quic.ListenAddr(s.addr, s.tlsConfig, &quic.Config{})
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

// handleStream reads a single PluginRequest from a bidirectional stream,
// dispatches it, writes the PluginResponse, and closes the stream.
func (s *Server) handleStream(ctx context.Context, stream quic.Stream) {
	defer stream.Close()

	var req pluginv1.PluginRequest
	if err := ReadMessage(stream, &req); err != nil {
		log.Printf("read request: %v", err)
		return
	}

	resp := s.dispatch(ctx, &req)
	resp.RequestId = req.RequestId

	if err := WriteMessage(stream, resp); err != nil {
		log.Printf("write response: %v", err)
	}
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

	case *pluginv1.PluginRequest_StorageRead:
		return s.handleStorageRead(ctx, r.StorageRead)

	case *pluginv1.PluginRequest_StorageWrite:
		return s.handleStorageWrite(ctx, r.StorageWrite)

	case *pluginv1.PluginRequest_StorageDelete:
		return s.handleStorageDelete(ctx, r.StorageDelete)

	case *pluginv1.PluginRequest_StorageList:
		return s.handleStorageList(ctx, r.StorageList)

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

// handleListTools returns all registered tool definitions.
func (s *Server) handleListTools() *pluginv1.PluginResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]*pluginv1.ToolDefinition, 0, len(s.tools))
	for _, entry := range s.tools {
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

// newRequestID generates a new UUID for request correlation.
func newRequestID() string {
	return uuid.New().String()
}
