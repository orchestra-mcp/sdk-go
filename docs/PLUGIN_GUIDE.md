# Plugin Development Guide

This guide walks through building a Go plugin from scratch using the Orchestra SDK.

## Prerequisites

- Go 1.23+
- The `github.com/orchestra-mcp/sdk-go` and `github.com/orchestra-mcp/gen-go` modules

## Step 1: Create the Module

```bash
mkdir my-plugin && cd my-plugin
go mod init github.com/my-org/my-plugin
go get github.com/orchestra-mcp/sdk-go
go get github.com/orchestra-mcp/gen-go
```

## Step 2: Define a Tool Handler

Create `internal/tools/greet.go`:

```go
package tools

import (
    "context"
    "fmt"

    pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
    "github.com/orchestra-mcp/sdk-go/helpers"
    "google.golang.org/protobuf/types/known/structpb"
)

func GreetSchema() *structpb.Struct {
    s, _ := structpb.NewStruct(map[string]any{
        "type": "object",
        "properties": map[string]any{
            "name": map[string]any{
                "type":        "string",
                "description": "Name to greet",
            },
        },
        "required": []any{"name"},
    })
    return s
}

func Greet(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
    if err := helpers.ValidateRequired(req.Arguments, "name"); err != nil {
        return helpers.ErrorResult("validation_error", err.Error()), nil
    }

    name := helpers.GetString(req.Arguments, "name")
    return helpers.TextResult(fmt.Sprintf("Hello, %s!", name)), nil
}
```

Key patterns:
- Define a `Schema()` function that returns a JSON Schema as `*structpb.Struct`.
- The handler signature is `func(ctx, *ToolRequest) (*ToolResponse, error)`.
- Use `helpers.ValidateRequired` to check required arguments.
- Use `helpers.GetString` / `GetInt` / `GetBool` to extract typed values.
- Return `helpers.TextResult` for success or `helpers.ErrorResult` for errors.

## Step 3: Create the Entry Point

Create `cmd/main.go`:

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/orchestra-mcp/sdk-go/plugin"
    "github.com/my-org/my-plugin/internal/tools"
)

func main() {
    p := plugin.New("tools.greet").
        Version("0.1.0").
        Description("A greeting plugin").
        ProvidesTools("greet").
        RegisterTool("greet", "Greet someone by name", tools.GreetSchema(), tools.Greet).
        BuildWithTools()

    p.ParseFlags()

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() { <-sigCh; cancel() }()

    if err := p.Run(ctx); err != nil {
        log.Fatalf("plugin error: %v", err)
    }
}
```

## Step 4: Build and Test

```bash
go build -o my-plugin ./cmd/
```

Run standalone (the plugin prints `READY <addr>` to stderr):

```bash
./my-plugin --listen-addr=localhost:0 --certs-dir=~/.orchestra/certs
```

Run with the orchestrator by adding it to `plugins.yaml`:

```yaml
plugins:
  - id: tools.greet
    binary: ./my-plugin
    enabled: true
```

## Step 5: Add Storage Access

If your plugin needs to read/write data, declare the dependency and use the orchestrator client:

```go
p := plugin.New("tools.greet").
    NeedsStorage("markdown").
    // ... other config ...
    BuildWithTools()
```

Inside a tool handler, access storage through the plugin's orchestrator client:

```go
func MyTool(p *plugin.Plugin) plugin.ToolHandler {
    return func(ctx context.Context, req *pluginv1.ToolRequest) (*pluginv1.ToolResponse, error) {
        client := p.OrchestratorClient()

        resp, err := client.Send(ctx, &pluginv1.PluginRequest{
            RequestId: uuid.New().String(),
            Request: &pluginv1.PluginRequest_StorageRead{
                StorageRead: &pluginv1.StorageReadRequest{
                    Path:        "my-project/features/FEAT-ABC.md",
                    StorageType: "markdown",
                },
            },
        })
        // ... handle response ...
    }
}
```

## Step 6: Write a Storage Plugin

If your plugin provides storage (rather than consuming it), implement `plugin.StorageHandler`:

```go
type MyStorage struct{}

func (s *MyStorage) Read(ctx context.Context, req *pluginv1.StorageReadRequest) (*pluginv1.StorageReadResponse, error) {
    // Load data from your backend
}
func (s *MyStorage) Write(ctx context.Context, req *pluginv1.StorageWriteRequest) (*pluginv1.StorageWriteResponse, error) {
    // Persist data to your backend
}
func (s *MyStorage) Delete(ctx context.Context, req *pluginv1.StorageDeleteRequest) (*pluginv1.StorageDeleteResponse, error) {
    // Remove data
}
func (s *MyStorage) List(ctx context.Context, req *pluginv1.StorageListRequest) (*pluginv1.StorageListResponse, error) {
    // Enumerate entries
}
```

Then register it:

```go
p := plugin.New("storage.mybackend").
    ProvidesStorage("mybackend").
    SetStorageHandler(&MyStorage{}).
    BuildWithTools()
```

## Plugin Flags

Every plugin gets these flags automatically via `p.ParseFlags()`:

| Flag | Default | Description |
|---|---|---|
| `--orchestrator-addr` | (none) | Address of the orchestrator to connect to |
| `--listen-addr` | `localhost:0` | Address for the QUIC server to listen on |
| `--certs-dir` | `~/.orchestra/certs` | Directory for mTLS certificates |
| `--manifest` | false | Print plugin manifest as JSON and exit |

## Lifecycle Hooks

Implement `plugin.LifecycleHooks` for custom boot/shutdown logic:

```go
type MyHooks struct{}

func (h *MyHooks) OnBoot(config map[string]string) error {
    // Initialize database connections, load state, etc.
    return nil
}

func (h *MyHooks) OnShutdown() error {
    // Flush buffers, close connections, etc.
    return nil
}
```

Register hooks:

```go
p := plugin.New("my.plugin").
    Lifecycle(&MyHooks{}).
    // ...
```

## Installable Plugins

To make your plugin installable via `orchestra install`, ensure:

1. Your `cmd/main.go` supports the `--manifest` flag (automatic if you call `p.ParseFlags()`).
2. The manifest JSON output includes `id`, `provides_tools`, and `provides_storage`.
3. Publish your repo to GitHub with tagged releases or a buildable Go module.

Users install with:

```bash
orchestra install github.com/my-org/my-plugin
```
