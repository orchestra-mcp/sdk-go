package plugin

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/quic-go/quic-go"
	"google.golang.org/protobuf/types/known/structpb"
)

// TestFramingRoundTrip verifies that WriteMessage and ReadMessage produce a
// correct length-delimited protobuf roundtrip.
func TestFramingRoundTrip(t *testing.T) {
	original := &pluginv1.PluginRequest{
		RequestId: "test-123",
		Request: &pluginv1.PluginRequest_Health{
			Health: &pluginv1.HealthRequest{},
		},
	}

	var buf bytes.Buffer
	if err := WriteMessage(&buf, original); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}

	if buf.Len() < 4 {
		t.Fatal("buffer too small to contain header")
	}

	var decoded pluginv1.PluginRequest
	if err := ReadMessage(&buf, &decoded); err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}

	if decoded.RequestId != original.RequestId {
		t.Errorf("request_id: got %q, want %q", decoded.RequestId, original.RequestId)
	}
	if decoded.GetHealth() == nil {
		t.Error("expected Health oneof to be set")
	}
}

// TestFramingLargeMessage verifies that messages exceeding MaxMessageSize are
// rejected by ReadMessage.
func TestFramingLargeMessage(t *testing.T) {
	// Fabricate a header claiming ~4GB payload to test the size guard.
	header := []byte{0xFF, 0xFF, 0xFF, 0xFF}
	reader := bytes.NewReader(header)
	var msg pluginv1.PluginRequest
	err := ReadMessage(reader, &msg)
	if err == nil {
		t.Fatal("expected error for oversized message header")
	}
}

// TestCertsGeneration verifies that EnsureCA and GenerateCert create valid
// certificates and keys on disk.
func TestCertsGeneration(t *testing.T) {
	tmpDir := t.TempDir()

	caCert, caKey, err := EnsureCA(tmpDir)
	if err != nil {
		t.Fatalf("EnsureCA: %v", err)
	}
	if caCert == nil || caKey == nil {
		t.Fatal("EnsureCA returned nil cert or key")
	}

	// Verify CA files exist.
	for _, name := range []string{"ca.crt", "ca.key"} {
		p := filepath.Join(tmpDir, name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}

	// Generate a named cert.
	tlsCert, err := GenerateCert(tmpDir, "test-plugin", caCert, caKey)
	if err != nil {
		t.Fatalf("GenerateCert: %v", err)
	}
	if len(tlsCert.Certificate) == 0 {
		t.Fatal("GenerateCert returned empty certificate")
	}

	// Verify named cert files exist.
	for _, name := range []string{"test-plugin.crt", "test-plugin.key"} {
		p := filepath.Join(tmpDir, name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}

	// Calling EnsureCA again should load from disk, not regenerate.
	caCert2, _, err := EnsureCA(tmpDir)
	if err != nil {
		t.Fatalf("EnsureCA (reload): %v", err)
	}
	if !caCert.Equal(caCert2) {
		t.Error("EnsureCA regenerated the CA instead of loading it")
	}
}

// TestServerTLSConfig verifies that ServerTLSConfig produces a valid config.
func TestServerTLSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, err := ServerTLSConfig(tmpDir, "test-server")
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("ServerTLSConfig returned nil")
	}
	if len(cfg.Certificates) == 0 {
		t.Error("no certificates in server TLS config")
	}
	if cfg.ClientCAs == nil {
		t.Error("no ClientCAs in server TLS config")
	}
}

// TestClientTLSConfig verifies that ClientTLSConfig produces a valid config.
func TestClientTLSConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg, err := ClientTLSConfig(tmpDir, "test-client")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}
	if cfg == nil {
		t.Fatal("ClientTLSConfig returned nil")
	}
	if len(cfg.Certificates) == 0 {
		t.Error("no certificates in client TLS config")
	}
	if cfg.RootCAs == nil {
		t.Error("no RootCAs in client TLS config")
	}
}

