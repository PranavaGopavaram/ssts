package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/pranavgopavaram/ssts/internal/database"
	"github.com/pranavgopavaram/ssts/pkg/models"
)

// Additional API handlers

// @Summary Update test configuration
// @Description Update an existing test configuration
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Param test body models.TestConfiguration true "Updated test configuration"
// @Success 200 {object} models.TestConfiguration
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id} [put]
func (s *Server) updateTest(c *gin.Context) {
	id := c.Param("id")

	var test models.TestConfiguration
	if err := c.ShouldBindJSON(&test); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Ensure ID matches
	test.ID = id
	test.Updated = time.Now()

	repo := database.NewRepository(s.db)
	if err := repo.UpdateTestConfiguration(&test); err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Test not found"})
		} else {
			s.logger.Error("Failed to update test", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to update test"})
		}
		return
	}

	c.JSON(http.StatusOK, test)
}

// @Summary Delete test configuration
// @Description Delete a test configuration
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Success 204 "No Content"
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id} [delete]
func (s *Server) deleteTest(c *gin.Context) {
	id := c.Param("id")

	repo := database.NewRepository(s.db)
	if err := repo.DeleteTestConfiguration(id); err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Test not found"})
		} else {
			s.logger.Error("Failed to delete test", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete test"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// @Summary Stop test execution
// @Description Stop a running test
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id}/stop [post]
func (s *Server) stopTest(c *gin.Context) {
	id := c.Param("id")

	// Find running execution for this test
	executions := s.orchestrator.ListExecutions()
	var executionID string
	for _, exec := range executions {
		if exec.TestID == id && exec.Status == models.StatusRunning {
			executionID = exec.ID
			break
		}
	}

	if executionID == "" {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "No running execution found for this test"})
		return
	}

	if err := s.orchestrator.StopTest(executionID); err != nil {
		s.logger.Error("Failed to stop test", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to stop test"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message":      "Test stopped successfully",
		"execution_id": executionID,
	})
}

// @Summary Get test status
// @Description Get the current status of a test
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Success 200 {object} models.TestExecution
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id}/status [get]
func (s *Server) getTestStatus(c *gin.Context) {
	id := c.Param("id")

	// Find the latest execution for this test
	executions := s.orchestrator.ListExecutions()
	var latestExecution *models.TestExecution
	for _, exec := range executions {
		if exec.TestID == id {
			if latestExecution == nil || exec.StartTime.After(*latestExecution.StartTime) {
				latestExecution = &exec
			}
		}
	}

	if latestExecution == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "No execution found for this test"})
		return
	}

	c.JSON(http.StatusOK, latestExecution)
}

// @Summary Get test results
// @Description Get aggregated results for a test
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Success 200 {object} models.TestResult
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id}/results [get]
func (s *Server) getTestResults(c *gin.Context) {
	id := c.Param("id")

	// Find completed executions for this test
	executions := s.orchestrator.ListExecutions()
	var completedExecutions []models.TestExecution
	for _, exec := range executions {
		if exec.TestID == id && exec.Status == models.StatusCompleted {
			completedExecutions = append(completedExecutions, exec)
		}
	}

	if len(completedExecutions) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "No completed executions found for this test"})
		return
	}

	// Get the latest completed execution
	latestExecution := completedExecutions[0]
	for _, exec := range completedExecutions {
		if exec.StartTime.After(*latestExecution.StartTime) {
			latestExecution = exec
		}
	}

	// Build test result
	result := models.TestResult{
		TestID:   id,
		Status:   latestExecution.Status,
		Duration: latestExecution.Duration,
		Passed:   latestExecution.Status == models.StatusCompleted,
		Score:    calculateTestScore(latestExecution),
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Get test metrics
// @Description Get metrics for a specific test
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Param start query string false "Start time (RFC3339)"
// @Param end query string false "End time (RFC3339)"
// @Success 200 {array} models.MetricPoint
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id}/metrics [get]
func (s *Server) getTestMetrics(c *gin.Context) {
	id := c.Param("id")

	// Parse time range
	timeRange := models.TimeRange{
		Start: time.Now().Add(-1 * time.Hour),
		End:   time.Now(),
	}

	if startStr := c.Query("start"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			timeRange.Start = t
		}
	}

	if endStr := c.Query("end"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			timeRange.End = t
		}
	}

	// Query metrics from InfluxDB
	metrics, err := s.influxDB.QueryMetrics(context.Background(), id, "system_cpu", timeRange)
	if err != nil {
		s.logger.Error("Failed to query metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to query metrics"})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// @Summary Export test data
