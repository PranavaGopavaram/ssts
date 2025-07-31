package safety

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/pranavgopavaram/ssts/pkg/models"
	"github.com/sirupsen/logrus"
)

// Monitor provides safety monitoring and enforcement
type Monitor struct {
	systemMonitor  SystemMonitor
	alertManager   AlertManager
	config         Config
	emergencyStop  chan string
	violations     []Violation
	mu             sync.RWMutex
	logger         *logrus.Logger
}

// Config defines safety monitor configuration
type Config struct {
	CheckInterval        time.Duration `yaml:"check_interval"`
	AlertThreshold       float64       `yaml:"alert_threshold"`
	EmergencyThreshold   float64       `yaml:"emergency_threshold"`
	AutoStopEnabled      bool          `yaml:"auto_stop_enabled"`
	RampUpEnabled        bool          `yaml:"ramp_up_enabled"`
	RampUpDuration       time.Duration `yaml:"ramp_up_duration"`
	RampUpSteps          int           `yaml:"ramp_up_steps"`
	CooldownPeriod       time.Duration `yaml:"cooldown_period"`
	MaxViolationsPerMin  int           `yaml:"max_violations_per_min"`
}

// SystemMonitor interface for system monitoring
type SystemMonitor interface {
	GetCPUUsage() (float64, error)
	GetMemoryUsage() (float64, error)
	GetDiskUsage() (float64, error)
	GetNetworkUsage() (float64, error)
	GetSystemTemperature() (float64, error)
}

// AlertManager interface for alert management
type AlertManager interface {
	SendAlert(alert Alert) error
}

// Violation represents a safety limit violation
type Violation struct {
	Type         string    `json:"type"`
	CurrentValue float64   `json:"current_value"`
	Limit        float64   `json:"limit"`
	Severity     Severity  `json:"severity"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Critical     bool      `json:"critical"`
}

// Severity levels for violations
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Alert represents a safety alert
type Alert struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Message   string                 `json:"message"`
	Severity  Severity               `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// NewMonitor creates a new safety monitor
func NewMonitor(systemMonitor SystemMonitor, alertManager AlertManager, config Config, logger *logrus.Logger) *Monitor {
	if config.CheckInterval == 0 {
		config.CheckInterval = 1 * time.Second
	}
	if config.AlertThreshold == 0 {
		config.AlertThreshold = 85.0
	}
	if config.EmergencyThreshold == 0 {
		config.EmergencyThreshold = 95.0
	}
	if config.RampUpDuration == 0 {
		config.RampUpDuration = 30 * time.Second
	}
	if config.RampUpSteps == 0 {
		config.RampUpSteps = 10
	}
	if config.CooldownPeriod == 0 {
		config.CooldownPeriod = 60 * time.Second
	}
	if config.MaxViolationsPerMin == 0 {
		config.MaxViolationsPerMin = 5
	}

	return &Monitor{
		systemMonitor: systemMonitor,
		alertManager:  alertManager,
		config:        config,
		emergencyStop: make(chan string, 10),
		violations:    make([]Violation, 0),
		logger:        logger,
	}
}

// Start starts the safety monitoring
func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.performSafetyCheck()
		}
	}
}

