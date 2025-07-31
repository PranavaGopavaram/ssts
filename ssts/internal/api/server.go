package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/pranavgopavaram/ssts/internal/config"
	"github.com/pranavgopavaram/ssts/internal/core"
	"github.com/pranavgopavaram/ssts/internal/database"
	"github.com/pranavgopavaram/ssts/pkg/models"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	db           *database.Database
	influxDB     *database.InfluxDB
	orchestrator *core.Orchestrator
	wsHub        *WebSocketHub
	logger       *zap.Logger
	engine       *gin.Engine
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *database.Database, orchestrator *core.Orchestrator, logger *zap.Logger) *Server {
	// Initialize InfluxDB
	influxDB := database.NewInfluxDB(cfg.InfluxDB)

	// Initialize WebSocket hub
	wsHub := NewWebSocketHub()
	go wsHub.Run()

	server := &Server{
		config:       cfg,
		db:           db,
		influxDB:     influxDB,
		orchestrator: orchestrator,
		wsHub:        wsHub,
		logger:       logger,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Configure gin mode
	if s.config.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	s.engine = gin.New()

	// Middleware
	s.engine.Use(gin.Recovery())
	s.engine.Use(s.loggingMiddleware())
	s.engine.Use(s.corsMiddleware())

	// Health check
	s.engine.GET("/health", s.healthCheck)

	// API routes
	api := s.engine.Group("/api/v1")
	{
		// Authentication routes (if enabled)
		if s.config.Auth.Enabled {
			auth := api.Group("/auth")
			{
				auth.POST("/login", s.login)
				auth.POST("/logout", s.logout)
				auth.POST("/refresh", s.refreshToken)
			}
			// Apply auth middleware to protected routes
			api.Use(s.authMiddleware())
		}

		// Test configuration routes
		tests := api.Group("/tests")
		{
			tests.GET("", s.listTests)
			tests.POST("", s.createTest)
			tests.GET("/:id", s.getTest)
			tests.PUT("/:id", s.updateTest)
			tests.DELETE("/:id", s.deleteTest)
			tests.POST("/:id/run", s.runTest)
			tests.POST("/:id/stop", s.stopTest)
			tests.GET("/:id/status", s.getTestStatus)
			tests.GET("/:id/results", s.getTestResults)
			tests.GET("/:id/metrics", s.getTestMetrics)
			tests.POST("/:id/export", s.exportTestData)
		}

		// Test execution routes
		executions := api.Group("/executions")
		{
			executions.GET("", s.listExecutions)
			executions.GET("/:id", s.getExecution)
			executions.POST("/:id/stop", s.stopExecution)
			executions.GET("/:id/metrics", s.getExecutionMetrics)
			executions.GET("/:id/logs", s.getExecutionLogs)
		}

		// Plugin routes
		plugins := api.Group("/plugins")
		{
			plugins.GET("", s.listPlugins)
			plugins.GET("/:name", s.getPlugin)
			plugins.GET("/:name/schema", s.getPluginSchema)
			plugins.POST("/:name/validate", s.validatePluginConfig)
		}

		// System routes
		system := api.Group("/system")
		{
			system.GET("/metrics", s.getSystemMetrics)
			system.GET("/health", s.getSystemHealth)
			system.GET("/info", s.getSystemInfo)
		}

		// User routes (if auth enabled)
		if s.config.Auth.Enabled {
			users := api.Group("/users")
			{
				users.GET("/profile", s.getUserProfile)
				users.PUT("/profile", s.updateUserProfile)
				users.POST("/change-password", s.changePassword)
			}
		}
	}

	// WebSocket endpoint
	s.engine.GET("/ws", s.handleWebSocket)

	// Swagger documentation
	s.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Static files for web dashboard
	s.engine.Static("/static", "./web/dist/static")
	s.engine.StaticFile("/", "./web/dist/index.html")
	s.engine.StaticFile("/favicon.ico", "./web/dist/favicon.ico")

	// Catch-all for SPA routing
	s.engine.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == "/" || !gin.IsDebugging() {
			c.File("./web/dist/index.html")
		} else {
			c.Status(404)
		}
	})
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Address, s.config.Server.Port)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
	}

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		s.logger.Info("Starting HTTP server", zap.String("address", addr))

		if s.config.Server.TLS.Enabled {
			serverErr <- server.ListenAndServeTLS(
				s.config.Server.TLS.CertFile,
				s.config.Server.TLS.KeyFile,
			)
		} else {
			serverErr <- server.ListenAndServe()
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		s.logger.Info("Shutting down HTTP server")

		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server shutdown error", zap.Error(err))
			return err
		}

		s.logger.Info("HTTP server stopped")
		return nil
	}
}