// @Description Export test data in various formats
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Param request body models.ExportRequest true "Export request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id}/export [post]
func (s *Server) exportTestData(c *gin.Context) {
	id := c.Param("id")

	var request models.ExportRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	request.TestID = id

	// TODO: Implement data export functionality
	// This would include:
	// - Query metrics from InfluxDB
	// - Generate reports in requested format (JSON, CSV, PDF)
	// - Return download link or data directly

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Export functionality not yet implemented",
		"request": request,
	})
}

// Execution handlers

// @Summary List test executions
// @Description Get a list of test executions
// @Tags executions
// @Accept json
// @Produce json
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Param status query string false "Filter by status"
// @Success 200 {array} models.TestExecution
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/executions [get]
func (s *Server) listExecutions(c *gin.Context) {
	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)
	status := c.Query("status")

	repo := database.NewRepository(s.db)
	var executions []models.TestExecution
	var err error

	if status != "" {
		executions, err = repo.ListTestExecutionsByStatus(models.ExecutionStatus(status), limit, offset)
	} else {
		executions, err = repo.ListTestExecutions(limit, offset)
	}

	if err != nil {
		s.logger.Error("Failed to list executions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to list executions"})
		return
	}

	c.JSON(http.StatusOK, executions)
}

// @Summary Get test execution
// @Description Get a specific test execution by ID
// @Tags executions
// @Accept json
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {object} models.TestExecution
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/executions/{id} [get]
func (s *Server) getExecution(c *gin.Context) {
	id := c.Param("id")

	execution, err := s.orchestrator.GetTestStatus(id)
	if err != nil {
		if err.Error() == "test execution not found: "+id {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Execution not found"})
		} else {
			s.logger.Error("Failed to get execution", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get execution"})
		}
		return
	}

	c.JSON(http.StatusOK, execution)
}

// @Summary Stop test execution
// @Description Stop a running test execution
// @Tags executions
// @Accept json
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/executions/{id}/stop [post]
func (s *Server) stopExecution(c *gin.Context) {
	id := c.Param("id")

	if err := s.orchestrator.StopTest(id); err != nil {
		if err.Error() == "test execution not found: "+id {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Execution not found"})
		} else {
			s.logger.Error("Failed to stop execution", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to stop execution"})
		}
		return
	}

	c.JSON(http.StatusOK, map[string]string{
		"message": "Execution stopped successfully",
	})
}

// @Summary Get execution metrics
// @Description Get metrics for a specific execution
// @Tags executions
// @Accept json
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {array} models.MetricPoint
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/executions/{id}/metrics [get]
func (s *Server) getExecutionMetrics(c *gin.Context) {
	id := c.Param("id")

	metrics, err := s.orchestrator.GetTestMetrics(id)
	if err != nil {
		if err.Error() == "test execution not found: "+id {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Execution not found"})
		} else {
			s.logger.Error("Failed to get execution metrics", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get execution metrics"})
		}
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// @Summary Get execution logs
// @Description Get logs for a specific execution
// @Tags executions
// @Accept json
// @Produce json
// @Param id path string true "Execution ID"
// @Success 200 {array} string
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/executions/{id}/logs [get]
func (s *Server) getExecutionLogs(c *gin.Context) {
	id := c.Param("id")

	// TODO: Implement log retrieval
	// This would involve querying logs from a log storage system
	
	c.JSON(http.StatusOK, []string{
		"Log retrieval not yet implemented",
		"Execution ID: " + id,
	})
}

// Plugin handlers

// @Summary List plugins
// @Description Get a list of available plugins
// @Tags plugins
// @Accept json
// @Produce json
// @Success 200 {array} models.Plugin
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/plugins [get]
func (s *Server) listPlugins(c *gin.Context) {
	// Get plugins from plugin manager
	plugins := s.orchestrator.GetPluginManager().ListPlugins()
	
	// Convert to response format
	pluginList := make([]map[string]interface{}, 0, len(plugins))
	for _, plugin := range plugins {
		pluginInfo := map[string]interface{}{
			"name":         plugin.Name(),
			"version":      plugin.Version(),
			"description":  plugin.Description(),
			"safety_limits": plugin.GetSafetyLimits(),
		}
		pluginList = append(pluginList, pluginInfo)
	}

	c.JSON(http.StatusOK, pluginList)
}

// @Summary Get plugin details
// @Description Get details for a specific plugin
// @Tags plugins
// @Accept json
// @Produce json
// @Param name path string true "Plugin name"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/plugins/{name} [get]
func (s *Server) getPlugin(c *gin.Context) {
	name := c.Param("name")

	plugin, exists := s.orchestrator.GetPluginManager().GetPlugin(name)
	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Plugin not found"})
		return
	}

	pluginInfo := map[string]interface{}{
		"name":         plugin.Name(),
		"version":      plugin.Version(),
		"description":  plugin.Description(),
		"safety_limits": plugin.GetSafetyLimits(),
	}

	c.JSON(http.StatusOK, pluginInfo)
}

// @Summary Get plugin configuration schema
// @Description Get the JSON schema for plugin configuration
// @Tags plugins
// @Accept json
// @Produce json
// @Param name path string true "Plugin name"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/plugins/{name}/schema [get]
func (s *Server) getPluginSchema(c *gin.Context) {
	name := c.Param("name")

	plugin, exists := s.orchestrator.GetPluginManager().GetPlugin(name)
	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Plugin not found"})
		return
	}

	schema := plugin.ConfigSchema()
	c.Data(http.StatusOK, "application/json", schema)
}

// @Summary Validate plugin configuration
// @Description Validate a plugin configuration against its schema
// @Tags plugins
// @Accept json
// @Produce json
// @Param name path string true "Plugin name"
// @Param config body map[string]interface{} true "Plugin configuration"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/plugins/{name}/validate [post]
func (s *Server) validatePluginConfig(c *gin.Context) {
	name := c.Param("name")

	plugin, exists := s.orchestrator.GetPluginManager().GetPlugin(name)
	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "Plugin not found"})
		return
	}

	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Validate configuration by trying to initialize the plugin
	if err := plugin.Initialize(config); err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Clean up after validation
	plugin.Cleanup()

	c.JSON(http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}

// System handlers

// @Summary Get system metrics
// @Description Get current system metrics
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} models.SystemMetrics
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/system/metrics [get]
func (s *Server) getSystemMetrics(c *gin.Context) {
	// TODO: Get metrics from metrics collector
	// For now, return placeholder data
	
	metrics := models.SystemMetrics{
		Timestamp: time.Now(),
		CPU: models.CPUMetrics{
			UsagePercent: 45.2,
		},
		Memory: models.MemoryMetrics{
			UsagePercent: 62.8,
		},
	}

	c.JSON(http.StatusOK, metrics)
}

