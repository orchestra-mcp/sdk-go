package plugin

import (
	"context"
	"crypto/tls"
	"fmt"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"github.com/quic-go/quic-go"
)

// OrchestratorClient is a QUIC client for communicating with the orchestrator
// or another plugin server. Each Send call opens a new bidirectional QUIC
// stream, writes the request, reads the response, and closes the stream.
type OrchestratorClient struct {
	conn quic.Connection
}

// NewOrchestratorClient dials a QUIC connection to the given address using the
// provided TLS configuration.
func NewOrchestratorClient(ctx context.Context, addr string, tlsConfig *tls.Config) (*OrchestratorClient, error) {
	conn, err := quic.DialAddr(ctx, addr, tlsConfig, &quic.Config{})
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return &OrchestratorClient{conn: conn}, nil
}

// Send opens a new bidirectional QUIC stream, writes the PluginRequest, reads
// the PluginResponse, and returns it. Each call is one complete RPC exchange.
func (c *OrchestratorClient) Send(ctx context.Context, req *pluginv1.PluginRequest) (*pluginv1.PluginResponse, error) {
	stream, err := c.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}
	defer stream.Close()

	if err := WriteMessage(stream, req); err != nil {
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Signal that we are done writing so the server knows the request is complete.
	if err := stream.Close(); err != nil {
		// Close on a QUIC stream may already have been called by defer, but we
		// need to signal write-side FIN. Use CancelWrite if Close fails.
	}

	var resp pluginv1.PluginResponse
	if err := ReadMessage(stream, &resp); err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return &resp, nil
}

// Close terminates the underlying QUIC connection.
func (c *OrchestratorClient) Close() error {
	return c.conn.CloseWithError(0, "client closed")
}
