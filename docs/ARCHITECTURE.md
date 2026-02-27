# Architecture

## Overview

`sdk-go` is the Go Plugin SDK for Orchestra. Every Go-based plugin imports this module to get QUIC transport, mTLS certificate management, Protobuf framing, and helper utilities. The SDK handles all the boilerplate so plugin authors focus on their tool handlers.

## Package Layout

```
sdk-go/
  plugin/           # Core plugin runtime
    plugin.go        # Plugin struct, PluginBuilder (fluent API), Run lifecycle
    server.go        # QUIC server: accept connections, dispatch requests
    client.go        # OrchestratorClient: QUIC client for sending requests
    framing.go       # Length-delimited Protobuf read/write
    certs.go         # mTLS CA + certificate auto-generation (ed25519)
    manifest.go      # ManifestBuilder: fluent manifest construction
    lifecycle.go     # LifecycleHooks interface (OnBoot, OnShutdown)
    plugin_test.go   # Unit tests
  helpers/           # Argument extraction and result building
    args.go          # GetString, GetInt, GetBool, GetFloat64, GetStringSlice
    results.go       # TextResult, JSONResult, ErrorResult, Markdown formatters
    paths.go         # FeaturePath, ProjectPath, directory constants
    strings.go       # Slugify, NowISO, NewUUID, NewFeatureID
    validate.go      # ValidateRequired, ValidateOneOf
  types/             # Domain types
    feature.go       # FeatureData, FeatureStatus constants, ReviewEntry
    project.go       # ProjectData
    workflow.go      # ValidTransitions map, CanTransition, NextStatuses
  protocol/          # MCP JSON-RPC types
    jsonrpc.go       # JSONRPCRequest, JSONRPCResponse, error codes
    mcp.go           # MCPInitializeResult, MCPToolDefinition, MCPToolResult
```

## Plugin Lifecycle

```
1. New("plugin.id")          Create a PluginBuilder
2. .Version("0.1.0")         Set metadata
3. .ProvidesTools(...)        Declare capabilities
4. .NeedsStorage("markdown")  Declare dependencies
5. .RegisterTool(name, ...)   Register tool handlers
6. .BuildWithTools()           Build the Plugin with tools pre-registered
7. p.ParseFlags()              Parse --orchestrator-addr, --listen-addr, --certs-dir
8. p.Run(ctx)                  Start the plugin:
   a. Generate/load mTLS certs
   b. Start QUIC server
   c. Print "READY <addr>" to stderr
   d. Connect to orchestrator (if configured)
   e. Send Register request
   f. Serve requests until context cancelled
   g. Call OnShutdown, close connections
```

## QUIC Server

The `Server` accepts QUIC connections and dispatches each bidirectional stream as a single request-response exchange:

1. `AcceptStream` -- get a new stream from a connection
2. `ReadMessage` -- read length-delimited `PluginRequest`
3. `dispatch` -- route to the correct handler based on the `oneof` type
4. `WriteMessage` -- write the `PluginResponse`
5. Close the stream

Dispatch routing:

| Request Type | Handler |
|---|---|
| `register` | Immediate accept |
| `boot` | `LifecycleHooks.OnBoot(config)` |
| `shutdown` | `LifecycleHooks.OnShutdown()` |
| `health` | Immediate healthy response |
| `list_tools` | Return all registered `ToolDefinition`s |
| `tool_call` | Look up handler by name, invoke it |
| `storage_read/write/delete/list` | Delegate to `StorageHandler` |

## QUIC Client

`OrchestratorClient` wraps a QUIC connection. Each `Send` call opens a new bidirectional stream, writes the request, reads the response, and closes the stream. This is a one-stream-per-RPC model.

## mTLS Certificates

All QUIC connections use mutual TLS (mTLS) with ed25519 keys:

- `EnsureCA(certsDir)` -- Load or generate a CA at `~/.orchestra/certs/ca.crt` + `ca.key`
- `GenerateCert(certsDir, name, caCert, caKey)` -- Generate a named cert signed by the CA
- `ServerTLSConfig` -- mTLS config requiring client certs, ALPN `"orchestra-plugin"`, TLS 1.3+
- `ClientTLSConfig` -- mTLS config trusting the CA, presenting a client cert

Certificates are generated lazily on first use and cached on disk.

## Framing

Length-delimited Protobuf over any `io.Reader`/`io.Writer`:

```
WriteMessage(w, msg)  ->  [4B big-endian uint32 length][N bytes marshaled proto]
ReadMessage(r, msg)   <-  [4B big-endian uint32 length][N bytes marshaled proto]
```

Maximum message size: 16 MB.

## Helpers

The `helpers` package provides type-safe extraction from `structpb.Struct`:

- **Args**: `GetString`, `GetStringOr`, `GetInt`, `GetFloat64`, `GetBool`, `GetStringSlice`
- **Results**: `TextResult`, `JSONResult`, `ErrorResult`
- **Validation**: `ValidateRequired`, `ValidateOneOf`
- **Strings**: `Slugify`, `NowISO`, `NewUUID`, `NewFeatureID`
- **Paths**: `FeaturePath`, `ProjectPath`
- **Markdown**: `FormatFeatureMD`, `FormatFeatureListMD`, `FormatProjectMD`, `FormatProjectListMD`, `FormatStatusCountsMD`
