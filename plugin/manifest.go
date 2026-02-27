package plugin

import (
	pluginv1 "github.com/orchestra-mcp/gen-go/orchestra/plugin/v1"
)

// ManifestBuilder provides a fluent API for constructing a PluginManifest.
type ManifestBuilder struct {
	manifest *pluginv1.PluginManifest
}

// NewManifest creates a new ManifestBuilder with the given plugin ID.
func NewManifest(id string) *ManifestBuilder {
	return &ManifestBuilder{
		manifest: &pluginv1.PluginManifest{
			Id:       id,
			Language: "go",
		},
	}
}

// Version sets the semantic version of the plugin.
func (b *ManifestBuilder) Version(v string) *ManifestBuilder {
	b.manifest.Version = v
	return b
}

// Description sets the human-readable description.
func (b *ManifestBuilder) Description(d string) *ManifestBuilder {
	b.manifest.Description = d
	return b
}

// Author sets the plugin author.
func (b *ManifestBuilder) Author(a string) *ManifestBuilder {
	b.manifest.Author = a
	return b
}

// Binary sets the path to the plugin binary.
func (b *ManifestBuilder) Binary(bin string) *ManifestBuilder {
	b.manifest.Binary = bin
	return b
}

// ProvidesTools declares the tool names this plugin provides.
func (b *ManifestBuilder) ProvidesTools(names ...string) *ManifestBuilder {
	b.manifest.ProvidesTools = append(b.manifest.ProvidesTools, names...)
	return b
}

// ProvidesStorage declares the storage types this plugin provides.
func (b *ManifestBuilder) ProvidesStorage(types ...string) *ManifestBuilder {
	b.manifest.ProvidesStorage = append(b.manifest.ProvidesStorage, types...)
	return b
}

// ProvidesTransport declares the transport types this plugin provides.
func (b *ManifestBuilder) ProvidesTransport(types ...string) *ManifestBuilder {
	b.manifest.ProvidesTransport = append(b.manifest.ProvidesTransport, types...)
	return b
}

// NeedsStorage declares the storage types this plugin requires from others.
func (b *ManifestBuilder) NeedsStorage(types ...string) *ManifestBuilder {
	b.manifest.NeedsStorage = append(b.manifest.NeedsStorage, types...)
	return b
}

// NeedsEvents declares the event types this plugin subscribes to.
func (b *ManifestBuilder) NeedsEvents(events ...string) *ManifestBuilder {
	b.manifest.NeedsEvents = append(b.manifest.NeedsEvents, events...)
	return b
}

// NeedsAI declares the AI providers this plugin requires.
func (b *ManifestBuilder) NeedsAI(providers ...string) *ManifestBuilder {
	b.manifest.NeedsAi = append(b.manifest.NeedsAi, providers...)
	return b
}

// NeedsTools declares the tools this plugin requires from other plugins.
func (b *ManifestBuilder) NeedsTools(tools ...string) *ManifestBuilder {
	b.manifest.NeedsTools = append(b.manifest.NeedsTools, tools...)
	return b
}

// Build returns the constructed PluginManifest.
func (b *ManifestBuilder) Build() *pluginv1.PluginManifest {
	return b.manifest
}
