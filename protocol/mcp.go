package protocol

// MCPServerCapabilities describes the capabilities the MCP server supports.
type MCPServerCapabilities struct {
	Tools   *MCPToolsCapability   `json:"tools,omitempty"`
	Prompts *MCPPromptsCapability `json:"prompts,omitempty"`
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
