package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pranavgopavaram/ssts/pkg/models"
)

// MemoryStressConfig defines configuration for memory stress testing
type MemoryStressConfig struct {
	AllocSize    string `json:"alloc_size"`    // 1GB, 500MB, etc.
	Pattern      string `json:"pattern"`       // sequential, random, fragmented
	AccessType   string `json:"access_type"`   // read, write, readwrite
	Workers      int    `json:"workers"`       // Number of worker threads
	ChunkSize    string `json:"chunk_size"`    // Size of individual allocations
	AccessDelay  int    `json:"access_delay"`  // Delay between accesses in ms
}

// MemoryStressPlugin implements memory stress testing
type MemoryStressPlugin struct {
	config       MemoryStressConfig
	metrics      *MemoryMetrics
	mu           sync.RWMutex
	allocations  [][]byte
	stopChan     chan bool
	allocSizeMB  int64
	chunkSizeMB  int64
}

// MemoryMetrics tracks memory stress test metrics
type MemoryMetrics struct {
	AllocationRate int64   `json:"alloc_rate_mb_per_sec"`
	AccessLatency  float64 `json:"access_latency_ns"`
	PageFaults     int64   `json:"page_faults_per_sec"`
	CacheHitRatio  float64 `json:"cache_hit_ratio"`
	AllocatedMB    int64   `json:"allocated_mb"`
	AccessCount    int64   `json:"access_count"`
}

// NewMemoryStressPlugin creates a new memory stress plugin
func NewMemoryStressPlugin() *MemoryStressPlugin {
	return &MemoryStressPlugin{
		metrics:     &MemoryMetrics{},
		allocations: make([][]byte, 0),
		stopChan:    make(chan bool),
	}
}

// Name returns the plugin name
func (m *MemoryStressPlugin) Name() string {
	return "memory-stress"
}

// Version returns the plugin version
func (m *MemoryStressPlugin) Version() string {
	return "1.0.0"
}

// Description returns the plugin description
func (m *MemoryStressPlugin) Description() string {
	return "Memory stress testing plugin with various allocation patterns"
}

// ConfigSchema returns the JSON schema for configuration
func (m *MemoryStressPlugin) ConfigSchema() []byte {
	schema := `{
		"type": "object",
		"properties": {
			"alloc_size": {
				"type": "string",
				"default": "1GB",
				"description": "Total amount of memory to allocate (e.g., 1GB, 500MB)"
			},
			"pattern": {
				"type": "string",
				"enum": ["sequential", "random", "fragmented"],
				"default": "sequential",
				"description": "Memory allocation pattern"
			},
			"access_type": {
				"type": "string",
				"enum": ["read", "write", "readwrite"],
				"default": "readwrite",
				"description": "Type of memory access operations"
			},
			"workers": {
				"type": "integer",
				"minimum": 1,
				"maximum": 64,
				"default": 4,
				"description": "Number of worker threads"
			},
			"chunk_size": {
				"type": "string",
				"default": "64MB",
				"description": "Size of individual memory chunks"
			},
			"access_delay": {
				"type": "integer",
				"minimum": 0,
				"maximum": 1000,
				"default": 10,
				"description": "Delay between memory accesses in milliseconds"
			}
		}
	}`
	return []byte(schema)
}

// Initialize initializes the plugin with configuration
func (m *MemoryStressPlugin) Initialize(config interface{}) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(configBytes, &m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	if m.config.AllocSize == "" {
		m.config.AllocSize = "1GB"
	}
	if m.config.Pattern == "" {
		m.config.Pattern = "sequential"
	}
	if m.config.AccessType == "" {
		m.config.AccessType = "readwrite"
	}
	if m.config.Workers <= 0 {
		m.config.Workers = 4
	}
	if m.config.ChunkSize == "" {
		m.config.ChunkSize = "64MB"
	}

	// Parse memory sizes
	m.allocSizeMB, err = m.parseMemorySize(m.config.AllocSize)
	if err != nil {
		return fmt.Errorf("invalid alloc_size: %w", err)
	}

	m.chunkSizeMB, err = m.parseMemorySize(m.config.ChunkSize)
	if err != nil {
		return fmt.Errorf("invalid chunk_size: %w", err)
	}

	return nil
}

