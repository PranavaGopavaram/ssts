package plugins

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pranavgopavaram/ssts/pkg/models"
)

// IOStressConfig defines configuration for I/O stress testing
type IOStressConfig struct {
	FileSize      string `json:"file_size"`      // 1GB, 100MB, etc.
	BlockSize     string `json:"block_size"`     // 64KB, 1MB, etc.
	Operations    string `json:"operations"`     // read, write, mixed
	Workers       int    `json:"workers"`        // Number of worker threads
	Fsync         bool   `json:"fsync"`          // Force sync after writes
	Direct        bool   `json:"direct"`         // Use O_DIRECT for unbuffered I/O
	TempDir       string `json:"temp_dir"`       // Directory for test files
	Sequential    bool   `json:"sequential"`     // Sequential vs random I/O
	ReadWriteRatio float64 `json:"read_write_ratio"` // For mixed operations (0.0-1.0)
}

// IOStressPlugin implements I/O stress testing
type IOStressPlugin struct {
	config      IOStressConfig
	metrics     *IOMetrics
	mu          sync.RWMutex
	testFiles   []string
	stopChan    chan bool
	fileSizeBytes int64
	blockSizeBytes int64
}

// IOMetrics tracks I/O stress test metrics
type IOMetrics struct {
	ReadBytesPerSec  int64   `json:"read_bytes_per_sec"`
	WriteBytesPerSec int64   `json:"write_bytes_per_sec"`
	ReadOpsPerSec    int64   `json:"read_ops_per_sec"`
	WriteOpsPerSec   int64   `json:"write_ops_per_sec"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	IOPS             int64   `json:"iops"`
	TotalBytesRead   int64   `json:"total_bytes_read"`
	TotalBytesWritten int64  `json:"total_bytes_written"`
	ErrorCount       int64   `json:"error_count"`
}

// NewIOStressPlugin creates a new I/O stress plugin
func NewIOStressPlugin() *IOStressPlugin {
	return &IOStressPlugin{
		metrics:   &IOMetrics{},
		testFiles: make([]string, 0),
		stopChan:  make(chan bool),
	}
}

// Name returns the plugin name
func (i *IOStressPlugin) Name() string {
	return "io-stress"
}

// Version returns the plugin version
func (i *IOStressPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (i *IOStressPlugin) Description() string {
	return "I/O stress testing plugin for disk and file system performance"
}

// ConfigSchema returns the JSON schema for configuration
func (i *IOStressPlugin) ConfigSchema() []byte {
	schema := `{
		"type": "object",
		"properties": {
			"file_size": {
				"type": "string",
				"default": "1GB",
				"description": "Size of test files (e.g., 1GB, 100MB)"
			},
			"block_size": {
				"type": "string",
				"default": "64KB",
				"description": "I/O block size (e.g., 64KB, 1MB)"
			},
			"operations": {
				"type": "string",
				"enum": ["read", "write", "mixed"],
				"default": "mixed",
				"description": "Type of I/O operations to perform"
			},
			"workers": {
				"type": "integer",
				"minimum": 1,
				"maximum": 32,
				"default": 4,
				"description": "Number of worker threads"
			},
			"fsync": {
				"type": "boolean",
				"default": false,
				"description": "Force synchronous writes"
			},
			"direct": {
				"type": "boolean",
				"default": false,
				"description": "Use direct I/O (unbuffered)"
			},
			"temp_dir": {
				"type": "string",
				"default": "/tmp",
				"description": "Directory for temporary test files"
			},
			"sequential": {
				"type": "boolean",
				"default": true,
				"description": "Use sequential I/O instead of random"
			},
			"read_write_ratio": {
				"type": "number",
				"minimum": 0.0,
				"maximum": 1.0,
				"default": 0.5,
				"description": "Ratio of reads to writes for mixed operations"
			}
		}
	}`
	return []byte(schema)
}

// Initialize initializes the plugin with configuration
func (i *IOStressPlugin) Initialize(config interface{}) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(configBytes, &i.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	if i.config.FileSize == "" {
		i.config.FileSize = "1GB"
	}
	if i.config.BlockSize == "" {
		i.config.BlockSize = "64KB"
	}
	if i.config.Operations == "" {
		i.config.Operations = "mixed"
	}
	if i.config.Workers <= 0 {
		i.config.Workers = 4
	}
	if i.config.TempDir == "" {
		i.config.TempDir = "/tmp"
	}
	if i.config.ReadWriteRatio <= 0 {
		i.config.ReadWriteRatio = 0.5
	}

	// Parse sizes
	i.fileSizeBytes, err = i.parseSize(i.config.FileSize)
	if err != nil {
		return fmt.Errorf("invalid file_size: %w", err)
	}

	i.blockSizeBytes, err = i.parseSize(i.config.BlockSize)
	if err != nil {
		return fmt.Errorf("invalid block_size: %w", err)
	}

	// Validate temp directory
	if _, err := os.Stat(i.config.TempDir); os.IsNotExist(err) {
		return fmt.Errorf("temp directory does not exist: %s", i.config.TempDir)
	}

	return nil
}

// parseSize parses size strings like "1GB", "64KB"
func (i *IOStressPlugin) parseSize(size string) (int64, error) {
	size = strings.TrimSpace(strings.ToUpper(size))
	
	var multiplier int64 = 1
	if strings.HasSuffix(size, "GB") {
		multiplier = 1024 * 1024 * 1024
		size = strings.TrimSuffix(size, "GB")
	} else if strings.HasSuffix(size, "MB") {
		multiplier = 1024 * 1024
		size = strings.TrimSuffix(size, "MB")
	} else if strings.HasSuffix(size, "KB") {
		multiplier = 1024
		size = strings.TrimSuffix(size, "KB")
	} else if strings.HasSuffix(size, "B") {
		size = strings.TrimSuffix(size, "B")
	}

	value, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value: %w", err)
	}

	return value * multiplier, nil
}

// Execute runs the I/O stress test
func (i *IOStressPlugin) Execute(ctx context.Context, params models.TestParams) error {
	// Reset metrics
	i.mu.Lock()
	i.metrics = &IOMetrics{}
	i.mu.Unlock()

	// Create test files
	if err := i.createTestFiles(ctx); err != nil {
		return fmt.Errorf("failed to create test files: %w", err)
	}

	// Start metrics collection
	go i.collectMetrics(ctx)

	// Start I/O workers
	var wg sync.WaitGroup
	for workerID := 0; workerID < i.config.Workers; workerID++ {
		wg.Add(1)
		go i.ioWorker(ctx, &wg, workerID)
	}

	// Wait for completion or context cancellation
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

// createTestFiles creates the test files for I/O operations
func (i *IOStressPlugin) createTestFiles(ctx context.Context) error {
	for workerID := 0; workerID < i.config.Workers; workerID++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		filename := filepath.Join(i.config.TempDir, fmt.Sprintf("ssts_io_test_%d_%d.dat", 
			time.Now().Unix(), workerID))

		if err := i.createTestFile(filename); err != nil {
			return fmt.Errorf("failed to create test file %s: %w", filename, err)
		}

		i.mu.Lock()
		i.testFiles = append(i.testFiles, filename)
		i.mu.Unlock()
	}

	return nil
}

// createTestFile creates a single test file with random data
func (i *IOStressPlugin) createTestFile(filename string) error {
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if i.config.Direct {
		// Note: O_DIRECT is not available on all platforms
		// In a production implementation, this would be handled differently
		flags |= os.O_SYNC
	}

	file, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write test data in blocks
	buffer := make([]byte, i.blockSizeBytes)
	bytesWritten := int64(0)

	for bytesWritten < i.fileSizeBytes {
		select {
		default:
		}

		remaining := i.fileSizeBytes - bytesWritten
		if remaining < i.blockSizeBytes {
			buffer = buffer[:remaining]
		}

		// Fill buffer with random data
		if _, err := rand.Read(buffer); err != nil {
			return err
		}

		n, err := file.Write(buffer)
		if err != nil {
			return err
		}

		bytesWritten += int64(n)

		if i.config.Fsync {
			if err := file.Sync(); err != nil {
				return err
			}
		}
	}

	return nil
}

// ioWorker performs I/O operations
func (i *IOStressPlugin) ioWorker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	i.mu.RLock()
	if workerID >= len(i.testFiles) {
		i.mu.RUnlock()
		return
	}
	filename := i.testFiles[workerID]
	i.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return
		case <-i.stopChan:
			return
		default:
		}

		start := time.Now()
		err := i.performIOOperation(filename)
		latency := time.Since(start)

		i.mu.Lock()
		if err != nil {
			i.metrics.ErrorCount++
		} else {
			i.metrics.AvgLatencyMs = float64(latency.Nanoseconds()) / 1000000.0
		}
		i.mu.Unlock()

		// Small delay to prevent overwhelming the system
		time.Sleep(1 * time.Millisecond)
	}
}

// performIOOperation performs a single I/O operation
func (i *IOStressPlugin) performIOOperation(filename string) error {
	operation := i.config.Operations
	if operation == "mixed" {
		// Decide based on read/write ratio
		if float64(time.Now().UnixNano()%1000)/1000.0 < i.config.ReadWriteRatio {
			operation = "read"
		} else {
			operation = "write"
		}
	}

	switch operation {
	case "read":
		return i.performRead(filename)
	case "write":
		return i.performWrite(filename)
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

// performRead performs a read operation
func (i *IOStressPlugin) performRead(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, i.blockSizeBytes)
	
	// Determine read position
	var offset int64
	if !i.config.Sequential {
		// Random position
		maxOffset := i.fileSizeBytes - i.blockSizeBytes
		if maxOffset > 0 {
			offset = int64(time.Now().UnixNano()) % maxOffset
		}
	}

	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}

	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return err
	}

	// Update metrics
	i.mu.Lock()
	i.metrics.TotalBytesRead += int64(n)
	i.metrics.ReadOpsPerSec++
	i.mu.Unlock()

	return nil
}

// performWrite performs a write operation
func (i *IOStressPlugin) performWrite(filename string) error {
	flags := os.O_WRONLY
	if i.config.Direct {
		flags |= os.O_SYNC
	}

	file, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, i.blockSizeBytes)
	if _, err := rand.Read(buffer); err != nil {
		return err
	}

	// Determine write position
	var offset int64
	if !i.config.Sequential {
		// Random position
		maxOffset := i.fileSizeBytes - i.blockSizeBytes
		if maxOffset > 0 {
			offset = int64(time.Now().UnixNano()) % maxOffset
		}
	}

	if _, err := file.Seek(offset, 0); err != nil {
		return err
	}

	n, err := file.Write(buffer)
	if err != nil {
		return err
	}

	if i.config.Fsync {
		if err := file.Sync(); err != nil {
			return err
		}
	}

	// Update metrics
	i.mu.Lock()
	i.metrics.TotalBytesWritten += int64(n)
	i.metrics.WriteOpsPerSec++
	i.mu.Unlock()

	return nil
}

// collectMetrics collects performance metrics
func (i *IOStressPlugin) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastBytesRead, lastBytesWritten int64
	var lastReadOps, lastWriteOps int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			i.mu.Lock()
			
			// Calculate per-second rates
			currentBytesRead := i.metrics.TotalBytesRead
			currentBytesWritten := i.metrics.TotalBytesWritten
			currentReadOps := i.metrics.ReadOpsPerSec
			currentWriteOps := i.metrics.WriteOpsPerSec

			i.metrics.ReadBytesPerSec = currentBytesRead - lastBytesRead
			i.metrics.WriteBytesPerSec = currentBytesWritten - lastBytesWritten
			i.metrics.IOPS = (currentReadOps - lastReadOps) + (currentWriteOps - lastWriteOps)

			lastBytesRead = currentBytesRead
			lastBytesWritten = currentBytesWritten
			lastReadOps = currentReadOps
			lastWriteOps = currentWriteOps
			
			i.mu.Unlock()
		}
	}
}

// Cleanup cleans up test files and resources
func (i *IOStressPlugin) Cleanup() error {
	close(i.stopChan)

	// Remove test files
	i.mu.Lock()
	for _, filename := range i.testFiles {
		if err := os.Remove(filename); err != nil {
			// Log error but don't fail cleanup
			fmt.Printf("Warning: failed to remove test file %s: %v\n", filename, err)
		}
	}
	i.testFiles = i.testFiles[:0]
	i.mu.Unlock()

	return nil
}

// GetMetrics returns current metrics
func (i *IOStressPlugin) GetMetrics() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return map[string]interface{}{
		"read_bytes_per_sec":  i.metrics.ReadBytesPerSec,
		"write_bytes_per_sec": i.metrics.WriteBytesPerSec,
		"read_ops_per_sec":    i.metrics.ReadOpsPerSec,
		"write_ops_per_sec":   i.metrics.WriteOpsPerSec,
		"avg_latency_ms":      i.metrics.AvgLatencyMs,
		"iops":                i.metrics.IOPS,
		"total_bytes_read":    i.metrics.TotalBytesRead,
		"total_bytes_written": i.metrics.TotalBytesWritten,
		"error_count":         i.metrics.ErrorCount,
	}
}

// GetSafetyLimits returns safety limits for I/O testing
func (i *IOStressPlugin) GetSafetyLimits() models.SafetyLimits {
	return models.SafetyLimits{
		MaxCPUPercent:    30.0, // I/O test shouldn't use much CPU
		MaxMemoryPercent: 20.0, // Minimal memory usage
		MaxDiskPercent:   95.0, // Allow high disk usage
		MaxNetworkMbps:   10.0,
	}
}

// HealthCheck performs a health check
func (i *IOStressPlugin) HealthCheck() error {
	// Create a small test file to verify I/O functionality
	testFile := filepath.Join(i.config.TempDir, "ssts_health_check.tmp")
	
	// Test write
	if err := i.writeTestData(testFile); err != nil {
		return fmt.Errorf("I/O health check write failed: %w", err)
	}
	
	// Test read
	if err := i.readTestData(testFile); err != nil {
		os.Remove(testFile)
		return fmt.Errorf("I/O health check read failed: %w", err)
	}
	
	// Clean up
	os.Remove(testFile)
	return nil
}

func (i *IOStressPlugin) writeTestData(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	testData := []byte("SSTS I/O Health Check Test Data")
	_, err = file.Write(testData)
	return err
}

func (i *IOStressPlugin) readTestData(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	buffer := make([]byte, 100)
	_, err = file.Read(buffer)
	return err
}