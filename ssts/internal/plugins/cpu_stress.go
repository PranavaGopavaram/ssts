package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"github.com/pranavgopavaram/ssts/pkg/models"
)

// CPUStressConfig defines the configuration for CPU stress testing
type CPUStressConfig struct {
	Workers   int    `json:"workers"`                      // Number of worker goroutines (0 = number of CPUs)
	Algorithm string `json:"algorithm"`                    // prime, fibonacci, matrix, pi
	Intensity int    `json:"intensity"`                    // 1-100 scale
	RampUp    bool   `json:"ramp_up" default:"true"`      // Gradual intensity increase
}

// CPUStressPlugin implements CPU stress testing
type CPUStressPlugin struct {
	config          CPUStressConfig
	metrics         *CPUMetrics
	mu              sync.RWMutex
	stopChan        chan bool
	currentWorkers  int
	operationsCount int64
}

// CPUMetrics tracks CPU stress test metrics
type CPUMetrics struct {
	OperationsPerSecond int64   `json:"ops_per_sec"`
	CalculationAccuracy float64 `json:"accuracy_percent"`
	ThermalThrottling   bool    `json:"thermal_throttle"`
	CoreUtilization     []float64 `json:"core_usage"`
	WorkerCount         int     `json:"worker_count"`
}

// NewCPUStressPlugin creates a new CPU stress plugin
func NewCPUStressPlugin() *CPUStressPlugin {
	return &CPUStressPlugin{
		metrics:  &CPUMetrics{},
		stopChan: make(chan bool),
	}
}

// Name returns the plugin name
func (c *CPUStressPlugin) Name() string {
	return "cpu-stress"
}

// Version returns the plugin version
func (c *CPUStressPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (c *CPUStressPlugin) Description() string {
	return "CPU stress testing plugin with multiple algorithms"
}

// ConfigSchema returns the JSON schema for configuration
func (c *CPUStressPlugin) ConfigSchema() []byte {
	schema := `{
		"type": "object",
		"properties": {
			"workers": {
				"type": "integer",
				"minimum": 0,
				"maximum": 256,
				"default": 0,
				"description": "Number of worker threads (0 = number of CPUs)"
			},
			"algorithm": {
				"type": "string",
				"enum": ["prime", "fibonacci", "matrix", "pi"],
				"default": "prime",
				"description": "CPU stress algorithm to use"
			},
			"intensity": {
				"type": "integer",
				"minimum": 1,
				"maximum": 100,
				"default": 70,
				"description": "Test intensity from 1-100"
			},
			"ramp_up": {
				"type": "boolean",
				"default": true,
				"description": "Enable gradual intensity ramp-up"
			}
		},
		"required": ["algorithm"]
	}`
	return []byte(schema)
}

// Initialize initializes the plugin with configuration
func (c *CPUStressPlugin) Initialize(config interface{}) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(configBytes, &c.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	if c.config.Workers <= 0 {
		c.config.Workers = runtime.NumCPU()
	}
	if c.config.Intensity <= 0 {
		c.config.Intensity = 70
	}
	if c.config.Algorithm == "" {
		c.config.Algorithm = "prime"
	}

	c.currentWorkers = c.config.Workers
	c.metrics.WorkerCount = c.currentWorkers

	return nil
}

// Execute runs the CPU stress test
func (c *CPUStressPlugin) Execute(ctx context.Context, params models.TestParams) error {
	c.mu.Lock()
	c.operationsCount = 0
	c.mu.Unlock()

	var wg sync.WaitGroup
	
	// Start metrics collection
	go c.collectMetrics(ctx)

	// Ramp up if enabled
	if c.config.RampUp {
		return c.executeWithRampUp(ctx, params, &wg)
	}

	return c.executeFullIntensity(ctx, params, &wg)
}

// executeWithRampUp gradually increases intensity
func (c *CPUStressPlugin) executeWithRampUp(ctx context.Context, params models.TestParams, wg *sync.WaitGroup) error {
	rampUpDuration := time.Duration(float64(params.Duration) * 0.1) // 10% of total duration
	if rampUpDuration < 10*time.Second {
		rampUpDuration = 10 * time.Second
	}

	steps := 10
	stepDuration := rampUpDuration / time.Duration(steps)
	
	for step := 1; step <= steps; step++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		intensity := (c.config.Intensity * step) / steps
		c.startWorkers(ctx, intensity, wg)
		
		time.Sleep(stepDuration)
	}

	// Run at full intensity for remaining time
	remainingDuration := params.Duration - rampUpDuration
	time.Sleep(remainingDuration)

	return nil
}

// executeFullIntensity runs at full intensity immediately
func (c *CPUStressPlugin) executeFullIntensity(ctx context.Context, params models.TestParams, wg *sync.WaitGroup) error {
	c.startWorkers(ctx, c.config.Intensity, wg)
	
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(params.Duration):
		return nil
	}
}

