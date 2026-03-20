package protocol

// MCPProtocolVersion is the MCP protocol version supported by this server.
const MCPProtocolVersion = "2025-11-25"

// MCPServerCapabilities describes the capabilities the MCP server supports.
type MCPServerCapabilities struct {
	Tools     *MCPToolsCapability     `json:"tools,omitempty"`
	Prompts   *MCPPromptsCapability   `json:"prompts,omitempty"`
	Logging   *MCPLoggingCapability   `json:"logging,omitempty"`
	Resources *MCPResourcesCapability `json:"resources,omitempty"`
}

// MCPToolsCapability describes tool-related capabilities.
type MCPToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPInitializeResult is the response to the MCP initialize method.
type MCPInitializeResult struct {
	ProtocolVersion string                `json:"protocolVersion"`
	Capabilities    MCPServerCapabilities `json:"capabilities"`
	ServerInfo      MCPServerInfo         `json:"serverInfo"`
	SessionID       string                `json:"_sessionId,omitempty"`
}

// MCPServerInfo identifies the MCP server.
// Extended in MCP 2025-11-25 with title, description, icons, and websiteUrl.
type MCPServerInfo struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Title       string    `json:"title,omitempty"`
	Description string    `json:"description,omitempty"`
	Icons       []MCPIcon `json:"icons,omitempty"`
	WebsiteURL  string    `json:"websiteUrl,omitempty"`
}

// MCPIcon represents an icon for a server, tool, resource, or prompt.
// Added in MCP 2025-11-25.
type MCPIcon struct {
	Src      string   `json:"src"`
	MimeType string   `json:"mimeType,omitempty"`
	Sizes    []string `json:"sizes,omitempty"`
}

// MCPToolAnnotations provides hints about a tool's behavior.
// Added in MCP 2025-11-25.
type MCPToolAnnotations struct {
	Title           string `json:"title,omitempty"`
	ReadOnlyHint    *bool  `json:"readOnlyHint,omitempty"`
	DestructiveHint *bool  `json:"destructiveHint,omitempty"`
	IdempotentHint  *bool  `json:"idempotentHint,omitempty"`
	OpenWorldHint   *bool  `json:"openWorldHint,omitempty"`
}

// MCPToolDefinition describes a single tool for the MCP tools/list response.
// Extended in MCP 2025-11-25 with title, icons, outputSchema, and annotations.
type MCPToolDefinition struct {
	Name         string              `json:"name"`
	Title        string              `json:"title,omitempty"`
	Description  string              `json:"description,omitempty"`
	Icons        []MCPIcon           `json:"icons,omitempty"`
	InputSchema  any                 `json:"inputSchema"`
	OutputSchema any                 `json:"outputSchema,omitempty"`
	Annotations  *MCPToolAnnotations `json:"annotations,omitempty"`
}

// MCPToolResult is the response to an MCP tools/call request.
type MCPToolResult struct {
	Content []MCPContent   `json:"content"`
	IsError bool           `json:"isError,omitempty"`
	Meta    map[string]any `json:"_meta,omitempty"` // Rich UI metadata (populated by WebGate for UI clients)
}

// MCPContent is a single content block in a tool result or prompt message.
// Extended in MCP 2025-11-25 with image, audio, and resource_link support.
type MCPContent struct {
	Type     string `json:"type"`               // "text", "image", "audio", "resource_link"
	Text     string `json:"text,omitempty"`     // for type "text"
	Data     string `json:"data,omitempty"`     // base64-encoded for type "image" or "audio"
	MimeType string `json:"mimeType,omitempty"` // for type "image" or "audio"
	URI      string `json:"uri,omitempty"`      // for type "resource_link"
}

// MCPResourcesCapability describes resource-related capabilities.
type MCPResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPResource describes a single resource for the MCP resources/list response.
// Extended in MCP 2025-11-25 with icons.
type MCPResource struct {
	URI         string    `json:"uri"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	MimeType    string    `json:"mimeType,omitempty"`
	Icons       []MCPIcon `json:"icons,omitempty"`
}

// MCPResourceTemplate describes a URI template for resource discovery.
// Extended in MCP 2025-11-25 with icons.
type MCPResourceTemplate struct {
	URITemplate string    `json:"uriTemplate"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	MimeType    string    `json:"mimeType,omitempty"`
	Icons       []MCPIcon `json:"icons,omitempty"`
}

// MCPResourceContent is a single content block returned by resources/read.
type MCPResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
}

// MCPLoggingCapability describes logging capabilities.
type MCPLoggingCapability struct{}

// MCPLogLevel represents MCP log severity levels (RFC 5424).
type MCPLogLevel string

const (
	LogLevelDebug     MCPLogLevel = "debug"
	LogLevelInfo      MCPLogLevel = "info"
	LogLevelNotice    MCPLogLevel = "notice"
	LogLevelWarning   MCPLogLevel = "warning"
	LogLevelError     MCPLogLevel = "error"
	LogLevelCritical  MCPLogLevel = "critical"
	LogLevelAlert     MCPLogLevel = "alert"
	LogLevelEmergency MCPLogLevel = "emergency"
)

// LogLevelSeverity returns the numeric severity for a log level (0=debug, 7=emergency).
// Returns -1 for unknown levels.
func LogLevelSeverity(level MCPLogLevel) int {
	switch level {
	case LogLevelDebug:
		return 0
	case LogLevelInfo:
		return 1
	case LogLevelNotice:
		return 2
	case LogLevelWarning:
		return 3
	case LogLevelError:
		return 4
	case LogLevelCritical:
		return 5
	case LogLevelAlert:
		return 6
	case LogLevelEmergency:
		return 7
	default:
		return -1
	}
}

// MCPPromptsCapability describes prompt-related capabilities.
type MCPPromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPPromptDefinition describes a single prompt for the MCP prompts/list response.
// Extended in MCP 2025-11-25 with icons.
type MCPPromptDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Icons       []MCPIcon           `json:"icons,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
}

// MCPPromptArgument describes a single argument for a prompt.
type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// MCPPromptMessage is a single message returned by prompts/get.
type MCPPromptMessage struct {
	Role    string     `json:"role"`
	Content MCPContent `json:"content"`
}

// MCPPromptResult is the response to an MCP prompts/get request.
type MCPPromptResult struct {
	Description string             `json:"description,omitempty"`
	Messages    []MCPPromptMessage `json:"messages"`
}