// Middleware functions

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		if raw != "" {
			path = path + "?" + raw
		}

		s.logger.Info("HTTP request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", c.ClientIP()),
		)
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     s.config.Server.CORS.AllowOrigins,
		AllowMethods:     s.config.Server.CORS.AllowMethods,
		AllowHeaders:     s.config.Server.CORS.AllowHeaders,
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	return cors.New(config)
}

func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement JWT authentication
		// For now, just pass through
		c.Next()
	}
}

// Health check endpoint
func (s *Server) healthCheck(c *gin.Context) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"services":  make(map[string]string),
	}

	// Check database health
	if err := s.db.HealthCheck(); err != nil {
		health["services"].(map[string]string)["database"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["services"].(map[string]string)["database"] = "healthy"
	}

	// Check InfluxDB health
	if err := s.influxDB.HealthCheck(context.Background()); err != nil {
		health["services"].(map[string]string)["influxdb"] = "unhealthy"
		health["status"] = "degraded"
	} else {
		health["services"].(map[string]string)["influxdb"] = "healthy"
	}

	if health["status"] == "healthy" {
		c.JSON(http.StatusOK, health)
	} else {
		c.JSON(http.StatusServiceUnavailable, health)
	}
}

// Test configuration handlers

// @Summary List test configurations
// @Description Get a list of all test configurations
// @Tags tests
// @Accept json
// @Produce json
// @Param limit query int false "Limit number of results" default(50)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} models.TestConfiguration
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests [get]
func (s *Server) listTests(c *gin.Context) {
	limit := c.DefaultQuery("limit", "50")
	offset := c.DefaultQuery("offset", "0")

	repo := database.NewRepository(s.db)
	tests, err := repo.ListTestConfigurations(parseInt(limit, 50), parseInt(offset, 0))
	if err != nil {
		s.logger.Error("Failed to list tests", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to list tests"})
		return
	}

	c.JSON(http.StatusOK, tests)
}

// @Summary Create test configuration
// @Description Create a new test configuration
// @Tags tests
// @Accept json
// @Produce json
// @Param test body models.TestConfiguration true "Test configuration"
// @Success 201 {object} models.TestConfiguration
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests [post]
func (s *Server) createTest(c *gin.Context) {
	var test models.TestConfiguration
	if err := c.ShouldBindJSON(&test); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	// Set creation time and ID
	test.Created = time.Now()
	test.Updated = time.Now()

	repo := database.NewRepository(s.db)
	if err := repo.CreateTestConfiguration(&test); err != nil {
		s.logger.Error("Failed to create test", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to create test"})
		return
	}

	c.JSON(http.StatusCreated, test)
}

// @Summary Get test configuration
// @Description Get a specific test configuration by ID
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Success 200 {object} models.TestConfiguration
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id} [get]
func (s *Server) getTest(c *gin.Context) {
	id := c.Param("id")

	repo := database.NewRepository(s.db)
	test, err := repo.GetTestConfiguration(id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Test not found"})
		} else {
			s.logger.Error("Failed to get test", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get test"})
		}
		return
	}

	c.JSON(http.StatusOK, test)
}

// @Summary Run test
// @Description Execute a test configuration
// @Tags tests
// @Accept json
// @Produce json
// @Param id path string true "Test ID"
// @Param params body models.TestParams true "Test execution parameters"
// @Success 202 {object} TestExecutionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tests/{id}/run [post]
func (s *Server) runTest(c *gin.Context) {
	id := c.Param("id")

	var params models.TestParams
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	repo := database.NewRepository(s.db)
	test, err := repo.GetTestConfiguration(id)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "Test not found"})
		} else {
			s.logger.Error("Failed to get test", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get test"})
		}
		return
	}

	// Use test duration if not specified in params
	if params.Duration == 0 {
		params.Duration = test.Duration
	}

	// Start test execution
	executionID, err := s.orchestrator.StartTest(*test, params)
	if err != nil {
		s.logger.Error("Failed to start test", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to start test"})
		return
	}

	response := TestExecutionResponse{
		ExecutionID: executionID,
		Status:      "started",
		Message:     "Test execution started successfully",
	}

	c.JSON(http.StatusAccepted, response)
}

// WebSocket handler for real-time updates
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	client := &WSClient{
		hub:  s.wsHub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}

// Helper functions

func parseInt(s string, defaultValue int) int {
	if len(s) == 0 {
		return defaultValue
	}
	// Simple int parsing - replace with strconv.Atoi in production
	return defaultValue
}

// Response types

type ErrorResponse struct {
	Error string `json:"error"`
}

type TestExecutionResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}
