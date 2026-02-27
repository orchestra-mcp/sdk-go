package plugin

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

// Plugin is the top-level runtime for a Go-based Orchestra plugin. It manages
// a QUIC server (to receive requests), a QUIC client (to connect to the
// orchestrator), the tool registry, and lifecycle hooks.
type Plugin struct {
	manifest          *pluginv1.PluginManifest
	server            *Server
	orchestratorAddr  string
	listenAddr        string
	certsDir          string
	lifecycle         LifecycleHooks
	orchestratorClient *OrchestratorClient
}

// PluginBuilder provides a fluent API for constructing a Plugin.
type PluginBuilder struct {
	manifestBuilder  *ManifestBuilder
	listenAddr       string
	orchestratorAddr string
	certsDir         string
	lifecycle        LifecycleHooks
	tools            []pendingTool
	prompts          []pendingPrompt
	storageHandler   StorageHandler
}

// pendingTool holds tool registration data until the plugin is built.
type pendingTool struct {
	name        string
	description string
	schema      *structpb.Struct
	handler     ToolHandler
}

// pendingPrompt holds prompt registration data until the plugin is built.
type pendingPrompt struct {
	name        string
	description string
	arguments   []*pluginv1.PromptArgument
	handler     PromptHandler
}

// New creates a new PluginBuilder with the given plugin ID.
func New(id string) *PluginBuilder {
	return &PluginBuilder{
		manifestBuilder: NewManifest(id),
		listenAddr:      "localhost:0",
		certsDir:        DefaultCertsDir,
		lifecycle:       noopLifecycle{},
	}
}

// Version sets the plugin version.
func (b *PluginBuilder) Version(v string) *PluginBuilder {
	b.manifestBuilder.Version(v)
	return b
}

// Description sets the plugin description.
func (b *PluginBuilder) Description(d string) *PluginBuilder {
	b.manifestBuilder.Description(d)
	return b
}

// Author sets the plugin author.
func (b *PluginBuilder) Author(a string) *PluginBuilder {
	b.manifestBuilder.Author(a)
	return b
}

// Binary sets the plugin binary path.
func (b *PluginBuilder) Binary(bin string) *PluginBuilder {
	b.manifestBuilder.Binary(bin)
	return b
}

// ListenAddr sets the address the plugin server listens on.
func (b *PluginBuilder) ListenAddr(addr string) *PluginBuilder {
	b.listenAddr = addr
	return b
}

// OrchestratorAddr sets the address of the orchestrator to connect to.
func (b *PluginBuilder) OrchestratorAddr(addr string) *PluginBuilder {
	b.orchestratorAddr = addr
	return b
}

// CertsDir sets the directory for mTLS certificates.
func (b *PluginBuilder) CertsDir(dir string) *PluginBuilder {
	b.certsDir = dir
	return b
}

// Lifecycle sets the lifecycle hooks implementation.
func (b *PluginBuilder) Lifecycle(hooks LifecycleHooks) *PluginBuilder {
	b.lifecycle = hooks
	return b
}

// ProvidesTools declares tools in the manifest.
func (b *PluginBuilder) ProvidesTools(names ...string) *PluginBuilder {
	b.manifestBuilder.ProvidesTools(names...)
	return b
}

// ProvidesStorage declares storage types in the manifest.
func (b *PluginBuilder) ProvidesStorage(types ...string) *PluginBuilder {
	b.manifestBuilder.ProvidesStorage(types...)
	return b
}

// ProvidesTransport declares transport types in the manifest.
func (b *PluginBuilder) ProvidesTransport(types ...string) *PluginBuilder {
	b.manifestBuilder.ProvidesTransport(types...)
	return b
}

// NeedsStorage declares storage dependencies in the manifest.
func (b *PluginBuilder) NeedsStorage(types ...string) *PluginBuilder {
	b.manifestBuilder.NeedsStorage(types...)
	return b
}

// NeedsEvents declares event dependencies in the manifest.
func (b *PluginBuilder) NeedsEvents(events ...string) *PluginBuilder {
	b.manifestBuilder.NeedsEvents(events...)
	return b
}