// CheckSafetyLimits checks if current system state violates safety limits
func (m *Monitor) CheckSafetyLimits(limits models.SafetyLimits) *Violation {
	// Check CPU usage
	if cpuUsage, err := m.systemMonitor.GetCPUUsage(); err == nil {
		if cpuUsage > limits.MaxCPUPercent {
			violation := &Violation{
				Type:         "cpu",
				CurrentValue: cpuUsage,
				Limit:        limits.MaxCPUPercent,
				Message:      fmt.Sprintf("CPU usage %.1f%% exceeds limit %.1f%%", cpuUsage, limits.MaxCPUPercent),
				Timestamp:    time.Now(),
				Critical:     cpuUsage > m.config.EmergencyThreshold,
			}
			
			if cpuUsage > m.config.EmergencyThreshold {
				violation.Severity = SeverityCritical
			} else if cpuUsage > m.config.AlertThreshold {
				violation.Severity = SeverityError
			} else {
				violation.Severity = SeverityWarning
			}

			m.recordViolation(*violation)
			return violation
		}
	}

	// Check memory usage
	if memUsage, err := m.systemMonitor.GetMemoryUsage(); err == nil {
		if memUsage > limits.MaxMemoryPercent {
			violation := &Violation{
				Type:         "memory",
				CurrentValue: memUsage,
				Limit:        limits.MaxMemoryPercent,
				Message:      fmt.Sprintf("Memory usage %.1f%% exceeds limit %.1f%%", memUsage, limits.MaxMemoryPercent),
				Timestamp:    time.Now(),
				Critical:     memUsage > m.config.EmergencyThreshold,
			}

			if memUsage > m.config.EmergencyThreshold {
				violation.Severity = SeverityCritical
			} else if memUsage > m.config.AlertThreshold {
				violation.Severity = SeverityError
			} else {
				violation.Severity = SeverityWarning
			}

			m.recordViolation(*violation)
			return violation
		}
	}

	// Check disk usage
	if diskUsage, err := m.systemMonitor.GetDiskUsage(); err == nil {
		if diskUsage > limits.MaxDiskPercent {
			violation := &Violation{
				Type:         "disk",
				CurrentValue: diskUsage,
				Limit:        limits.MaxDiskPercent,
				Message:      fmt.Sprintf("Disk usage %.1f%% exceeds limit %.1f%%", diskUsage, limits.MaxDiskPercent),
				Timestamp:    time.Now(),
				Critical:     diskUsage > m.config.EmergencyThreshold,
			}

			if diskUsage > m.config.EmergencyThreshold {
				violation.Severity = SeverityCritical
			} else if diskUsage > m.config.AlertThreshold {
				violation.Severity = SeverityError
			} else {
				violation.Severity = SeverityWarning
			}

			m.recordViolation(*violation)
			return violation
		}
	}

	// Check network usage
	if netUsage, err := m.systemMonitor.GetNetworkUsage(); err == nil {
		if netUsage > limits.MaxNetworkMbps {
			violation := &Violation{
				Type:         "network",
				CurrentValue: netUsage,
				Limit:        limits.MaxNetworkMbps,
				Message:      fmt.Sprintf("Network usage %.1f Mbps exceeds limit %.1f Mbps", netUsage, limits.MaxNetworkMbps),
				Timestamp:    time.Now(),
				Critical:     false, // Network usage rarely critical
			}

			if netUsage > limits.MaxNetworkMbps*2 {
				violation.Severity = SeverityError
			} else {
				violation.Severity = SeverityWarning
			}

			m.recordViolation(*violation)
			return violation
		}
	}

	return nil
}

// performSafetyCheck performs a comprehensive safety check
func (m *Monitor) performSafetyCheck() {
	// Check system health
	if temp, err := m.systemMonitor.GetSystemTemperature(); err == nil {
		if temp > 85.0 { // High temperature threshold
			violation := Violation{
				Type:         "temperature",
				CurrentValue: temp,
				Limit:        85.0,
				Message:      fmt.Sprintf("System temperature %.1f°C is too high", temp),
				Timestamp:    time.Now(),
				Severity:     SeverityCritical,
				Critical:     temp > 90.0,
			}

			m.recordViolation(violation)

			if violation.Critical {
				m.sendEmergencyStop(fmt.Sprintf("Critical temperature: %.1f°C", temp))
			}
		}
	}

	// Check violation rate
	recentViolations := m.getRecentViolations(1 * time.Minute)
	if len(recentViolations) > m.config.MaxViolationsPerMin {
		m.sendEmergencyStop(fmt.Sprintf("Too many violations: %d in last minute", len(recentViolations)))
	}

	// Check memory pressure
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	if memStats.Sys > 2*1024*1024*1024 { // 2GB threshold
		if memStats.HeapAlloc > memStats.Sys/2 {
			violation := Violation{
				Type:         "memory_pressure",
				CurrentValue: float64(memStats.HeapAlloc) / float64(memStats.Sys) * 100,
				Limit:        50.0,
				Message:      "High memory pressure detected",
				Timestamp:    time.Now(),
				Severity:     SeverityWarning,
				Critical:     false,
			}

			m.recordViolation(violation)
		}
	}
}

// recordViolation records a safety violation
func (m *Monitor) recordViolation(violation Violation) {
	m.mu.Lock()
	m.violations = append(m.violations, violation)
	
	// Keep only recent violations (last hour)
	cutoff := time.Now().Add(-1 * time.Hour)
	filtered := m.violations[:0]
	for _, v := range m.violations {
		if v.Timestamp.After(cutoff) {
			filtered = append(filtered, v)
		}
	}
	m.violations = filtered
	m.mu.Unlock()

	// Send alert
	alert := Alert{
		Type:      violation.Type,
		Message:   violation.Message,
		Severity:  violation.Severity,
		Timestamp: violation.Timestamp,
		Metadata: map[string]interface{}{
			"current_value": violation.CurrentValue,
			"limit":         violation.Limit,
			"critical":      violation.Critical,
		},
	}

	if err := m.alertManager.SendAlert(alert); err != nil {
		m.logger.WithError(err).Error("Failed to send alert")
	}

	m.logger.WithFields(logrus.Fields{
		"type":          violation.Type,
		"current_value": violation.CurrentValue,
		"limit":         violation.Limit,
		"severity":      violation.Severity,
	}).Warn("Safety violation recorded")
}

