package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/pranavgopavaram/ssts/internal/config"
	"github.com/pranavgopavaram/ssts/internal/database"
	"github.com/pranavgopavaram/ssts/internal/metrics"
	"github.com/pranavgopavaram/ssts/internal/plugins"
	"github.com/pranavgopavaram/ssts/internal/safety"
	"github.com/pranavgopavaram/ssts/pkg/models"
)

// Orchestrator manages the overall test execution and coordination
type Orchestrator struct {
	config           *config.Config
	db               *database.Database
	influxDB         *database.InfluxDB
	pluginManager    *plugins.PluginManager
	safetyMonitor    *safety.Monitor
	metricsCollector *metrics.Collector
	testOrchestrator *TestOrchestrator
	logger           *zap.Logger
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(cfg *config.Config, db *database.Database, pluginMgr *plugins.PluginManager, logger *zap.Logger) *Orchestrator {
	// Initialize InfluxDB
	influxDB := database.NewInfluxDB(cfg.InfluxDB)

	// Create logrus logger from zap logger
	logrusLogger := logrus.New()

	// Initialize system monitor
	systemMonitor := safety.NewSystemMonitor()

	// Initialize alert manager
	alertManager := safety.NewAlertManager(logrusLogger)

	// Convert safety config to safety.Config
	safetyConfig := safety.Config{
		CheckInterval:       1 * time.Second,
		AlertThreshold:      85.0,
		EmergencyThreshold:  95.0,
		AutoStopEnabled:     true,
		RampUpEnabled:       true,
		RampUpDuration:      30 * time.Second,
		RampUpSteps:         10,
		CooldownPeriod:      60 * time.Second,
		MaxViolationsPerMin: 5,
	}

	// Initialize safety monitor with correct arguments
	safetyMonitor := safety.NewMonitor(systemMonitor, alertManager, safetyConfig, logrusLogger)

	// Initialize metrics collector with correct arguments
	metricsCollector := metrics.NewCollector(logger)

	// Initialize test orchestrator with correct arguments
	testOrchestrator := NewTestOrchestrator(pluginMgr, safetyMonitor, metricsCollector, logrusLogger)

	return &Orchestrator{
		config:           cfg,
		db:               db,
		influxDB:         influxDB,
		pluginManager:    pluginMgr,
		safetyMonitor:    safetyMonitor,
		metricsCollector: metricsCollector,
		testOrchestrator: testOrchestrator,
		logger:           logger,
	}
}

// ExecuteTestFromFile executes a test from a configuration file
func (o *Orchestrator) ExecuteTestFromFile(ctx context.Context, configPath string) (*models.TestResult, error) {
	// Load test configuration from file
	configData, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var testConfig models.TestConfiguration

	// Try YAML first, then JSON
	if err := yaml.Unmarshal(configData, &testConfig); err != nil {
		if err := json.Unmarshal(configData, &testConfig); err != nil {
			return nil, fmt.Errorf("failed to parse config file (tried YAML and JSON): %w", err)
		}
	}

	// Set default values if not specified
	if testConfig.Duration == 0 {
		testConfig.Duration = 60 * time.Second
	}
	if testConfig.Safety.MaxCPUPercent == 0 {
		testConfig.Safety = models.DefaultSafetyLimits()
	}

	// Create test parameters
	params := models.TestParams{
		Duration:    testConfig.Duration,
		Intensity:   70, // Default intensity
		Concurrency: 1,  // Default concurrency
	}

	// Parse custom parameters from config
	if len(testConfig.Config) > 0 {
		var customParams map[string]interface{}
		if err := json.Unmarshal(testConfig.Config, &customParams); err == nil {
			params.CustomParams = customParams

			// Extract common parameters
			if intensity, ok := customParams["intensity"].(float64); ok {
				params.Intensity = int(intensity)
			}
			if concurrency, ok := customParams["concurrency"].(float64); ok {
				params.Concurrency = int(concurrency)
			}
		}
	}

	// Start test execution
	executionID, err := o.testOrchestrator.StartTest(testConfig, params)
	if err != nil {
		return nil, fmt.Errorf("failed to start test: %w", err)
	}

	o.logger.Info("Test execution started",
		zap.String("execution_id", executionID),
		zap.String("plugin", testConfig.Plugin),
		zap.Duration("duration", params.Duration),
	)

	// Wait for test completion
	return o.waitForTestCompletion(ctx, executionID, params.Duration)
}

// waitForTestCompletion waits for a test to complete and returns the result
func (o *Orchestrator) waitForTestCompletion(ctx context.Context, executionID string, maxDuration time.Duration) (*models.TestResult, error) {
	// Create a timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, maxDuration+30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			// Emergency stop the test
			o.testOrchestrator.EmergencyStop(executionID, "Test execution timeout")
			return nil, fmt.Errorf("test execution timeout")

		case <-ticker.C:
			execution, err := o.testOrchestrator.GetTestStatus(executionID)
			if err != nil {
				return nil, fmt.Errorf("failed to get test status: %w", err)
			}

			// Check if test is complete
			if execution.Status == models.StatusCompleted ||
				execution.Status == models.StatusFailed ||
				execution.Status == models.StatusStopped {

				// Get test metrics
				metrics, err := o.testOrchestrator.GetTestMetrics(executionID)
				if err != nil {
					o.logger.Warn("Failed to get test metrics", zap.Error(err))
					metrics = []models.MetricPoint{}
				}

				// Calculate test score and determine if passed
				score := o.calculateTestScore(execution, metrics)
				passed := execution.Status == models.StatusCompleted && score >= 70.0

				result := &models.TestResult{
					TestID:   execution.TestID,
					Status:   execution.Status,
					Duration: execution.Duration,
					Metrics:  metrics,
					Score:    score,
					Passed:   passed,
				}

				if execution.ErrorMessage != nil {
					result.Errors = []string{*execution.ErrorMessage}
				}

				o.logger.Info("Test execution completed",
					zap.String("execution_id", executionID),
					zap.String("status", string(execution.Status)),
					zap.Float64("score", score),
					zap.Bool("passed", passed),
				)

				return result, nil
			}
		}
	}
}