// @Summary Get system health
// @Description Get system health status
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/system/health [get]
func (s *Server) getSystemHealth(c *gin.Context) {
	s.healthCheck(c)
}

// @Summary Get system information
// @Description Get system information and capabilities
// @Tags system
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/system/info [get]
func (s *Server) getSystemInfo(c *gin.Context) {
	info := map[string]interface{}{
		"version":     "1.0.0",
		"build_time":  time.Now().Format(time.RFC3339),
		"go_version":  "1.21",
		"plugins":     len(s.orchestrator.GetPluginManager().ListPlugins()),
		"features": map[string]bool{
			"websocket":      true,
			"authentication": s.config.Auth.Enabled,
			"metrics":        s.config.Metrics.Enabled,
			"influxdb":       true,
		},
	}

	c.JSON(http.StatusOK, info)
}

// User handlers (placeholder - implement when auth is enabled)

func (s *Server) getUserProfile(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "User management not implemented"})
}

func (s *Server) updateUserProfile(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "User management not implemented"})
}

func (s *Server) changePassword(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "User management not implemented"})
}

// Auth handlers (placeholder)

func (s *Server) login(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "Authentication not implemented"})
}

func (s *Server) logout(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "Authentication not implemented"})
}

func (s *Server) refreshToken(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, ErrorResponse{Error: "Authentication not implemented"})
}

// Helper functions

func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	if valueStr := c.Query(key); valueStr != "" {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func calculateTestScore(execution models.TestExecution) float64 {
	// Simple scoring algorithm - can be enhanced
	if execution.Status == models.StatusCompleted {
		return 100.0
	} else if execution.Status == models.StatusFailed {
		return 0.0
	}
	return 50.0
}