// getRecentViolations returns violations within the specified duration
func (m *Monitor) getRecentViolations(duration time.Duration) []Violation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	recent := make([]Violation, 0)

	for _, violation := range m.violations {
		if violation.Timestamp.After(cutoff) {
			recent = append(recent, violation)
		}
	}

	return recent
}

// sendEmergencyStop sends an emergency stop signal
func (m *Monitor) sendEmergencyStop(reason string) {
	select {
	case m.emergencyStop <- reason:
		m.logger.WithField("reason", reason).Error("Emergency stop triggered")
	default:
		m.logger.Warn("Emergency stop channel full, dropping signal")
	}
}

// GetEmergencyStopChannel returns the emergency stop channel
func (m *Monitor) GetEmergencyStopChannel() <-chan string {
	return m.emergencyStop
}

// GetViolations returns recent violations
func (m *Monitor) GetViolations() []Violation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	violations := make([]Violation, len(m.violations))
	copy(violations, m.violations)
	return violations
}

// GetSafetyStatus returns current safety status
func (m *Monitor) GetSafetyStatus() SafetyStatus {
	recentViolations := m.getRecentViolations(5 * time.Minute)
	
	status := SafetyStatus{
		Overall:           "healthy",
		RecentViolations:  len(recentViolations),
		LastViolation:     nil,
		SystemHealth:      m.getSystemHealth(),
		Timestamp:         time.Now(),
	}

	if len(recentViolations) > 0 {
		status.LastViolation = &recentViolations[len(recentViolations)-1]
		
		if len(recentViolations) > 3 {
			status.Overall = "degraded"
		} else {
			status.Overall = "warning"
		}
	}

	// Check for critical violations
	for _, violation := range recentViolations {
		if violation.Critical {
			status.Overall = "critical"
			break
		}
	}

	return status
}

// SafetyStatus represents the current safety status
type SafetyStatus struct {
	Overall          string      `json:"overall"`
	RecentViolations int         `json:"recent_violations"`
	LastViolation    *Violation  `json:"last_violation,omitempty"`
	SystemHealth     SystemHealth `json:"system_health"`
	Timestamp        time.Time   `json:"timestamp"`
}

// SystemHealth represents system health metrics
type SystemHealth struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Temperature float64 `json:"temperature"`
}

// getSystemHealth gets current system health metrics
func (m *Monitor) getSystemHealth() SystemHealth {
	health := SystemHealth{}

	if cpu, err := m.systemMonitor.GetCPUUsage(); err == nil {
		health.CPUUsage = cpu
	}

	if mem, err := m.systemMonitor.GetMemoryUsage(); err == nil {
		health.MemoryUsage = mem
	}

	if disk, err := m.systemMonitor.GetDiskUsage(); err == nil {
		health.DiskUsage = disk
	}

	if temp, err := m.systemMonitor.GetSystemTemperature(); err == nil {
		health.Temperature = temp
	}

	return health
}

// CalculateRampUpIntensity calculates intensity for ramp-up phase
func (m *Monitor) CalculateRampUpIntensity(elapsed time.Duration, targetIntensity int) int {
	if !m.config.RampUpEnabled || elapsed >= m.config.RampUpDuration {
		return targetIntensity
	}

	progress := float64(elapsed) / float64(m.config.RampUpDuration)
	stepSize := float64(targetIntensity) / float64(m.config.RampUpSteps)
	currentStep := int(progress * float64(m.config.RampUpSteps))
	
	intensity := int(float64(currentStep) * stepSize)
	if intensity > targetIntensity {
		intensity = targetIntensity
	}

	return intensity
}

// IsInCooldownPeriod checks if system is in cooldown period after a violation
func (m *Monitor) IsInCooldownPeriod() bool {
	recentViolations := m.getRecentViolations(m.config.CooldownPeriod)
	
	for _, violation := range recentViolations {
		if violation.Severity == SeverityError || violation.Severity == SeverityCritical {
			return true
		}
	}

	return false
}