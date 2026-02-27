# Orchestra Plugin SDK for Go

Build Orchestra plugins in Go with QUIC transport, mTLS, and Protobuf framing.

## Install

```bash
go get github.com/orchestra-mcp/sdk-go
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/orchestra-mcp/sdk-go/plugin"
    "github.com/orchestra-mcp/sdk-go/helpers"
)

func main() {
    p := plugin.New("tools.greeter").
        ProvidesTools("greet").
        Build()

    p.RegisterTool("greet", func(ctx context.Context, args map[string]any) (map[string]any, error) {
        name := helpers.StringArg(args, "name", "world")
        return helpers.Success("message", "Hello, "+name+"!"), nil
    })

    p.Run(context.Background())
}
```

## What's Included

- **plugin** -- Plugin builder with QUIC server/client, mTLS certificate generation, and length-delimited Protobuf framing
- **types** -- Feature, Project, and Workflow data structures
- **helpers** -- Argument extraction, result builders, path utilities, validation
- **protocol** -- JSON-RPC and MCP type definitions

## How It Works

1. Your plugin starts a QUIC listener with auto-generated mTLS certificates
2. It prints `READY <address>` to stderr
3. The orchestrator connects and sends `PluginRequest` messages
4. Your registered tool handlers process requests and return `PluginResponse` messages

## Related Packages

| Package | Description |
|---------|-------------|
| [gen-go](https://github.com/orchestra-mcp/gen-go) | Generated Protobuf types |
| [orchestrator](https://github.com/orchestra-mcp/orchestrator) | Central hub that loads plugins |
| [proto](https://github.com/orchestra-mcp/proto) | Source `.proto` definitions |

## License

[MIT](LICENSE)
