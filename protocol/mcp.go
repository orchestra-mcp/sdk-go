package protocol

// MCPServerCapabilities describes the capabilities the MCP server supports.
type MCPServerCapabilities struct {
	Tools *MCPToolsCapability `json:"tools,omitempty"`
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
