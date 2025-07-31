package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	InfluxDB InfluxDBConfig `mapstructure:"influxdb"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
	Safety   SafetyConfig   `mapstructure:"safety"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

// ServerConfig contains HTTP server configuration
type ServerConfig struct {
	Address      string        `mapstructure:"address"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	TLS          TLSConfig     `mapstructure:"tls"`
	CORS         CORSConfig    `mapstructure:"cors"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowOrigins []string `mapstructure:"allow_origins"`
	AllowMethods []string `mapstructure:"allow_methods"`
	AllowHeaders []string `mapstructure:"allow_headers"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	Type     string `mapstructure:"type"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

// InfluxDBConfig contains InfluxDB configuration
type InfluxDBConfig struct {
	URL    string `mapstructure:"url"`
	Token  string `mapstructure:"token"`
	Org    string `mapstructure:"org"`
	Bucket string `mapstructure:"bucket"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// LogConfig contains logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// SafetyConfig contains safety limits configuration
type SafetyConfig struct {
	GlobalLimits    GlobalLimits    `mapstructure:"global_limits"`
	Monitoring      MonitoringConfig `mapstructure:"monitoring"`
	RampUp          RampUpConfig    `mapstructure:"ramp_up"`
	EmergencyStop   bool           `mapstructure:"emergency_stop"`
}

// GlobalLimits contains global safety limits
type GlobalLimits struct {
	MaxCPUPercent             float64 `mapstructure:"max_cpu_percent"`
	MaxMemoryPercent          float64 `mapstructure:"max_memory_percent"`
	MaxDiskPercent            float64 `mapstructure:"max_disk_percent"`
	EmergencyStopThreshold    float64 `mapstructure:"emergency_stop_threshold"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	CheckInterval    time.Duration `mapstructure:"check_interval"`
	AlertThreshold   float64       `mapstructure:"alert_threshold"`
	AutoStopEnabled  bool          `mapstructure:"auto_stop_enabled"`
}

// RampUpConfig contains ramp-up configuration
type RampUpConfig struct {
	Enabled  bool          `mapstructure:"enabled"`
	Duration time.Duration `mapstructure:"duration"`
	Steps    int           `mapstructure:"steps"`
}

// AuthConfig contains authentication configuration
type AuthConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	JWTSecret     string        `mapstructure:"jwt_secret"`
	TokenExpiry   time.Duration `mapstructure:"token_expiry"`
	RefreshExpiry time.Duration `mapstructure:"refresh_expiry"`
}

// MetricsConfig contains metrics collection configuration
type MetricsConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	CollectionInterval time.Duration `mapstructure:"collection_interval"`
	BatchSize         int           `mapstructure:"batch_size"`
	FlushInterval     time.Duration `mapstructure:"flush_interval"`
	Retention         RetentionConfig `mapstructure:"retention"`
}

