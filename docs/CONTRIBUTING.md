# Contributing to sdk-go

## Prerequisites

- Go 1.23+
- `gofmt`, `go vet`

## Development Setup

```bash
git clone https://github.com/orchestra-mcp/sdk-go.git
cd sdk-go
go mod download
go build ./...
```

This module depends on `gen-go` (generated Protobuf types). During development in the mono-repo, a `replace` directive in `go.mod` points to the local copy. When releasing as a standalone repo, the replace is removed and a tagged version of `gen-go` is used.

## Running Tests

```bash
go test ./...
```

Tests use `testify` for assertions. Run with verbose output:

```bash
go test -v ./...
```

## Package Guidelines

- **plugin/** -- Core runtime. Changes here affect all plugins. Be conservative.
- **helpers/** -- Utility functions. Keep them simple, well-tested, and dependency-free.
- **types/** -- Domain types shared across plugins. Must be serializable to/from JSON.
- **protocol/** -- MCP JSON-RPC types for the transport layer. Must match the MCP specification.

## Code Style

- Run `gofmt` on all files. CI will reject unformatted code.
- Run `go vet ./...` to catch common issues.
- All exported functions must have doc comments.
- Error handling: never ignore errors. Wrap with context: `fmt.Errorf("context: %w", err)`.
- Use `context.Context` through the entire call chain.
- Use interfaces for testability. The `StorageHandler` and `LifecycleHooks` interfaces are examples.

## Testing Approach

- Table-driven tests for deterministic functions (helpers, validation).
- Mock the `Sender` interface for testing transport/handler logic without a real QUIC connection.
- Test framing roundtrips: marshal, frame, unframe, unmarshal.
- Test mTLS: verify mutual authentication, ensure unsigned certs are rejected.

## Pull Request Process

1. Fork the repository and create a feature branch from `main`.
2. Write tests for new functionality. Aim for full coverage on helpers and types.
3. Run `go test ./...` and `go vet ./...` before submitting.
4. Keep PRs focused -- one feature or fix per PR.
5. Update doc comments if the public API changes.

## Related Repositories

- [orchestra-mcp/proto](https://github.com/orchestra-mcp/proto) -- Protobuf schema
- [orchestra-mcp/gen-go](https://github.com/orchestra-mcp/gen-go) -- Generated Go types
- [orchestra-mcp/orchestrator](https://github.com/orchestra-mcp/orchestrator) -- Central hub
- [orchestra-mcp](https://github.com/orchestra-mcp) -- Organization home
