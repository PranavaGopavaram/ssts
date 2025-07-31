package core

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pranavgopavaram/ssts/internal/plugins"
	"github.com/pranavgopavaram/ssts/internal/safety"
	"github.com/pranavgopavaram/ssts/pkg/models"
	"github.com/sirupsen/logrus"
)

// TestOrchestrator manages test execution lifecycle
type TestOrchestrator struct {
	pluginManager   *plugins.PluginManager
	safetyMonitor   *safety.Monitor
	metricsCollector MetricsCollector
	executions      map[string]*TestExecution
	mu              sync.RWMutex
	logger          *logrus.Logger
}

// TestExecution represents an active test execution
type TestExecution struct {
	ID           string
	Config       models.TestConfiguration
	Status       models.ExecutionStatus
	StartTime    time.Time
	EndTime      *time.Time
	Context      context.Context
	Cancel       context.CancelFunc
	Metrics      []models.MetricPoint
	ErrorMessage *string
	mu           sync.RWMutex
}

// MetricsCollector interface for collecting metrics
type MetricsCollector interface {
	CollectSystemMetrics() models.SystemMetrics
	CollectPluginMetrics(pluginName string, plugin plugins.StressPlugin) map[string]interface{}
	StartCollection(ctx context.Context, testID string)
	StopCollection(testID string)
}

// NewTestOrchestrator creates a new test orchestrator
func NewTestOrchestrator(
	pluginManager *plugins.PluginManager,
	safetyMonitor *safety.Monitor,
	metricsCollector MetricsCollector,
	logger *logrus.Logger,
) *TestOrchestrator {
	return &TestOrchestrator{
		pluginManager:    pluginManager,
		safetyMonitor:    safetyMonitor,
		metricsCollector: metricsCollector,
		executions:       make(map[string]*TestExecution),
		logger:           logger,
	}
}

// StartTest starts a new test execution
func (to *TestOrchestrator) StartTest(config models.TestConfiguration, params models.TestParams) (string, error) {
	// Validate plugin exists
	plugin, exists := to.pluginManager.GetPlugin(config.Plugin)
	if !exists {
		return "", fmt.Errorf("plugin not found: %s", config.Plugin)
	}

	// Create execution ID
	executionID := uuid.New().String()

	// Create execution context
	ctx, cancel := context.WithTimeout(context.Background(), params.Duration)

	// Create test execution
	execution := &TestExecution{
		ID:        executionID,
		Config:    config,
		Status:    models.StatusPending,
		StartTime: time.Now(),
		Context:   ctx,
		Cancel:    cancel,
		Metrics:   make([]models.MetricPoint, 0),
	}

	// Store execution
	to.mu.Lock()
	to.executions[executionID] = execution
	to.mu.Unlock()

	// Start test in goroutine
	go to.executeTest(execution, plugin, params)

	to.logger.WithFields(logrus.Fields{
		"execution_id": executionID,
		"plugin":       config.Plugin,
		"duration":     params.Duration,
	}).Info("Test execution started")

	return executionID, nil
}

// executeTest executes a test
func (to *TestOrchestrator) executeTest(execution *TestExecution, plugin plugins.StressPlugin, params models.TestParams) {
	defer func() {
		if r := recover(); r != nil {
			to.handleTestPanic(execution, r)
		}
	}()

	// Update status to running
	execution.mu.Lock()
	execution.Status = models.StatusRunning
	execution.mu.Unlock()

	// Start safety monitoring
	safetyCtx, safetyCancel := context.WithCancel(execution.Context)
	defer safetyCancel()

	go to.monitorSafety(safetyCtx, execution, plugin.GetSafetyLimits())

	// Start metrics collection
	to.metricsCollector.StartCollection(execution.Context, execution.ID)
	defer to.metricsCollector.StopCollection(execution.ID)

	// Parse plugin configuration
	var pluginConfig interface{}
	if len(execution.Config.Config) > 0 {
		if err := json.Unmarshal(execution.Config.Config, &pluginConfig); err != nil {
			to.finishTestWithError(execution, fmt.Errorf("failed to parse plugin config: %w", err))
			return
		}
	}

	// Execute the test
	err := to.pluginManager.ExecutePlugin(execution.Context, execution.Config.Plugin, pluginConfig, params)
	
	if err != nil {
		if execution.Context.Err() == context.Canceled {
			to.finishTestWithStatus(execution, models.StatusStopped)
		} else {
			to.finishTestWithError(execution, err)
		}
		return
	}

	// Test completed successfully
	to.finishTestWithStatus(execution, models.StatusCompleted)
}

// monitorSafety monitors system safety during test execution
func (to *TestOrchestrator) monitorSafety(ctx context.Context, execution *TestExecution, safetyLimits models.SafetyLimits) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if violation := to.safetyMonitor.CheckSafetyLimits(safetyLimits); violation != nil {
				to.logger.WithFields(logrus.Fields{
					"execution_id": execution.ID,
					"violation":    violation.Type,
					"value":        violation.CurrentValue,
					"limit":        violation.Limit,
				}).Warn("Safety limit violation detected")

				// Emergency stop if critical
				if violation.Critical {
					to.EmergencyStop(execution.ID, fmt.Sprintf("Critical safety violation: %s", violation.Message))
					return
				}
			}
		}
	}
}

// StopTest stops a running test
func (to *TestOrchestrator) StopTest(executionID string) error {
	to.mu.RLock()
	execution, exists := to.executions[executionID]
	to.mu.RUnlock()

	if !exists {
		return fmt.Errorf("test execution not found: %s", executionID)
	}

	execution.mu.Lock()
	if execution.Status != models.StatusRunning {
		execution.mu.Unlock()
		return fmt.Errorf("test is not running: %s", execution.Status)
	}
	execution.mu.Unlock()

	// Cancel the test
	execution.Cancel()

	to.logger.WithField("execution_id", executionID).Info("Test execution stopped")
	return nil
}