// NeedsAI declares AI provider dependencies in the manifest.
func (b *PluginBuilder) NeedsAI(providers ...string) *PluginBuilder {
	b.manifestBuilder.NeedsAI(providers...)
	return b
}

// NeedsTools declares tool dependencies in the manifest.
func (b *PluginBuilder) NeedsTools(tools ...string) *PluginBuilder {
	b.manifestBuilder.NeedsTools(tools...)
	return b
}

// RegisterTool adds a tool to the plugin. The tool name is also added to
// the manifest's ProvidesTools list.
func (b *PluginBuilder) RegisterTool(name string, description string, schema *structpb.Struct, handler ToolHandler) *PluginBuilder {
	b.tools = append(b.tools, pendingTool{
		name:        name,
		description: description,
		schema:      schema,
		handler:     handler,
	})
	b.manifestBuilder.ProvidesTools(name)
	return b
}

// RegisterPrompt adds a prompt to the plugin. The prompt name is also added to
// the manifest's ProvidesPrompts list.
func (b *PluginBuilder) RegisterPrompt(name string, description string, args []*pluginv1.PromptArgument, handler PromptHandler) *PluginBuilder {
	b.prompts = append(b.prompts, pendingPrompt{
		name:        name,
		description: description,
		arguments:   args,
		handler:     handler,
	})
	b.manifestBuilder.ProvidesPrompts(name)
	return b
}

// SetStorageHandler sets the storage handler for the plugin. This is used by
// storage plugins that handle StorageRead/Write/Delete/List requests.
func (b *PluginBuilder) SetStorageHandler(h StorageHandler) *PluginBuilder {
	b.storageHandler = h
	return b
}

// Build constructs the Plugin. Call Run to start it.
func (b *PluginBuilder) Build() *Plugin {
	return &Plugin{
		manifest:         b.manifestBuilder.Build(),
		listenAddr:       b.listenAddr,
		orchestratorAddr: b.orchestratorAddr,
		certsDir:         b.certsDir,
		lifecycle:        b.lifecycle,
	}
}

// BuildWithTools constructs the Plugin and registers all pending tools and
// handlers on its server. This is the recommended way to build a plugin.
func (b *PluginBuilder) BuildWithTools() *Plugin {
	p := b.Build()
	// Server will be created in Run, so we store tools on a temporary list.
	// Actually, we need to set up the server now so tools can be registered.
	// The server's TLS config will be set in Run.
	p.server = NewServer(p.listenAddr, nil)
	p.server.SetLifecycleHooks(p.lifecycle)
	for _, t := range b.tools {
		p.server.RegisterTool(t.name, t.description, t.schema, t.handler)
	}
	for _, pr := range b.prompts {
		p.server.RegisterPrompt(pr.name, pr.description, pr.arguments, pr.handler)
	}
	if b.storageHandler != nil {
		p.server.SetStorageHandler(b.storageHandler)
	}
	return p
}

// ParseFlags parses standard CLI flags and overrides builder values.
// Flags: --orchestrator-addr, --listen-addr, --certs-dir, --manifest
//
// If --manifest is passed, the plugin prints its manifest as JSON to stdout
// and exits immediately. This is used by `orchestra install` to query plugin
// metadata without starting the full QUIC stack.
func (p *Plugin) ParseFlags() {
	var printManifest bool
	flag.StringVar(&p.orchestratorAddr, "orchestrator-addr", p.orchestratorAddr, "Address of the orchestrator")
	flag.StringVar(&p.listenAddr, "listen-addr", p.listenAddr, "Address for the plugin server to listen on")
	flag.StringVar(&p.certsDir, "certs-dir", p.certsDir, "Directory for mTLS certificates")
	flag.BoolVar(&printManifest, "manifest", false, "Print plugin manifest as JSON and exit")
	flag.Parse()

	if printManifest {
		p.printManifestAndExit()
	}
}

