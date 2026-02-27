package plugin

// LifecycleHooks defines optional callbacks that a plugin can implement to
// respond to lifecycle events from the orchestrator.
type LifecycleHooks interface {
	// OnBoot is called when the orchestrator sends a Boot request with
	// configuration parameters. The plugin should initialize any resources
	// it needs and return an error if boot fails.
	OnBoot(config map[string]string) error

	// OnShutdown is called when the orchestrator requests a graceful shutdown.
	// The plugin should release resources and return an error if cleanup fails.
	OnShutdown() error
}

// noopLifecycle is the default implementation that does nothing.
type noopLifecycle struct{}

func (noopLifecycle) OnBoot(map[string]string) error { return nil }
func (noopLifecycle) OnShutdown() error              { return nil }
