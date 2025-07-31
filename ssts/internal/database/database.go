package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/pranavgopavaram/ssts/internal/config"
	"github.com/pranavgopavaram/ssts/pkg/models"
)

// Database wraps GORM database connection
type Database struct {
	*gorm.DB
}

// Initialize initializes the database connection and performs migrations
func Initialize(cfg config.DatabaseConfig) (*Database, error) {
	var db *gorm.DB
	var err error

	// Configure GORM logger
	logLevel := logger.Silent
	if cfg.Type == "sqlite" {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	// Connect based on database type
	switch cfg.Type {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.Username, cfg.Password, cfg.Database, cfg.SSLMode)
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.Database), gormConfig)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool for postgres
	if cfg.Type == "postgres" || cfg.Type == "postgresql" {
		sqlDB, err := db.DB()
		if err != nil {
			return nil, fmt.Errorf("failed to get sql.DB: %w", err)
		}

		// Connection pool settings
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	// Auto-migrate schemas
	if err := runMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &Database{DB: db}, nil
}

// runMigrations performs database schema migrations
func runMigrations(db *gorm.DB) error {
	// Create extensions for PostgreSQL
	if db.Dialector.Name() == "postgres" {
		db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")
		db.Exec("CREATE EXTENSION IF NOT EXISTS \"pgcrypto\"")
	}

	// Auto-migrate all models
	models := []interface{}{
		&models.User{},
		&models.Plugin{},
		&models.TestConfiguration{},
		&models.TestExecution{},
	}

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Create indexes
	if err := createIndexes(db); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// createIndexes creates database indexes for performance
func createIndexes(db *gorm.DB) error {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_test_executions_status ON test_executions(status)",
		"CREATE INDEX IF NOT EXISTS idx_test_executions_start_time ON test_executions(start_time)",
		"CREATE INDEX IF NOT EXISTS idx_test_configurations_plugin ON test_configurations(plugin)",
		"CREATE INDEX IF NOT EXISTS idx_test_executions_test_id ON test_executions(test_id)",
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_plugins_name ON plugins(name)",
	}

	for _, index := range indexes {
		if err := db.Exec(index).Error; err != nil {
			// Log warning but don't fail - some indexes might already exist
			fmt.Printf("Warning: failed to create index: %v\n", err)
		}
	}

	return nil
}

// Close closes the database connection
func (db *Database) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck performs a database health check
func (db *Database) HealthCheck() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

// Repository provides data access methods
type Repository struct {
	db *Database
}

// NewRepository creates a new repository
func NewRepository(db *Database) *Repository {
	return &Repository{db: db}
}

// Users repository methods
func (r *Repository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}

// Test configurations repository methods
func (r *Repository) CreateTestConfiguration(config *models.TestConfiguration) error {
	return r.db.Create(config).Error
}

func (r *Repository) GetTestConfiguration(id string) (*models.TestConfiguration, error) {
	var config models.TestConfiguration
	err := r.db.Where("id = ?", id).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *Repository) ListTestConfigurations(limit, offset int) ([]models.TestConfiguration, error) {
	var configs []models.TestConfiguration
	err := r.db.Limit(limit).Offset(offset).Order("created DESC").Find(&configs).Error
	return configs, err
}

func (r *Repository) UpdateTestConfiguration(config *models.TestConfiguration) error {
	return r.db.Save(config).Error
}

func (r *Repository) DeleteTestConfiguration(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.TestConfiguration{}).Error
}

// Test executions repository methods
func (r *Repository) CreateTestExecution(execution *models.TestExecution) error {
	return r.db.Create(execution).Error
}

func (r *Repository) GetTestExecution(id string) (*models.TestExecution, error) {
	var execution models.TestExecution
	err := r.db.Where("id = ?", id).First(&execution).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

func (r *Repository) ListTestExecutions(limit, offset int) ([]models.TestExecution, error) {
	var executions []models.TestExecution
	err := r.db.Limit(limit).Offset(offset).Order("created DESC").Find(&executions).Error
	return executions, err
}

func (r *Repository) ListTestExecutionsByStatus(status models.ExecutionStatus, limit, offset int) ([]models.TestExecution, error) {
	var executions []models.TestExecution
	err := r.db.Where("status = ?", status).Limit(limit).Offset(offset).Order("created DESC").Find(&executions).Error
	return executions, err
}

func (r *Repository) UpdateTestExecution(execution *models.TestExecution) error {
	return r.db.Save(execution).Error
}

func (r *Repository) DeleteTestExecution(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.TestExecution{}).Error
}

// Plugin repository methods
func (r *Repository) CreatePlugin(plugin *models.Plugin) error {
	return r.db.Create(plugin).Error
}

func (r *Repository) GetPlugin(name string) (*models.Plugin, error) {
	var plugin models.Plugin
	err := r.db.Where("name = ?", name).First(&plugin).Error
	if err != nil {
		return nil, err
	}
	return &plugin, nil
}

func (r *Repository) ListPlugins() ([]models.Plugin, error) {
	var plugins []models.Plugin
	err := r.db.Where("enabled = ?", true).Order("name").Find(&plugins).Error
	return plugins, err
}

func (r *Repository) UpdatePlugin(plugin *models.Plugin) error {
	return r.db.Save(plugin).Error
}

func (r *Repository) DeletePlugin(name string) error {
	return r.db.Where("name = ?", name).Delete(&models.Plugin{}).Error
}