// TestManifestBuilder verifies the fluent manifest builder.
func TestManifestBuilder(t *testing.T) {
	m := NewManifest("tools.features").
		Version("1.0.0").
		Description("Feature management tools").
		Author("Orchestra").
		Binary("tools-features").
		ProvidesTools("create_feature", "list_features").
		ProvidesStorage("markdown").
		NeedsStorage("markdown").
		NeedsEvents("feature.created").
		NeedsAI("claude").
		NeedsTools("storage_read").
		Build()

	if m.Id != "tools.features" {
		t.Errorf("Id: got %q, want %q", m.Id, "tools.features")
	}
	if m.Version != "1.0.0" {
		t.Errorf("Version: got %q, want %q", m.Version, "1.0.0")
	}
	if m.Language != "go" {
		t.Errorf("Language: got %q, want %q", m.Language, "go")
	}
	if len(m.ProvidesTools) != 2 {
		t.Errorf("ProvidesTools: got %d items, want 2", len(m.ProvidesTools))
	}
	if len(m.ProvidesStorage) != 1 {
		t.Errorf("ProvidesStorage: got %d items, want 1", len(m.ProvidesStorage))
	}
	if len(m.NeedsStorage) != 1 {
		t.Errorf("NeedsStorage: got %d items, want 1", len(m.NeedsStorage))
	}
}

// TestManifestBuilderWithPrompts verifies that ProvidesPrompts works.
func TestManifestBuilderWithPrompts(t *testing.T) {
	m := NewManifest("tools.marketplace").
		Version("0.1.0").
		Description("Marketplace plugin").
		ProvidesTools("install_pack", "list_packs").
		ProvidesPrompts("setup-project", "audit-packs").
		Build()

	if len(m.ProvidesPrompts) != 2 {
		t.Fatalf("ProvidesPrompts: got %d items, want 2", len(m.ProvidesPrompts))
	}
	if m.ProvidesPrompts[0] != "setup-project" {
		t.Errorf("ProvidesPrompts[0]: got %q, want %q", m.ProvidesPrompts[0], "setup-project")
	}
	if m.ProvidesPrompts[1] != "audit-packs" {
		t.Errorf("ProvidesPrompts[1]: got %q, want %q", m.ProvidesPrompts[1], "audit-packs")
	}
}