// parseMemorySize parses memory size strings like "1GB", "500MB"
func (m *MemoryStressPlugin) parseMemorySize(size string) (int64, error) {
	size = strings.TrimSpace(strings.ToUpper(size))
	
	var multiplier int64 = 1
	if strings.HasSuffix(size, "GB") {
		multiplier = 1024
		size = strings.TrimSuffix(size, "GB")
	} else if strings.HasSuffix(size, "MB") {
		size = strings.TrimSuffix(size, "MB")
	} else {
		return 0, fmt.Errorf("invalid memory size format, expected MB or GB suffix")
	}

	value, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory size value: %w", err)
	}

	return value * multiplier, nil
}

// Execute runs the memory stress test
func (m *MemoryStressPlugin) Execute(ctx context.Context, params models.TestParams) error {
	m.mu.Lock()
	m.metrics.AccessCount = 0
	m.metrics.AllocatedMB = 0
	m.mu.Unlock()

	// Start metrics collection
	go m.collectMetrics(ctx)

	// Calculate number of chunks needed
	numChunks := m.allocSizeMB / m.chunkSizeMB
	if numChunks <= 0 {
		numChunks = 1
	}

	// Allocate memory based on pattern
	if err := m.allocateMemory(ctx, int(numChunks)); err != nil {
		return fmt.Errorf("memory allocation failed: %w", err)
	}

	// Start memory access workers
	var wg sync.WaitGroup
	for i := 0; i < m.config.Workers; i++ {
		wg.Add(1)
		go m.memoryAccessWorker(ctx, &wg, i)
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

// allocateMemory allocates memory chunks based on the configured pattern
func (m *MemoryStressPlugin) allocateMemory(ctx context.Context, numChunks int) error {
	chunkBytes := m.chunkSizeMB * 1024 * 1024
	
	for i := 0; i < numChunks; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Allocate chunk
		chunk := make([]byte, chunkBytes)
		
		// Initialize based on pattern
		switch m.config.Pattern {
		case "sequential":
			m.initializeSequential(chunk)
		case "random":
			m.initializeRandom(chunk)
		case "fragmented":
			m.initializeFragmented(chunk, i)
		}

		m.mu.Lock()
		m.allocations = append(m.allocations, chunk)
		m.metrics.AllocatedMB += m.chunkSizeMB
		m.mu.Unlock()

		// Force garbage collection periodically
		if i%10 == 0 {
			runtime.GC()
		}

		// Small delay to prevent overwhelming the system
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

// initializeSequential initializes memory with sequential pattern
func (m *MemoryStressPlugin) initializeSequential(chunk []byte) {
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}
}

// initializeRandom initializes memory with random pattern
func (m *MemoryStressPlugin) initializeRandom(chunk []byte) {
	rand.Read(chunk)
}

// initializeFragmented initializes memory with fragmented pattern
func (m *MemoryStressPlugin) initializeFragmented(chunk []byte, chunkIndex int) {
	blockSize := 4096 // 4KB blocks
	for i := 0; i < len(chunk); i += blockSize {
		end := i + blockSize
		if end > len(chunk) {
			end = len(chunk)
		}
		
		// Fill every other block
		if (i/blockSize+chunkIndex)%2 == 0 {
			for j := i; j < end; j++ {
				chunk[j] = byte(j % 256)
			}
		}
	}
}

// memoryAccessWorker performs memory access operations
func (m *MemoryStressPlugin) memoryAccessWorker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	accessDelay := time.Duration(m.config.AccessDelay) * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		default:
		}

		m.mu.RLock()
		numAllocations := len(m.allocations)
		m.mu.RUnlock()

		if numAllocations == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Select random allocation
		allocIndex := rand.Intn(numAllocations)
		
		start := time.Now()
		m.performMemoryAccess(allocIndex)
		latency := time.Since(start)

		// Update metrics
		m.mu.Lock()
		m.metrics.AccessCount++
		m.metrics.AccessLatency = float64(latency.Nanoseconds())
		m.mu.Unlock()

		if accessDelay > 0 {
			time.Sleep(accessDelay)
		}
	}
}

