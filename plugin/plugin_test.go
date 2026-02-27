package plugin

import (
	"bytes"
	"context"
	"crypto/tls"
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

// quicListenForTest creates a QUIC listener on an ephemeral port and returns
// it so we can discover the actual port number.
func quicListenForTest(tlsConfig *tls.Config) (*quic.Listener, error) {
	return quic.ListenAddr("localhost:0", tlsConfig, &quic.Config{})
}