// RetentionConfig contains data retention configuration
type RetentionConfig struct {
	RealTime       time.Duration `mapstructure:"realtime"`
	HourlyAggr     time.Duration `mapstructure:"hourly_aggregates"`
	DailyAggr      time.Duration `mapstructure:"daily_aggregates"`
	Archive        time.Duration `mapstructure:"archive"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Address:      "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			TLS: TLSConfig{
				Enabled: false,
			},
			CORS: CORSConfig{
				AllowOrigins: []string{"*"},
				AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowHeaders: []string{"*"},
			},
		},
		Database: DatabaseConfig{
			Type:     "sqlite",
			Database: "./ssts.db",
			SSLMode:  "disable",
		},
		InfluxDB: InfluxDBConfig{
			URL:    "http://localhost:8086",
			Org:    "ssts",
			Bucket: "metrics",
		},
		Redis: RedisConfig{
			Address: "localhost:6379",
			DB:      0,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Safety: SafetyConfig{
			GlobalLimits: GlobalLimits{
				MaxCPUPercent:             80.0,
				MaxMemoryPercent:          70.0,
				MaxDiskPercent:            90.0,
				EmergencyStopThreshold:    95.0,
			},
			Monitoring: MonitoringConfig{
				CheckInterval:   1 * time.Second,
				AlertThreshold:  85.0,
				AutoStopEnabled: true,
			},
			RampUp: RampUpConfig{
				Enabled:  true,
				Duration: 30 * time.Second,
				Steps:    10,
			},
			EmergencyStop: true,
		},
		Auth: AuthConfig{
			Enabled:       false,
			TokenExpiry:   24 * time.Hour,
			RefreshExpiry: 7 * 24 * time.Hour,
		},
		Metrics: MetricsConfig{
			Enabled:            true,
			CollectionInterval: 1 * time.Second,
			BatchSize:          1000,
			FlushInterval:      5 * time.Second,
			Retention: RetentionConfig{
				RealTime:   24 * time.Hour,
				HourlyAggr: 30 * 24 * time.Hour,
				DailyAggr:  365 * 24 * time.Hour,
				Archive:    5 * 365 * 24 * time.Hour,
			},
		},
	}
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Set defaults
	setDefaults()

	// Load from file if exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	// Unmarshal to struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Safety.GlobalLimits.MaxCPUPercent < 1 || c.Safety.GlobalLimits.MaxCPUPercent > 100 {
		return fmt.Errorf("invalid max CPU percentage: %f", c.Safety.GlobalLimits.MaxCPUPercent)
	}

	if c.Safety.GlobalLimits.MaxMemoryPercent < 1 || c.Safety.GlobalLimits.MaxMemoryPercent > 100 {
		return fmt.Errorf("invalid max memory percentage: %f", c.Safety.GlobalLimits.MaxMemoryPercent)
	}

	return nil
}

// setDefaults sets default values for viper
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.address", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.read_timeout", "30s")
	viper.SetDefault("server.write_timeout", "30s")

	// Database defaults
	viper.SetDefault("database.type", "sqlite")
	viper.SetDefault("database.database", "./ssts.db")
	viper.SetDefault("database.ssl_mode", "disable")

	// InfluxDB defaults
	viper.SetDefault("influxdb.url", "http://localhost:8086")
	viper.SetDefault("influxdb.org", "ssts")
	viper.SetDefault("influxdb.bucket", "metrics")

	// Redis defaults
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.db", 0)

	// Logging defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.output", "stdout")

	// Safety defaults
	viper.SetDefault("safety.global_limits.max_cpu_percent", 80.0)
	viper.SetDefault("safety.global_limits.max_memory_percent", 70.0)
	viper.SetDefault("safety.global_limits.max_disk_percent", 90.0)
	viper.SetDefault("safety.global_limits.emergency_stop_threshold", 95.0)

	viper.SetDefault("safety.monitoring.check_interval", "1s")
	viper.SetDefault("safety.monitoring.alert_threshold", 85.0)
	viper.SetDefault("safety.monitoring.auto_stop_enabled", true)

	viper.SetDefault("safety.ramp_up.enabled", true)
	viper.SetDefault("safety.ramp_up.duration", "30s")
	viper.SetDefault("safety.ramp_up.steps", 10)
	viper.SetDefault("safety.emergency_stop", true)

	// Auth defaults
	viper.SetDefault("auth.enabled", false)
	viper.SetDefault("auth.token_expiry", "24h")
	viper.SetDefault("auth.refresh_expiry", "168h")

	// Metrics defaults
	viper.SetDefault("metrics.enabled", true)
	viper.SetDefault("metrics.collection_interval", "1s")
	viper.SetDefault("metrics.batch_size", 1000)
	viper.SetDefault("metrics.flush_interval", "5s")

	viper.SetDefault("metrics.retention.realtime", "24h")
	viper.SetDefault("metrics.retention.hourly_aggregates", "720h")
	viper.SetDefault("metrics.retention.daily_aggregates", "8760h")
	viper.SetDefault("metrics.retention.archive", "43800h")
}