// calculateTestScore calculates a test score based on execution and metrics
func (o *Orchestrator) calculateTestScore(execution *models.TestExecution, metrics []models.MetricPoint) float64 {
	baseScore := 100.0

	// Deduct points for failures
	if execution.Status == models.StatusFailed {
		baseScore -= 50.0
	} else if execution.Status == models.StatusStopped {
		baseScore -= 25.0
	}

	// Analyze metrics for performance scoring
	if len(metrics) == 0 {
		return baseScore * 0.5 // No metrics available
	}

	// Simple scoring based on metric availability and values
	// In a real implementation, this would be more sophisticated
	performanceScore := 1.0
	for _, metric := range metrics {
		if cpuUsage, ok := metric.Fields["usage_percent"].(float64); ok {
			if cpuUsage > 95.0 {
				performanceScore *= 0.9 // Deduct for very high CPU usage
			}
		}
	}

	return baseScore * performanceScore
}

// StartTest starts a new test execution
func (o *Orchestrator) StartTest(config models.TestConfiguration, params models.TestParams) (string, error) {
	return o.testOrchestrator.StartTest(config, params)
}

// StopTest stops a running test
func (o *Orchestrator) StopTest(executionID string) error {
	return o.testOrchestrator.StopTest(executionID)
}

// GetTestStatus returns the status of a test execution
func (o *Orchestrator) GetTestStatus(executionID string) (*models.TestExecution, error) {
	return o.testOrchestrator.GetTestStatus(executionID)
}

// ListExecutions returns all test executions
func (o *Orchestrator) ListExecutions() []models.TestExecution {
	return o.testOrchestrator.ListExecutions()
}

// GetTestMetrics returns metrics for a test execution
func (o *Orchestrator) GetTestMetrics(executionID string) ([]models.MetricPoint, error) {
	return o.testOrchestrator.GetTestMetrics(executionID)
}

// GetPluginManager returns the plugin manager
func (o *Orchestrator) GetPluginManager() *plugins.PluginManager {
	return o.pluginManager
}

// GetSystemHealth returns overall system health
func (o *Orchestrator) GetSystemHealth() map[string]interface{} {
	health := map[string]interface{}{
		"status":     "healthy",
		"timestamp":  time.Now(),
		"components": make(map[string]interface{}),
	}

	// Check database health
	if err := o.db.HealthCheck(); err != nil {
		health["components"].(map[string]interface{})["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		health["components"].(map[string]interface{})["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check InfluxDB health
	if err := o.influxDB.HealthCheck(context.Background()); err != nil {
		health["components"].(map[string]interface{})["influxdb"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "degraded"
	} else {
		health["components"].(map[string]interface{})["influxdb"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check plugins health
	pluginHealth := make(map[string]interface{})
	for _, plugin := range o.pluginManager.ListPlugins() {
		if err := plugin.HealthCheck(); err != nil {
			pluginHealth[plugin.Name()] = map[string]interface{}{
				"status": "unhealthy",
				"error":  err.Error(),
			}
		} else {
			pluginHealth[plugin.Name()] = map[string]interface{}{
				"status": "healthy",
			}
		}
	}
	health["components"].(map[string]interface{})["plugins"] = pluginHealth

	return health
}

// Cleanup performs cleanup operations
func (o *Orchestrator) Cleanup() error {
	o.logger.Info("Starting orchestrator cleanup")

	// Cleanup metrics collector
	if o.metricsCollector != nil {
		o.metricsCollector.Stop()
	}

	// Close InfluxDB
	if o.influxDB != nil {
		o.influxDB.Close()
	}

	// Close database
	if o.db != nil {
		if err := o.db.Close(); err != nil {
			o.logger.Error("Failed to close database", zap.Error(err))
			return err
		}
	}

	o.logger.Info("Orchestrator cleanup completed")
	return nil
}
