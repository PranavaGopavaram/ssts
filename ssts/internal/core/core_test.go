package core

import (
	"testing"

	"go.uber.org/zap"

	"github.com/pranavgopavaram/ssts/internal/config"
	"github.com/pranavgopavaram/ssts/internal/database"
	"github.com/pranavgopavaram/ssts/internal/plugins"
)

func TestOrchestratorCreation(t *testing.T) {
	// Create minimal config for testing
	cfg := &config.Config{
		InfluxDB: config.InfluxDBConfig{
			URL:    "http://localhost:8086",
			Token:  "test-token",
			Org:    "test-org",
			Bucket: "test-bucket",
		},
		Safety:  config.SafetyConfig{},
		Metrics: config.MetricsConfig{},
	}

	// Create logger
	logger, _ := zap.NewDevelopment()

	// Create database (nil for test)
	var db *database.Database

	// Create plugin manager
	pluginMgr := plugins.NewPluginManager()

	// Create orchestrator - this should work without errors
	orchestrator := NewOrchestrator(cfg, db, pluginMgr, logger)

	if orchestrator == nil {
		t.Fatal("Expected orchestrator to be created, got nil")
	}

	if orchestrator.config != cfg {
		t.Error("Expected orchestrator config to match input config")
	}

	if orchestrator.logger != logger {
		t.Error("Expected orchestrator logger to match input logger")
	}
}