// performMemoryAccess performs the configured type of memory access
func (m *MemoryStressPlugin) performMemoryAccess(allocIndex int) {
	m.mu.RLock()
	if allocIndex >= len(m.allocations) {
		m.mu.RUnlock()
		return
	}
	chunk := m.allocations[allocIndex]
	m.mu.RUnlock()

	// Random offset within chunk
	offset := rand.Intn(len(chunk) - 1024)
	if offset < 0 {
		offset = 0
	}

	switch m.config.AccessType {
	case "read":
		m.performRead(chunk, offset)
	case "write":
		m.performWrite(chunk, offset)
	case "readwrite":
		if rand.Intn(2) == 0 {
			m.performRead(chunk, offset)
		} else {
			m.performWrite(chunk, offset)
		}
	}
}

// performRead performs memory read operations
func (m *MemoryStressPlugin) performRead(chunk []byte, offset int) {
	// Read 1KB of data
	sum := 0
	for i := offset; i < offset+1024 && i < len(chunk); i++ {
		sum += int(chunk[i])
	}
	_ = sum // Prevent optimization
}

// performWrite performs memory write operations
func (m *MemoryStressPlugin) performWrite(chunk []byte, offset int) {
	// Write 1KB of data
	value := byte(rand.Intn(256))
	for i := offset; i < offset+1024 && i < len(chunk); i++ {
		chunk[i] = value
	}
}

// collectMetrics collects performance metrics
func (m *MemoryStressPlugin) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastAllocatedMB int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			currentAlloc := m.metrics.AllocatedMB
			
			// Calculate rates
			m.metrics.AllocationRate = currentAlloc - lastAllocatedMB
			lastAllocatedMB = currentAlloc
			
			m.mu.Unlock()
		}
	}
}

// Cleanup cleans up allocated memory and resources
func (m *MemoryStressPlugin) Cleanup() error {
	close(m.stopChan)
	
	m.mu.Lock()
	// Clear allocations to allow garbage collection
	m.allocations = m.allocations[:0]
	m.mu.Unlock()
	
	// Force garbage collection
	runtime.GC()
	
	return nil
}

// GetMetrics returns current metrics
func (m *MemoryStressPlugin) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"alloc_rate_mb_per_sec": m.metrics.AllocationRate,
		"access_latency_ns":     m.metrics.AccessLatency,
		"page_faults_per_sec":   m.metrics.PageFaults,
		"cache_hit_ratio":       m.metrics.CacheHitRatio,
		"allocated_mb":          m.metrics.AllocatedMB,
		"access_count":          m.metrics.AccessCount,
		"num_allocations":       len(m.allocations),
	}
}

// GetSafetyLimits returns safety limits for memory testing
func (m *MemoryStressPlugin) GetSafetyLimits() models.SafetyLimits {
	return models.SafetyLimits{
		MaxCPUPercent:    30.0, // Memory test shouldn't use much CPU
		MaxMemoryPercent: 85.0, // Allow high memory usage
		MaxDiskPercent:   50.0,
		MaxNetworkMbps:   10.0,
	}
}

// HealthCheck performs a health check
func (m *MemoryStressPlugin) HealthCheck() error {
	// Allocate a small test chunk
	testChunk := make([]byte, 1024)
	for i := range testChunk {
		testChunk[i] = byte(i % 256)
	}
	
	// Verify data integrity
	for i := range testChunk {
		if testChunk[i] != byte(i%256) {
			return fmt.Errorf("memory health check failed: data corruption detected")
		}
	}
	
	return nil
}