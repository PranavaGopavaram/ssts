package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ExecutionStatus represents the status of a test execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusStopped   ExecutionStatus = "stopped"
)

// TestConfiguration represents a stress test configuration
type TestConfiguration struct {
	ID          string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name        string                 `json:"name" gorm:"not null"`
	Description string                 `json:"description"`
	Plugin      string                 `json:"plugin" gorm:"not null"`
	Config      json.RawMessage        `json:"config" gorm:"type:jsonb"`
	Duration    time.Duration          `json:"duration"`
	Safety      SafetyLimits          `json:"safety" gorm:"embedded"`
	Created     time.Time             `json:"created" gorm:"autoCreateTime"`
	Updated     time.Time             `json:"updated" gorm:"autoUpdateTime"`
	CreatedBy   string                `json:"created_by"`
}

// TestExecution represents a test execution instance
type TestExecution struct {
	ID           string            `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TestID       string            `json:"test_id" gorm:"type:uuid;not null"`
	Status       ExecutionStatus   `json:"status" gorm:"default:pending"`
	StartTime    *time.Time        `json:"start_time"`
	EndTime      *time.Time        `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	ExitCode     *int              `json:"exit_code"`
	ErrorMessage *string           `json:"error_message"`
	Summary      json.RawMessage   `json:"summary" gorm:"type:jsonb"`
	Created      time.Time         `json:"created" gorm:"autoCreateTime"`
}

// SafetyLimits defines resource usage limits for safety
type SafetyLimits struct {
	MaxCPUPercent    float64 `json:"max_cpu_percent" gorm:"column:max_cpu_percent"`
	MaxMemoryPercent float64 `json:"max_memory_percent" gorm:"column:max_memory_percent"`
	MaxDiskPercent   float64 `json:"max_disk_percent" gorm:"column:max_disk_percent"`
	MaxNetworkMbps   float64 `json:"max_network_mbps" gorm:"column:max_network_mbps"`
}

// DefaultSafetyLimits returns default safety limits
func DefaultSafetyLimits() SafetyLimits {
	return SafetyLimits{
		MaxCPUPercent:    80.0,
		MaxMemoryPercent: 70.0,
		MaxDiskPercent:   90.0,
		MaxNetworkMbps:   100.0,
	}
}

// TestParams defines parameters for test execution
type TestParams struct {
	Duration     time.Duration          `json:"duration"`
	Intensity    int                    `json:"intensity"` // 1-100 scale
	Concurrency  int                    `json:"concurrency"`
	CustomParams map[string]interface{} `json:"custom_params"`
}

// MetricPoint represents a single metric data point
type MetricPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	TestID    string                 `json:"test_id"`
	Source    string                 `json:"source"`
	Type      string                 `json:"type"`
	Tags      map[string]string      `json:"tags"`
	Fields    map[string]interface{} `json:"fields"`
}

// SystemMetrics represents overall system metrics
type SystemMetrics struct {
	Timestamp time.Time      `json:"timestamp"`
	CPU       CPUMetrics     `json:"cpu"`
	Memory    MemoryMetrics  `json:"memory"`
	Disk      DiskMetrics    `json:"disk"`
	Network   NetworkMetrics `json:"network"`
}

// CPUMetrics represents CPU-related metrics
type CPUMetrics struct {
	UsagePercent   float64   `json:"usage_percent"`
	UserPercent    float64   `json:"user_percent"`
	SystemPercent  float64   `json:"system_percent"`
	IdlePercent    float64   `json:"idle_percent"`
	IOWaitPercent  float64   `json:"iowait_percent"`
	FrequencyMHz   int64     `json:"frequency_mhz"`
	Temperature    float64   `json:"temperature_celsius"`
	CoreUsage      []float64 `json:"core_usage"`
}

