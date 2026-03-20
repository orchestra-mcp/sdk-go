package protocol

// MCPProtocolVersion is the MCP protocol version supported by this server.
const MCPProtocolVersion = "2025-06-18"

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
type MCPServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// MCPToolDefinition describes a single tool for the MCP tools/list response.
type MCPToolDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	InputSchema any    `json:"inputSchema"`
}

// MCPToolResult is the response to an MCP tools/call request.
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent is a single content block in a tool result.
type MCPContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// MCPResourcesCapability describes resource-related capabilities.
type MCPResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

// MCPResource describes a single resource for the MCP resources/list response.
type MCPResource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

// MCPResourceTemplate describes a URI template for resource discovery.
type MCPResourceTemplate struct {
	URITemplate string `json:"uriTemplate"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
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
type MCPPromptDefinition struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
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