// EmergencyStop performs an emergency stop of a test
func (to *TestOrchestrator) EmergencyStop(executionID string, reason string) error {
	to.mu.RLock()
	execution, exists := to.executions[executionID]
	to.mu.RUnlock()

	if !exists {
		return fmt.Errorf("test execution not found: %s", executionID)
	}

	// Cancel the test immediately
	execution.Cancel()

	// Update status and error message
	execution.mu.Lock()
	execution.Status = models.StatusFailed
	execution.ErrorMessage = &reason
	now := time.Now()
	execution.EndTime = &now
	execution.mu.Unlock()

	to.logger.WithFields(logrus.Fields{
		"execution_id": executionID,
		"reason":       reason,
	}).Error("Emergency stop executed")

	return nil
}

// GetTestStatus returns the status of a test execution
func (to *TestOrchestrator) GetTestStatus(executionID string) (*models.TestExecution, error) {
	to.mu.RLock()
	execution, exists := to.executions[executionID]
	to.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("test execution not found: %s", executionID)
	}

	execution.mu.RLock()
	defer execution.mu.RUnlock()

	// Convert to model
	result := &models.TestExecution{
		ID:           execution.ID,
		TestID:       execution.Config.ID,
		Status:       execution.Status,
		StartTime:    &execution.StartTime,
		EndTime:      execution.EndTime,
		ErrorMessage: execution.ErrorMessage,
	}

	if execution.EndTime != nil {
		duration := execution.EndTime.Sub(execution.StartTime)
		result.Duration = duration
	}

	return result, nil
}

// ListExecutions returns all test executions
func (to *TestOrchestrator) ListExecutions() []models.TestExecution {
	to.mu.RLock()
	defer to.mu.RUnlock()

	executions := make([]models.TestExecution, 0, len(to.executions))
	for _, execution := range to.executions {
		execution.mu.RLock()
		
		modelExec := models.TestExecution{
			ID:           execution.ID,
			TestID:       execution.Config.ID,
			Status:       execution.Status,
			StartTime:    &execution.StartTime,
			EndTime:      execution.EndTime,
			ErrorMessage: execution.ErrorMessage,
		}

		if execution.EndTime != nil {
			duration := execution.EndTime.Sub(execution.StartTime)
			modelExec.Duration = duration
		}

		executions = append(executions, modelExec)
		execution.mu.RUnlock()
	}

	return executions
}

// GetTestMetrics returns metrics for a test execution
func (to *TestOrchestrator) GetTestMetrics(executionID string) ([]models.MetricPoint, error) {
	to.mu.RLock()
	execution, exists := to.executions[executionID]
	to.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("test execution not found: %s", executionID)
	}

	execution.mu.RLock()
	defer execution.mu.RUnlock()

	// Return copy of metrics
	metrics := make([]models.MetricPoint, len(execution.Metrics))
	copy(metrics, execution.Metrics)
	
	return metrics, nil
}

// CleanupCompletedTests removes completed test executions older than specified duration
func (to *TestOrchestrator) CleanupCompletedTests(maxAge time.Duration) int {
	to.mu.Lock()
	defer to.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for id, execution := range to.executions {
		execution.mu.RLock()
		shouldClean := execution.Status != models.StatusRunning && 
			execution.Status != models.StatusPending &&
			execution.EndTime != nil &&
			execution.EndTime.Before(cutoff)
		execution.mu.RUnlock()

		if shouldClean {
			delete(to.executions, id)
			cleaned++
		}
	}

	to.logger.WithField("cleaned_count", cleaned).Info("Cleaned up completed test executions")
	return cleaned
}

// finishTestWithError finishes a test with an error
func (to *TestOrchestrator) finishTestWithError(execution *TestExecution, err error) {
	execution.mu.Lock()
	execution.Status = models.StatusFailed
	errorMsg := err.Error()
	execution.ErrorMessage = &errorMsg
	now := time.Now()
	execution.EndTime = &now
	execution.mu.Unlock()

	to.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"error":        err.Error(),
	}).Error("Test execution failed")
}

// finishTestWithStatus finishes a test with a specific status
func (to *TestOrchestrator) finishTestWithStatus(execution *TestExecution, status models.ExecutionStatus) {
	execution.mu.Lock()
	execution.Status = status
	now := time.Now()
	execution.EndTime = &now
	execution.mu.Unlock()

	to.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"status":       status,
		"duration":     now.Sub(execution.StartTime),
	}).Info("Test execution finished")
}

// handleTestPanic handles panics during test execution
func (to *TestOrchestrator) handleTestPanic(execution *TestExecution, r interface{}) {
	errorMsg := fmt.Sprintf("Test panicked: %v", r)
	
	execution.mu.Lock()
	execution.Status = models.StatusFailed
	execution.ErrorMessage = &errorMsg
	now := time.Now()
	execution.EndTime = &now
	execution.mu.Unlock()

	to.logger.WithFields(logrus.Fields{
		"execution_id": execution.ID,
		"panic":        r,
	}).Error("Test execution panicked")
}

// AddMetric adds a metric point to a test execution
func (to *TestOrchestrator) AddMetric(executionID string, metric models.MetricPoint) error {
	to.mu.RLock()
	execution, exists := to.executions[executionID]
	to.mu.RUnlock()

	if !exists {
		return fmt.Errorf("test execution not found: %s", executionID)
	}

	execution.mu.Lock()
	execution.Metrics = append(execution.Metrics, metric)
	execution.mu.Unlock()

	return nil
}