// MemoryMetrics represents memory-related metrics
type MemoryMetrics struct {
	TotalBytes     int64   `json:"total_bytes"`
	UsedBytes      int64   `json:"used_bytes"`
	AvailableBytes int64   `json:"available_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	SwapUsedBytes  int64   `json:"swap_used_bytes"`
	CacheBytes     int64   `json:"cache_bytes"`
	BufferBytes    int64   `json:"buffer_bytes"`
}

// DiskMetrics represents disk I/O metrics
type DiskMetrics struct {
	ReadBytesPerSec  int64   `json:"read_bytes_per_sec"`
	WriteBytesPerSec int64   `json:"write_bytes_per_sec"`
	ReadOpsPerSec    int64   `json:"read_ops_per_sec"`
	WriteOpsPerSec   int64   `json:"write_ops_per_sec"`
	IOWaitPercent    float64 `json:"io_wait_percent"`
	QueueDepth       int64   `json:"queue_depth"`
	LatencyMs        float64 `json:"latency_ms"`
	UsagePercent     float64 `json:"usage_percent"`
}

// NetworkMetrics represents network-related metrics
type NetworkMetrics struct {
	RxBytesPerSec   int64   `json:"rx_bytes_per_sec"`
	TxBytesPerSec   int64   `json:"tx_bytes_per_sec"`
	RxPacketsPerSec int64   `json:"rx_packets_per_sec"`
	TxPacketsPerSec int64   `json:"tx_packets_per_sec"`
	RxErrors        int64   `json:"rx_errors"`
	TxErrors        int64   `json:"tx_errors"`
	LatencyMs       float64 `json:"latency_ms"`
}

// Plugin represents a stress test plugin
type Plugin struct {
	ID           string                 `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Name         string                 `json:"name" gorm:"unique;not null"`
	Version      string                 `json:"version"`
	Description  string                 `json:"description"`
	ConfigSchema json.RawMessage        `json:"config_schema" gorm:"type:jsonb"`
	SafetyLimits SafetyLimits          `json:"safety_limits" gorm:"embedded"`
	BinaryPath   string                 `json:"binary_path"`
	Checksum     string                 `json:"checksum"`
	InstalledAt  time.Time             `json:"installed_at" gorm:"autoCreateTime"`
	Enabled      bool                  `json:"enabled" gorm:"default:true"`
}

// User represents a system user
type User struct {
	ID           string          `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Username     string          `json:"username" gorm:"unique;not null"`
	Email        string          `json:"email" gorm:"unique;not null"`
	PasswordHash string          `json:"-" gorm:"not null"`
	Role         string          `json:"role" gorm:"default:user"`
	Preferences  json.RawMessage `json:"preferences" gorm:"type:jsonb"`
	Created      time.Time       `json:"created" gorm:"autoCreateTime"`
	LastLogin    *time.Time      `json:"last_login"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	TestID    string      `json:"test_id,omitempty"`
	Data      interface{} `json:"data"`
}

// TestResult represents aggregated test results
type TestResult struct {
	TestID        string                 `json:"test_id"`
	Status        ExecutionStatus        `json:"status"`
	Duration      time.Duration          `json:"duration"`
	Summary       map[string]interface{} `json:"summary"`
	Metrics       []MetricPoint          `json:"metrics"`
	Score         float64                `json:"score"`
	Passed        bool                   `json:"passed"`
	Errors        []string               `json:"errors,omitempty"`
}

// ExportRequest represents a data export request
type ExportRequest struct {
	TestID      string    `json:"test_id"`
	Format      string    `json:"format"`      // json, csv, pdf
	TimeRange   TimeRange `json:"time_range"`
	Metrics     []string  `json:"metrics"`
	Aggregation string    `json:"aggregation"` // raw, avg, max, min
}

// TimeRange represents a time range for queries
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// BeforeCreate hook for GORM to set UUID
func (t *TestConfiguration) BeforeCreate() {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
}

func (t *TestExecution) BeforeCreate() {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
}

func (p *Plugin) BeforeCreate() {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
}

func (u *User) BeforeCreate() {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
}