// startWorkers starts the CPU stress workers
func (c *CPUStressPlugin) startWorkers(ctx context.Context, intensity int, wg *sync.WaitGroup) {
	for i := 0; i < c.currentWorkers; i++ {
		wg.Add(1)
		go c.worker(ctx, intensity, wg)
	}
}

// worker performs CPU intensive operations
func (c *CPUStressPlugin) worker(ctx context.Context, intensity int, wg *sync.WaitGroup) {
	defer wg.Done()

	// Calculate work/sleep ratio based on intensity
	workTime := time.Duration(intensity) * time.Millisecond
	sleepTime := time.Duration(100-intensity) * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		default:
		}

		// Perform CPU intensive work
		start := time.Now()
		c.performWork()
		workDuration := time.Since(start)

		// Increment operations counter
		c.mu.Lock()
		c.operationsCount++
		c.mu.Unlock()

		// Sleep if needed to maintain intensity
		if workDuration < workTime && sleepTime > 0 {
			time.Sleep(sleepTime)
		}
	}
}

// performWork executes the configured algorithm
func (c *CPUStressPlugin) performWork() {
	switch c.config.Algorithm {
	case "prime":
		c.calculatePrimes(10000)
	case "fibonacci":
		c.calculateFibonacci(35)
	case "matrix":
		c.matrixMultiplication(100)
	case "pi":
		c.calculatePi(1000000)
	default:
		c.calculatePrimes(10000)
	}
}

// calculatePrimes finds prime numbers up to n
func (c *CPUStressPlugin) calculatePrimes(n int) {
	for i := 2; i <= n; i++ {
		isPrime := true
		for j := 2; j*j <= i; j++ {
			if i%j == 0 {
				isPrime = false
				break
			}
		}
		_ = isPrime
	}
}

// calculateFibonacci calculates fibonacci number (recursive)
func (c *CPUStressPlugin) calculateFibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return c.calculateFibonacci(n-1) + c.calculateFibonacci(n-2)
}

// matrixMultiplication performs matrix multiplication
func (c *CPUStressPlugin) matrixMultiplication(size int) {
	a := make([][]float64, size)
	b := make([][]float64, size)
	result := make([][]float64, size)

	// Initialize matrices
	for i := 0; i < size; i++ {
		a[i] = make([]float64, size)
		b[i] = make([]float64, size)
		result[i] = make([]float64, size)
		for j := 0; j < size; j++ {
			a[i][j] = float64(i + j)
			b[i][j] = float64(i * j)
		}
	}

	// Multiply matrices
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			for k := 0; k < size; k++ {
				result[i][j] += a[i][k] * b[k][j]
			}
		}
	}
}

// calculatePi calculates pi using Monte Carlo method
func (c *CPUStressPlugin) calculatePi(iterations int) float64 {
	inside := 0
	for i := 0; i < iterations; i++ {
		x := float64(i%1000) / 1000.0
		y := float64((i*7)%1000) / 1000.0
		if math.Sqrt(x*x+y*y) <= 1.0 {
			inside++
		}
	}
	return 4.0 * float64(inside) / float64(iterations)
}

// collectMetrics collects performance metrics
func (c *CPUStressPlugin) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastOpsCount int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.Lock()
			currentOps := c.operationsCount
			c.metrics.OperationsPerSecond = currentOps - lastOpsCount
			lastOpsCount = currentOps
			c.mu.Unlock()
		}
	}
}

// Cleanup cleans up resources
func (c *CPUStressPlugin) Cleanup() error {
	close(c.stopChan)
	return nil
}

// GetMetrics returns current metrics
func (c *CPUStressPlugin) GetMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"ops_per_sec":        c.metrics.OperationsPerSecond,
		"accuracy_percent":   c.metrics.CalculationAccuracy,
		"thermal_throttle":   c.metrics.ThermalThrottling,
		"core_usage":         c.metrics.CoreUtilization,
		"worker_count":       c.metrics.WorkerCount,
		"total_operations":   c.operationsCount,
	}
}

// GetSafetyLimits returns safety limits for CPU testing
func (c *CPUStressPlugin) GetSafetyLimits() models.SafetyLimits {
	return models.SafetyLimits{
		MaxCPUPercent:    95.0,
		MaxMemoryPercent: 20.0, // CPU test shouldn't use much memory
		MaxDiskPercent:   50.0,
		MaxNetworkMbps:   10.0,
	}
}

// HealthCheck performs a health check
func (c *CPUStressPlugin) HealthCheck() error {
	// Perform a quick calculation to verify CPU functionality
	result := c.calculateFibonacci(10)
	if result != 55 {
		return fmt.Errorf("CPU health check failed: expected 55, got %d", result)
	}
	return nil
}