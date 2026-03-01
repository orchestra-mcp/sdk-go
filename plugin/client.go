package plugin

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

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
	conn, err := quic.DialAddr(ctx, addr, tlsConfig, &quic.Config{
		MaxIdleTimeout:  5 * time.Minute,
		KeepAlivePeriod: 15 * time.Second,
	})
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

// SendStream opens a QUIC stream, writes the request, and returns a channel
// that receives multiple PluginResponse messages (StreamChunk + StreamEnd).
// The channel is closed after StreamEnd is received or on error. The caller
// can cancel the context to abort the stream.
func (c *OrchestratorClient) SendStream(ctx context.Context, req *pluginv1.PluginRequest) (<-chan *pluginv1.PluginResponse, error) {
	stream, err := c.conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}

	if err := WriteMessage(stream, req); err != nil {
		stream.Close()
		return nil, fmt.Errorf("write request: %w", err)
	}

	ch := make(chan *pluginv1.PluginResponse, 16)

	go func() {
		defer close(ch)
		defer stream.Close()

		for {
			var resp pluginv1.PluginResponse
			if err := ReadMessage(stream, &resp); err != nil {
				return
			}

			select {
			case ch <- &resp:
			case <-ctx.Done():
				// Send cancel if this was a streaming request.
				if ss := req.GetStreamStart(); ss != nil {
					cancelReq := &pluginv1.PluginRequest{
						RequestId: req.RequestId,
						Request: &pluginv1.PluginRequest_StreamCancel{
							StreamCancel: &pluginv1.StreamCancel{
								StreamId: ss.StreamId,
							},
						},
					}
					_ = WriteMessage(stream, cancelReq)
				}
				return
			}

			// Stop reading after StreamEnd.
			if resp.GetStreamEnd() != nil {
				return
			}
		}
	}()

	return ch, nil
}

// Close terminates the underlying QUIC connection.
func (c *OrchestratorClient) Close() error {
	return c.conn.CloseWithError(0, "client closed")
}
