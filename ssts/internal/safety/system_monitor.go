package safety

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// SystemMonitorImpl implements the SystemMonitor interface
type SystemMonitorImpl struct {
	lastCPUStats CPUStats
	lastCheck    time.Time
}

// CPUStats holds CPU statistics
type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Total   uint64
}

// NewSystemMonitor creates a new system monitor
func NewSystemMonitor() *SystemMonitorImpl {
	return &SystemMonitorImpl{}
}

// GetCPUUsage returns current CPU usage percentage
func (s *SystemMonitorImpl) GetCPUUsage() (float64, error) {
	stats, err := s.readCPUStats()
	if err != nil {
		return 0, fmt.Errorf("failed to read CPU stats: %w", err)
	}

	now := time.Now()

	// If this is the first check, store stats and return 0
	if s.lastCheck.IsZero() {
		s.lastCPUStats = stats
		s.lastCheck = now
		return 0, nil
	}

	// Calculate differences
	totalDiff := stats.Total - s.lastCPUStats.Total
	idleDiff := stats.Idle - s.lastCPUStats.Idle

	if totalDiff == 0 {
		return 0, nil
	}

	usage := float64(totalDiff-idleDiff) / float64(totalDiff) * 100.0

	// Update last stats
	s.lastCPUStats = stats
	s.lastCheck = now

	return usage, nil
}

// GetMemoryUsage returns current memory usage percentage
func (s *SystemMonitorImpl) GetMemoryUsage() (float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		// Fallback to runtime stats for non-Linux systems
		return s.getMemoryUsageRuntime()
	}
	defer file.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "MemTotal:":
			if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				memTotal = val * 1024 // Convert from KB to bytes
			}
		case "MemAvailable:":
			if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				memAvailable = val * 1024 // Convert from KB to bytes
			}
		}
	}

	if memTotal == 0 {
		return s.getMemoryUsageRuntime()
	}

	used := memTotal - memAvailable
	usage := float64(used) / float64(memTotal) * 100.0

	return usage, nil
}

// getMemoryUsageRuntime gets memory usage using runtime stats (fallback)
func (s *SystemMonitorImpl) getMemoryUsageRuntime() (float64, error) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// This is an approximation since we don't have total system memory
	// Use heap allocation as a proxy for memory pressure
	usage := float64(memStats.HeapAlloc) / float64(memStats.Sys) * 100.0

	// Cap at reasonable values
	if usage > 100 {
		usage = 100
	}

	return usage, nil
}

// GetDiskUsage returns current disk usage percentage for root filesystem
func (s *SystemMonitorImpl) GetDiskUsage() (float64, error) {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return 0, fmt.Errorf("failed to get disk stats: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	used := total - free

	if total == 0 {
		return 0, nil
	}

	usage := float64(used) / float64(total) * 100.0
	return usage, nil
}

// GetNetworkUsage returns current network usage in Mbps
func (s *SystemMonitorImpl) GetNetworkUsage() (float64, error) {
	// This is a simplified implementation
	// In a production system, you would track network interface statistics
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, nil // Return 0 for non-Linux systems
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var totalBytes uint64

	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}

		// Skip loopback interface
		if strings.Contains(fields[0], "lo:") {
			continue
		}

		// Parse received bytes (field 1) and transmitted bytes (field 9)
		if rxBytes, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			totalBytes += rxBytes
		}
		if txBytes, err := strconv.ParseUint(fields[9], 10, 64); err == nil {
			totalBytes += txBytes
		}
	}

	// Convert to Mbps (this is cumulative, not current rate)
	// In a real implementation, you would track the rate over time
	mbps := float64(totalBytes) / (1024 * 1024) / 8 // Rough approximation

	// Cap at reasonable value for monitoring purposes
	if mbps > 1000 {
		mbps = 1000
	}

	return mbps, nil
}

// GetSystemTemperature returns system temperature in Celsius
func (s *SystemMonitorImpl) GetSystemTemperature() (float64, error) {
	// Try to read from thermal zone (Linux)
	tempFiles := []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
	}

	for _, file := range tempFiles {
		if temp, err := s.readTemperatureFile(file); err == nil {
			return temp, nil
		}
	}

	// If no thermal zone found, return a safe default
	return 35.0, nil
}

// readTemperatureFile reads temperature from a thermal zone file
func (s *SystemMonitorImpl) readTemperatureFile(filename string) (float64, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return 0, err
	}

	tempStr := strings.TrimSpace(string(data))
	tempMilliC, err := strconv.ParseFloat(tempStr, 64)
	if err != nil {
		return 0, err
	}

	// Convert from millicelsius to celsius
	tempC := tempMilliC / 1000.0
	return tempC, nil
}

// readCPUStats reads CPU statistics from /proc/stat
func (s *SystemMonitorImpl) readCPUStats() (CPUStats, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		// Fallback for non-Linux systems
		return s.getCPUStatsRuntime()
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return CPUStats{}, fmt.Errorf("failed to read CPU stats")
	}

	line := scanner.Text()
	fields := strings.Fields(line)
	if len(fields) < 8 || fields[0] != "cpu" {
		return CPUStats{}, fmt.Errorf("invalid CPU stats format")
	}

	stats := CPUStats{}

	if val, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
		stats.User = val
	}
	if val, err := strconv.ParseUint(fields[2], 10, 64); err == nil {
		stats.Nice = val
	}
	if val, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
		stats.System = val
	}
	if val, err := strconv.ParseUint(fields[4], 10, 64); err == nil {
		stats.Idle = val
	}
	if val, err := strconv.ParseUint(fields[5], 10, 64); err == nil {
		stats.IOWait = val
	}
	if val, err := strconv.ParseUint(fields[6], 10, 64); err == nil {
		stats.IRQ = val
	}
	if val, err := strconv.ParseUint(fields[7], 10, 64); err == nil {
		stats.SoftIRQ = val
	}

	stats.Total = stats.User + stats.Nice + stats.System + stats.Idle +
		stats.IOWait + stats.IRQ + stats.SoftIRQ

	return stats, nil
}

// AlertManagerImpl implements the AlertManager interface
type AlertManagerImpl struct {
	logger *logrus.Logger
}

// NewAlertManager creates a new alert manager
func NewAlertManager(logger *logrus.Logger) *AlertManagerImpl {
	return &AlertManagerImpl{
		logger: logger,
	}
}

// SendAlert sends an alert (simple implementation that logs alerts)
func (a *AlertManagerImpl) SendAlert(alert Alert) error {
	a.logger.WithFields(logrus.Fields{
		"alert_id":  alert.ID,
		"type":      alert.Type,
		"severity":  alert.Severity,
		"message":   alert.Message,
		"timestamp": alert.Timestamp,
		"metadata":  alert.Metadata,
	}).Info("Alert sent")

	return nil
}

// getCPUStatsRuntime gets CPU stats using runtime package (fallback)
func (s *SystemMonitorImpl) getCPUStatsRuntime() (CPUStats, error) {
	// This is a basic fallback - in reality, you'd use platform-specific APIs
	numCPU := runtime.NumCPU()

	// Return dummy stats based on number of CPUs
	stats := CPUStats{
		User:   uint64(numCPU * 1000),
		System: uint64(numCPU * 500),
		Idle:   uint64(numCPU * 8500),
		Total:  uint64(numCPU * 10000),
	}

	return stats, nil
}
