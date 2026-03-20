package plugin

// CatalogEntry describes a registered MCP tool with its source plugin metadata.
type CatalogEntry struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	PluginID    string   `json:"plugin_id"`
	Category    string   `json:"category"`
	Providers   []string `json:"providers,omitempty"`
	Schema      string   `json:"schema,omitempty"`
}

// ToolCatalog provides read-only access to the router's tool catalog.
// The in-process Router implements this interface.
type ToolCatalog interface {
	// ListCatalog returns paginated catalog entries, optionally filtered by plugin.
	ListCatalog(pluginFilter string, offset, limit int) []CatalogEntry

	// CatalogCount returns the total number of entries (optionally filtered).
	CatalogCount(pluginFilter string) int

	// SearchCatalog searches tool names and descriptions for the query.
	SearchCatalog(query string) []CatalogEntry

	// GetCatalogEntry returns a single tool's details, or nil if not found.
	GetCatalogEntry(toolName string) *CatalogEntry

	// ListPluginIDs returns sorted unique plugin IDs.
	ListPluginIDs() []string
}
