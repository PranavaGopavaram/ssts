package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"

	"github.com/pranavgopavaram/ssts/internal/plugins"
	"github.com/pranavgopavaram/ssts/pkg/models"
)

type SystemMetrics struct {
	Timestamp time.Time `json:"timestamp"`
	CPU       struct {
		Usage float64 `json:"usage"`
		Cores int     `json:"cores"`
	} `json:"cpu"`
	Memory struct {
		Total     uint64  `json:"total"`
		Used      uint64  `json:"used"`
		Available uint64  `json:"available"`
		Usage     float64 `json:"usage"`
	} `json:"memory"`
	Disk struct {
		Total uint64  `json:"total"`
		Used  uint64  `json:"used"`
		Free  uint64  `json:"free"`
		Usage float64 `json:"usage"`
	} `json:"disk"`
	Network struct {
		BytesSent uint64 `json:"bytes_sent"`
		BytesRecv uint64 `json:"bytes_recv"`
	} `json:"network"`
}

type Collector struct {
	mu           sync.RWMutex
	logger       *zap.Logger
	metrics      SystemMetrics
	isCollecting bool
	stopChan     chan struct{}
}

func NewCollector(logger *zap.Logger) *Collector {
	return &Collector{
		logger:   logger,
		stopChan: make(chan struct{}),
	}
}

func (c *Collector) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.isCollecting {
		c.mu.Unlock()
		return nil
	}
	c.isCollecting = true
	c.mu.Unlock()

	go c.collectLoop(ctx)
	return nil
}

func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isCollecting {
		return
	}

	close(c.stopChan)
	c.isCollecting = false
}

func (c *Collector) GetMetrics() SystemMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metrics
}

func (c *Collector) collectLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collectSystemMetrics()
		}
	}
}

func (c *Collector) collectSystemMetrics() {
	var metrics SystemMetrics
	metrics.Timestamp = time.Now()

	// CPU metrics
	if cpuPercents, err := cpu.Percent(time.Second, false); err == nil && len(cpuPercents) > 0 {
		metrics.CPU.Usage = cpuPercents[0]
	}
	if cpuCounts, err := cpu.Counts(true); err == nil {
		metrics.CPU.Cores = cpuCounts
	}

	// Memory metrics
	if memStat, err := mem.VirtualMemory(); err == nil {
		metrics.Memory.Total = memStat.Total
		metrics.Memory.Used = memStat.Used
		metrics.Memory.Available = memStat.Available
		metrics.Memory.Usage = memStat.UsedPercent
	}

	// Disk metrics
	if diskStat, err := disk.Usage("/"); err == nil {
		metrics.Disk.Total = diskStat.Total
		metrics.Disk.Used = diskStat.Used
		metrics.Disk.Free = diskStat.Free
		metrics.Disk.Usage = diskStat.UsedPercent
	}

	// Network metrics
	if netStats, err := net.IOCounters(false); err == nil && len(netStats) > 0 {
		metrics.Network.BytesSent = netStats[0].BytesSent
		metrics.Network.BytesRecv = netStats[0].BytesRecv
	}

	c.mu.Lock()
	c.metrics = metrics
	c.mu.Unlock()
}

// CollectSystemMetrics returns current system metrics in the format expected by MetricsCollector interface
func (c *Collector) CollectSystemMetrics() models.SystemMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return models.SystemMetrics{
		Timestamp: c.metrics.Timestamp,
		CPU: models.CPUMetrics{
			UsagePercent: c.metrics.CPU.Usage,
			// Set other fields to 0 for now - could be enhanced later
		},
		Memory: models.MemoryMetrics{
			TotalBytes:     int64(c.metrics.Memory.Total),
			UsedBytes:      int64(c.metrics.Memory.Used),
			AvailableBytes: int64(c.metrics.Memory.Available),
			UsagePercent:   c.metrics.Memory.Usage,
		},
		Disk: models.DiskMetrics{
			UsagePercent: c.metrics.Disk.Usage,
			// Other disk metrics would need to be collected separately
		},
		Network: models.NetworkMetrics{
			RxBytesPerSec: int64(c.metrics.Network.BytesRecv),
			TxBytesPerSec: int64(c.metrics.Network.BytesSent),
		},
	}
}

// CollectPluginMetrics collects metrics from a specific plugin
func (c *Collector) CollectPluginMetrics(pluginName string, plugin plugins.StressPlugin) map[string]interface{} {
	metrics := make(map[string]interface{})

	// Basic plugin metrics
	metrics["plugin_name"] = pluginName
	metrics["timestamp"] = time.Now()

	// If plugin has specific metrics, collect them
	// This is a basic implementation - plugins could extend this
	metrics["status"] = "active"

	return metrics
}

// StartCollection starts metrics collection for a test
func (c *Collector) StartCollection(ctx context.Context, testID string) {
	c.logger.Info("Starting metrics collection", zap.String("test_id", testID))
	// Additional collection logic could be added here for test-specific metrics
}

// StopCollection stops metrics collection for a test
func (c *Collector) StopCollection(testID string) {
	c.logger.Info("Stopping metrics collection", zap.String("test_id", testID))
	// Additional cleanup logic could be added here
}