// TestQUICIntegration tests the full QUIC server/client flow:
// - Start a QUIC server with a registered tool
// - Connect a client
// - Test: Register, ListTools, ToolCall, Health
func TestQUICIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Generate shared certs.
	serverTLS, err := ServerTLSConfig(tmpDir, "integration-server")
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	clientTLS, err := ClientTLSConfig(tmpDir, "integration-client")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	// Create server with a test tool.
	srv := NewServer("localhost:0", serverTLS)

	echoSchema, _ := structpb.NewStruct(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string"},
		},
	})

	srv.RegisterTool("echo", "Echo back the message", echoSchema, func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
		msg := ""
		if req.Arguments != nil {
			if v, ok := req.Arguments.Fields["message"]; ok {
				msg = v.GetStringValue()
			}
		}
		result, _ := structpb.NewStruct(map[string]any{
			"text": "echo: " + msg,
		})
		return &pluginv1.ToolResponse{
			Success: true,
			Result:  result,
		}, nil
	})

	// Register a test prompt.
	srv.RegisterPrompt("greet", "Greeting prompt", []*pluginv1.PromptArgument{
		{Name: "name", Description: "Name to greet", Required: true},
	}, func(ctx context.Context, req *pluginv1.PromptGetRequest) (*pluginv1.PromptGetResponse, error) {
		name := req.Arguments["name"]
		if name == "" {
			name = "World"
		}
		return &pluginv1.PromptGetResponse{
			Description: "Greeting prompt",
			Messages: []*pluginv1.PromptMessage{
				{
					Role:    "user",
					Content: &pluginv1.ContentBlock{Type: "text", Text: "Hello, " + name + "!"},
				},
			},
		}, nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Listen on ephemeral port to discover the actual address, then close
	// so the server can bind to it.
	listener, err := quicListenForTest(serverTLS)
	if err != nil {
		t.Fatalf("listen for port: %v", err)
	}
	actualAddr := listener.Addr().String()
	listener.Close()

	// Start the server on the discovered port.
	srv.addr = actualAddr
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.ListenAndServe(ctx)
	}()

	// Give the server a moment to start accepting connections.
	time.Sleep(150 * time.Millisecond)

	// Connect client.
	client, err := NewOrchestratorClient(ctx, actualAddr, clientTLS)
	if err != nil {
		t.Fatalf("NewOrchestratorClient: %v", err)
	}
	defer client.Close()

	// Test: Register
	t.Run("Register", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "reg-1",
			Request: &pluginv1.PluginRequest_Register{
				Register: &pluginv1.PluginManifest{
					Id:      "test.plugin",
					Version: "0.1.0",
				},
			},
		})
		if err != nil {
			t.Fatalf("Send Register: %v", err)
		}
		reg := resp.GetRegister()
		if reg == nil {
			t.Fatal("expected register response")
		}
		if !reg.Accepted {
			t.Errorf("expected accepted=true, got false: %s", reg.RejectReason)
		}
	})

	// Test: ListTools
	t.Run("ListTools", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "lt-1",
			Request: &pluginv1.PluginRequest_ListTools{
				ListTools: &pluginv1.ListToolsRequest{},
			},
		})
		if err != nil {
			t.Fatalf("Send ListTools: %v", err)
		}
		lt := resp.GetListTools()
		if lt == nil {
			t.Fatal("expected list_tools response")
		}
		if len(lt.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(lt.Tools))
		}
		if lt.Tools[0].Name != "echo" {
			t.Errorf("tool name: got %q, want %q", lt.Tools[0].Name, "echo")
		}
	})

	// Test: ToolCall
	t.Run("ToolCall", func(t *testing.T) {
		args, _ := structpb.NewStruct(map[string]any{
			"message": "hello",
		})
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "tc-1",
			Request: &pluginv1.PluginRequest_ToolCall{
				ToolCall: &pluginv1.ToolRequest{
					ToolName:  "echo",
					Arguments: args,
				},
			},
		})
		if err != nil {
			t.Fatalf("Send ToolCall: %v", err)
		}
		tc := resp.GetToolCall()
		if tc == nil {
			t.Fatal("expected tool_call response")
		}
		if !tc.Success {
			t.Errorf("expected success=true, got false: %s", tc.ErrorMessage)
		}
		text := tc.Result.Fields["text"].GetStringValue()
		if text != "echo: hello" {
			t.Errorf("result text: got %q, want %q", text, "echo: hello")
		}
	})

	// Test: ToolCall for nonexistent tool
	t.Run("ToolCallNotFound", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "tc-2",
			Request: &pluginv1.PluginRequest_ToolCall{
				ToolCall: &pluginv1.ToolRequest{
					ToolName: "nonexistent",
				},
			},
		})
		if err != nil {
			t.Fatalf("Send ToolCall: %v", err)
		}
		tc := resp.GetToolCall()
		if tc == nil {
			t.Fatal("expected tool_call response")
		}
		if tc.Success {
			t.Error("expected success=false for nonexistent tool")
		}
		if tc.ErrorCode != "tool_not_found" {
			t.Errorf("error_code: got %q, want %q", tc.ErrorCode, "tool_not_found")
		}
	})

	// Test: RegisterPrompt + ListPrompts
	t.Run("ListPrompts", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "lp-1",
			Request: &pluginv1.PluginRequest_ListPrompts{
				ListPrompts: &pluginv1.ListPromptsRequest{},
			},
		})
		if err != nil {
			t.Fatalf("Send ListPrompts: %v", err)
		}
		lp := resp.GetListPrompts()
		if lp == nil {
			t.Fatal("expected list_prompts response")
		}
		if len(lp.Prompts) != 1 {
			t.Fatalf("expected 1 prompt, got %d", len(lp.Prompts))
		}
		if lp.Prompts[0].Name != "greet" {
			t.Errorf("prompt name: got %q, want %q", lp.Prompts[0].Name, "greet")
		}
		if len(lp.Prompts[0].Arguments) != 1 {
			t.Fatalf("expected 1 argument, got %d", len(lp.Prompts[0].Arguments))
		}
		if lp.Prompts[0].Arguments[0].Name != "name" {
			t.Errorf("argument name: got %q, want %q", lp.Prompts[0].Arguments[0].Name, "name")
		}
	})

	// Test: PromptGet
	t.Run("PromptGet", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "pg-1",
			Request: &pluginv1.PluginRequest_PromptGet{
				PromptGet: &pluginv1.PromptGetRequest{
					PromptName: "greet",
					Arguments:  map[string]string{"name": "Alice"},
				},
			},
		})
		if err != nil {
			t.Fatalf("Send PromptGet: %v", err)
		}
		pg := resp.GetPromptGet()
		if pg == nil {
			t.Fatal("expected prompt_get response")
		}
		if pg.Description != "Greeting prompt" {
			t.Errorf("description: got %q, want %q", pg.Description, "Greeting prompt")
		}
		if len(pg.Messages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(pg.Messages))
		}
		if pg.Messages[0].Role != "user" {
			t.Errorf("role: got %q, want %q", pg.Messages[0].Role, "user")
		}
		if pg.Messages[0].Content.Text != "Hello, Alice!" {
			t.Errorf("text: got %q, want %q", pg.Messages[0].Content.Text, "Hello, Alice!")
		}
	})

	// Test: PromptGet for nonexistent prompt
	t.Run("PromptGetNotFound", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "pg-2",
			Request: &pluginv1.PluginRequest_PromptGet{
				PromptGet: &pluginv1.PromptGetRequest{
					PromptName: "nonexistent",
				},
			},
		})
		if err != nil {
			t.Fatalf("Send PromptGet: %v", err)
		}
		pg := resp.GetPromptGet()
		if pg == nil {
			t.Fatal("expected prompt_get response")
		}
		// Not-found returns a response with description mentioning "not found"
		if pg.Description == "" {
			t.Error("expected non-empty description for not-found prompt")
		}
	})

	// Test: Health
	t.Run("Health", func(t *testing.T) {
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: "h-1",
			Request: &pluginv1.PluginRequest_Health{
				Health: &pluginv1.HealthRequest{},
			},
		})
		if err != nil {
			t.Fatalf("Send Health: %v", err)
		}
		h := resp.GetHealth()
		if h == nil {
			t.Fatal("expected health response")
		}
		if h.Status != pluginv1.HealthResult_STATUS_HEALTHY {
			t.Errorf("status: got %v, want HEALTHY", h.Status)
		}
	})

	cancel()
}

