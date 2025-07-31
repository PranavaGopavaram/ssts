package plugins

import (
	"context"

	"github.com/pranavgopavaram/ssts/pkg/models"
)

// StressPlugin defines the interface that all stress test plugins must implement
type StressPlugin interface {
	// Plugin metadata
	Name() string
	Version() string
	Description() string

	// Configuration schema
	ConfigSchema() []byte

	// Test lifecycle
	Initialize(config interface{}) error
	Execute(ctx context.Context, params models.TestParams) error
	Cleanup() error

	// Metrics
	GetMetrics() map[string]interface{}

	// Safety checks
	GetSafetyLimits() models.SafetyLimits

	// Health check
	HealthCheck() error
}

// PluginManager manages the loading and execution of plugins
type PluginManager struct {
	plugins map[string]StressPlugin
}

// NewPluginManager creates a new plugin manager
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins: make(map[string]StressPlugin),
	}
}

// RegisterPlugin registers a plugin with the manager
func (pm *PluginManager) RegisterPlugin(plugin StressPlugin) error {
	pm.plugins[plugin.Name()] = plugin
	return nil
}

// GetPlugin retrieves a plugin by name
func (pm *PluginManager) GetPlugin(name string) (StressPlugin, bool) {
	plugin, exists := pm.plugins[name]
	return plugin, exists
}

// ListPlugins returns all registered plugins
func (pm *PluginManager) ListPlugins() []StressPlugin {
	plugins := make([]StressPlugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		plugins = append(plugins, plugin)
	}
	return plugins
}

// ExecutePlugin executes a plugin with given parameters
func (pm *PluginManager) ExecutePlugin(ctx context.Context, name string, config interface{}, params models.TestParams) error {
	plugin, exists := pm.GetPlugin(name)
	if !exists {
		return ErrPluginNotFound
	}

	if err := plugin.Initialize(config); err != nil {
		return err
	}

	defer plugin.Cleanup()

	return plugin.Execute(ctx, params)
}