// printManifestAndExit writes the plugin manifest as JSON to stdout and exits.
func (p *Plugin) printManifestAndExit() {
	m := p.manifest
	fmt.Fprintf(os.Stdout, `{"id":%q,"version":%q,"description":%q,"provides_tools":%s,"provides_storage":%s,"needs_storage":%s}`,
		m.Id,
		m.Version,
		m.Description,
		jsonStringArray(m.ProvidesTools),
		jsonStringArray(m.ProvidesStorage),
		jsonStringArray(m.NeedsStorage),
	)
	fmt.Fprintln(os.Stdout)
	os.Exit(0)
}

func jsonStringArray(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	parts := make([]string, len(arr))
	for i, s := range arr {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// Run starts the plugin:
//  1. Ensures mTLS certificates exist (auto-generates if needed).
//  2. Starts the QUIC server on the listen address.
//  3. Prints "READY <addr>" to stderr.
//  4. If an orchestrator address is configured, connects and sends Register.
//  5. Serves requests until the context is cancelled.
//  6. On shutdown, calls lifecycle OnShutdown and closes connections.
func (p *Plugin) Run(ctx context.Context) error {
	// Step 1: Ensure certs.
	serverTLS, err := ServerTLSConfig(p.certsDir, p.manifest.Id)
	if err != nil {
		return fmt.Errorf("server TLS config: %w", err)
	}

	// Step 2: Create and start QUIC server.
	if p.server == nil {
		p.server = NewServer(p.listenAddr, serverTLS)
		p.server.SetLifecycleHooks(p.lifecycle)
	} else {
		// Server was pre-created by BuildWithTools; set TLS config.
		p.server.tlsConfig = serverTLS
		p.server.addr = p.listenAddr
	}

	// Start server in background.
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- p.server.ListenAndServe(ctx)
	}()

	// Step 3: Wait for the server to bind and signal readiness with the actual address.
	actualAddr := p.server.ActualAddr()
	fmt.Fprintf(os.Stderr, "READY %s\n", actualAddr)

	// Step 4: Connect to orchestrator if configured.
	if p.orchestratorAddr != "" {
		clientTLS, err := ClientTLSConfig(p.certsDir, p.manifest.Id+"-client")
		if err != nil {
			return fmt.Errorf("client TLS config: %w", err)
		}

		client, err := NewOrchestratorClient(ctx, p.orchestratorAddr, clientTLS)
		if err != nil {
			return fmt.Errorf("connect to orchestrator: %w", err)
		}
		p.orchestratorClient = client

		// Send Register.
		resp, err := client.Send(ctx, &pluginv1.PluginRequest{
			RequestId: newRequestID(),
			Request: &pluginv1.PluginRequest_Register{
				Register: p.manifest,
			},
		})
		if err != nil {
			return fmt.Errorf("send register: %w", err)
		}
		if reg := resp.GetRegister(); reg != nil && !reg.Accepted {
			return fmt.Errorf("registration rejected: %s", reg.RejectReason)
		}

		log.Printf("registered with orchestrator at %s", p.orchestratorAddr)
	}

	// Step 5: Wait for context cancellation or server error.
	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil {
			return err
		}
	}

	// Step 6: Shutdown.
	if p.lifecycle != nil {
		if err := p.lifecycle.OnShutdown(); err != nil {
			log.Printf("shutdown hook error: %v", err)
		}
	}
	if p.orchestratorClient != nil {
		p.orchestratorClient.Close()
	}

	return nil
}

// Server returns the underlying QUIC server, useful for registering additional
// tools after Build.
func (p *Plugin) Server() *Server {
	return p.server
}

// Manifest returns the plugin manifest.
func (p *Plugin) Manifest() *pluginv1.PluginManifest {
	return p.manifest
}

// OrchestratorClient returns the orchestrator QUIC client. This is nil until
// Run has connected to the orchestrator. Tool handlers that need storage access
// should call this lazily (e.g., via a closure) rather than at registration time.
func (p *Plugin) OrchestratorClient() *OrchestratorClient {
	return p.orchestratorClient
}