// TestStreamingIntegration tests the full QUIC streaming flow:
// - Start a QUIC server with a registered streaming tool
// - Connect a client
// - Send StreamStart, receive multiple StreamChunks + StreamEnd
func TestStreamingIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	serverTLS, err := ServerTLSConfig(tmpDir, "stream-server")
	if err != nil {
		t.Fatalf("ServerTLSConfig: %v", err)
	}

	clientTLS, err := ClientTLSConfig(tmpDir, "stream-client")
	if err != nil {
		t.Fatalf("ClientTLSConfig: %v", err)
	}

	srv := NewServer("localhost:0", serverTLS)

	// Register a streaming tool that sends 3 chunks.
	srv.RegisterStreamingTool("count", "Count to N", nil, func(ctx context.Context, req *pluginv1.StreamStart, chunks chan<- []byte) error {
		for i := 0; i < 3; i++ {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case chunks <- []byte(fmt.Sprintf("chunk-%d", i)):
			}
		}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	listener, err := quicListenForTest(serverTLS)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	actualAddr := listener.Addr().String()
	listener.Close()

	srv.addr = actualAddr
	go func() {
		srv.ListenAndServe(ctx)
	}()

	time.Sleep(150 * time.Millisecond)

	client, err := NewOrchestratorClient(ctx, actualAddr, clientTLS)
	if err != nil {
		t.Fatalf("NewOrchestratorClient: %v", err)
	}
	defer client.Close()

	t.Run("StreamingTool", func(t *testing.T) {
		ch, err := client.SendStream(ctx, &pluginv1.PluginRequest{
			RequestId: "stream-1",
			Request: &pluginv1.PluginRequest_StreamStart{
				StreamStart: &pluginv1.StreamStart{
					StreamId: "sid-1",
					ToolName: "count",
				},
			},
		})
		if err != nil {
			t.Fatalf("SendStream: %v", err)
		}

		var chunks []string
		var gotEnd bool
		for resp := range ch {
			if chunk := resp.GetStreamChunk(); chunk != nil {
				chunks = append(chunks, string(chunk.Data))
			}
			if end := resp.GetStreamEnd(); end != nil {
				gotEnd = true
				if !end.Success {
					t.Errorf("StreamEnd success=false: %s", end.ErrorMessage)
				}
				if end.TotalChunks != 3 {
					t.Errorf("TotalChunks: got %d, want 3", end.TotalChunks)
				}
			}
		}

		if !gotEnd {
			t.Error("never received StreamEnd")
		}
		if len(chunks) != 3 {
			t.Fatalf("expected 3 chunks, got %d", len(chunks))
		}
		for i, c := range chunks {
			expected := fmt.Sprintf("chunk-%d", i)
			if c != expected {
				t.Errorf("chunk[%d]: got %q, want %q", i, c, expected)
			}
		}
	})

	t.Run("StreamingToolNotFound", func(t *testing.T) {
		ch, err := client.SendStream(ctx, &pluginv1.PluginRequest{
			RequestId: "stream-2",
			Request: &pluginv1.PluginRequest_StreamStart{
				StreamStart: &pluginv1.StreamStart{
					StreamId: "sid-2",
					ToolName: "nonexistent",
				},
			},
		})
		if err != nil {
			t.Fatalf("SendStream: %v", err)
		}

		for resp := range ch {
			if end := resp.GetStreamEnd(); end != nil {
				if end.Success {
					t.Error("expected success=false for nonexistent streaming tool")
				}
				if end.ErrorCode != "tool_not_found" {
					t.Errorf("error_code: got %q, want %q", end.ErrorCode, "tool_not_found")
				}
				return
			}
		}
		t.Error("never received StreamEnd for nonexistent tool")
	})

	cancel()
}

// TestEventSubscription tests local event subscription and delivery.
func TestEventSubscription(t *testing.T) {
	srv := NewServer("localhost:0", nil)

	received := make(chan *pluginv1.EventDelivery, 1)
	subID := srv.Subscribe("feature.updated", nil, func(ctx context.Context, event *pluginv1.EventDelivery) {
		received <- event
	})

	if subID == "" {
		t.Fatal("Subscribe returned empty ID")
	}

	// Deliver an event via dispatch.
	payload, _ := structpb.NewStruct(map[string]any{
		"feature_id": "FEAT-001",
	})
	ctx := context.Background()
	srv.handleEventDelivery(ctx, &pluginv1.EventDelivery{
		Topic:        "feature.updated",
		EventType:    "status_changed",
		Payload:      payload,
		SourcePlugin: "tools.features",
	})

	select {
	case event := <-received:
		if event.Topic != "feature.updated" {
			t.Errorf("topic: got %q, want %q", event.Topic, "feature.updated")
		}
		if event.EventType != "status_changed" {
			t.Errorf("event_type: got %q, want %q", event.EventType, "status_changed")
		}
		fid := event.Payload.Fields["feature_id"].GetStringValue()
		if fid != "FEAT-001" {
			t.Errorf("feature_id: got %q, want %q", fid, "FEAT-001")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for event delivery")
	}

	// Unsubscribe and verify no more deliveries.
	srv.Unsubscribe(subID)
	srv.handleEventDelivery(ctx, &pluginv1.EventDelivery{
		Topic:     "feature.updated",
		EventType: "deleted",
	})

	select {
	case <-received:
		t.Error("received event after unsubscribe")
	case <-time.After(100 * time.Millisecond):
		// Expected — no event after unsubscribe.
	}
}

// TestEventFilteredSubscription tests that event filters work correctly.
func TestEventFilteredSubscription(t *testing.T) {
	srv := NewServer("localhost:0", nil)

	received := make(chan *pluginv1.EventDelivery, 2)
	srv.Subscribe("build.completed", map[string]string{"project": "my-app"}, func(ctx context.Context, event *pluginv1.EventDelivery) {
		received <- event
	})

	ctx := context.Background()

	// Event matching the filter — should be delivered.
	matchPayload, _ := structpb.NewStruct(map[string]any{"project": "my-app", "status": "success"})
	srv.handleEventDelivery(ctx, &pluginv1.EventDelivery{
		Topic:   "build.completed",
		Payload: matchPayload,
	})

	// Event NOT matching the filter — should be dropped.
	noMatchPayload, _ := structpb.NewStruct(map[string]any{"project": "other-app"})
	srv.handleEventDelivery(ctx, &pluginv1.EventDelivery{
		Topic:   "build.completed",
		Payload: noMatchPayload,
	})

	select {
	case event := <-received:
		proj := event.Payload.Fields["project"].GetStringValue()
		if proj != "my-app" {
			t.Errorf("expected project=my-app, got %q", proj)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for matching event")
	}

	// Verify no second event was delivered.
	select {
	case <-received:
		t.Error("received event that should have been filtered out")
	case <-time.After(100 * time.Millisecond):
		// Expected.
	}
}

// quicListenForTest creates a QUIC listener on an ephemeral port and returns
// it so we can discover the actual port number.
func quicListenForTest(tlsConfig *tls.Config) (*quic.Listener, error) {
	return quic.ListenAddr("localhost:0", tlsConfig, &quic.Config{})
}
