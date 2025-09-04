package main

import (
	"context"

	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/seasbee/go-ormx/pkg/logging"
	"github.com/seasbee/go-ormx/pkg/models"
	"github.com/seasbee/go-ormx/pkg/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ExampleUser represents a user entity for demonstration
type ExampleUser struct {
	models.BaseModel
	Username string `gorm:"uniqueIndex;not null" json:"username" validate:"required,min=3,max=50"`
	Email    string `gorm:"uniqueIndex;not null" json:"email" validate:"required,email"`
	FullName string `gorm:"not null" json:"full_name" validate:"required,min=2,max=100"`
	IsActive bool   `gorm:"default:true;index" json:"is_active"`
}

// TableName returns the table name for ExampleUser
func (ExampleUser) TableName() string {
	return "example_users"
}

// UserRepository provides user-specific repository operations
type UserRepository struct {
	*repository.BaseRepository[ExampleUser]
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB, logger logging.Logger) *UserRepository {
	config := repository.DefaultRepositoryConfig()
	baseRepo := repository.NewBaseRepository[ExampleUser](db, logger, config)
	return &UserRepository{
		BaseRepository: baseRepo,
	}
}

// FindByUsername finds a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*ExampleUser, error) {
	var user ExampleUser
	if err := r.FindFirstByConditions(ctx, &user, "username = ?", username); err != nil {
		return nil, fmt.Errorf("failed to find user by username: %w", err)
	}
	return &user, nil
}

// FindByEmail finds a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*ExampleUser, error) {
	var user ExampleUser
	if err := r.FindFirstByConditions(ctx, &user, "email = ?", email); err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return &user, nil
}

// FindActiveUsers finds all active users
func (r *UserRepository) FindActiveUsers(ctx context.Context) ([]ExampleUser, error) {
	var users []ExampleUser
	if err := r.FindAllByConditionsWithOffset(ctx, 10, 0, &users, "is_active = ?", true); err != nil {
		return nil, fmt.Errorf("failed to find active users: %w", err)
	}
	return users, nil
}

// UpdateUserStatus updates user active status
func (r *UserRepository) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	user := &ExampleUser{
		IsActive: isActive,
	}
	return r.UpdateByID(ctx, user, userID)
}

// ConnectionManager demonstrates advanced connection management
type ConnectionManager struct {
	config         *DatabaseConfig
	primaryDB      *gorm.DB
	connectionPool *ConnectionPool
	shutdownChan   chan os.Signal
	shutdownWg     sync.WaitGroup
	metrics        *ConnectionMetrics
	healthChecker  *HealthChecker
	userRepo       *UserRepository
	logger         logging.Logger
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	SSLMode         string
	MaxConnections  int
	MinConnections  int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// ConnectionPool manages connection lifecycle
type ConnectionPool struct {
	maxConnections int
	minConnections int
	connections    chan *gorm.DB
	mu             sync.RWMutex
}

// ConnectionMetrics tracks connection performance metrics
type ConnectionMetrics struct {
	ActiveConnections int64
	TotalConnections  int64
	FailedConnections int64
	mu                sync.RWMutex
}

// HealthChecker monitors database health
type HealthChecker struct {
	interval     time.Duration
	timeout      time.Duration
	stopChan     chan struct{}
	healthStatus bool
	mu           sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	// Create logger
	logger := logging.NewLogger(logging.LogLevelInfo, os.Stdout, &logging.TextFormatter{})

	return &ConnectionManager{
		shutdownChan: make(chan os.Signal, 1),
		metrics:      &ConnectionMetrics{},
		healthChecker: &HealthChecker{
			interval: 30 * time.Second,
			timeout:  5 * time.Second,
			stopChan: make(chan struct{}),
		},
		connectionPool: &ConnectionPool{
			maxConnections: 100,
			minConnections: 10,
		},
		logger: logger,
	}
}

// Initialize sets up the connection manager with configuration
func (cm *ConnectionManager) Initialize() error {
	// Load configuration
	if err := cm.loadConfiguration(); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize primary connection
	if err := cm.initializePrimaryConnection(); err != nil {
		return fmt.Errorf("failed to initialize primary connection: %w", err)
	}

	// Initialize connection pool
	if err := cm.initializeConnectionPool(); err != nil {
		return fmt.Errorf("failed to initialize connection pool: %w", err)
	}

	// Start background services
	cm.startBackgroundServices()

	return nil
}

// loadConfiguration loads database configuration
func (cm *ConnectionManager) loadConfiguration() error {
	cm.config = &DatabaseConfig{
		Host:            getEnvOrDefault("DB_HOST", "localhost"),
		Port:            getEnvOrDefaultInt("DB_PORT", 5432),
		Username:        getEnvOrDefault("DB_USERNAME", "postgres"),
		Password:        getEnvOrDefault("DB_PASSWORD", "password"),
		Database:        getEnvOrDefault("DB_NAME", "example_db"),
		SSLMode:         getEnvOrDefault("DB_SSL_MODE", "disable"),
		MaxConnections:  100,
		MinConnections:  10,
		MaxIdleConns:    25,
		ConnMaxLifetime: 1 * time.Hour,
		ConnMaxIdleTime: 5 * time.Minute,
	}
	return nil
}

// initializePrimaryConnection sets up the primary database connection
func (cm *ConnectionManager) initializePrimaryConnection() error {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cm.config.Host, cm.config.Port, cm.config.Username, cm.config.Password, cm.config.Database, cm.config.SSLMode)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cm.config.MaxConnections)
	sqlDB.SetMaxIdleConns(cm.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cm.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cm.config.ConnMaxIdleTime)

	cm.primaryDB = db

	// Auto-migrate models
	if err := cm.primaryDB.AutoMigrate(&ExampleUser{}); err != nil {
		return fmt.Errorf("failed to migrate models: %w", err)
	}

	// Initialize user repository
	cm.userRepo = NewUserRepository(cm.primaryDB, cm.logger)

	cm.logger.Info(context.Background(), "Primary connection initialized", logging.Int("max_connections", cm.config.MaxConnections))
	return nil
}

// initializeConnectionPool sets up the connection pool
func (cm *ConnectionManager) initializeConnectionPool() error {
	cm.connectionPool.connections = make(chan *gorm.DB, cm.connectionPool.maxConnections)

	// Pre-populate pool with minimum connections
	for i := 0; i < cm.connectionPool.minConnections; i++ {
		conn, err := cm.createDatabaseConnection()
		if err != nil {
			log.Printf("Failed to create connection for pool: %v", err)
			continue
		}
		cm.connectionPool.connections <- conn
		cm.metrics.incrementTotalConnections()
	}

	cm.logger.Info(context.Background(), "Connection pool initialized",
		logging.Int("min_connections", cm.connectionPool.minConnections),
		logging.Int("max_connections", cm.connectionPool.maxConnections))
	return nil
}

// createDatabaseConnection creates a new GORM database connection
func (cm *ConnectionManager) createDatabaseConnection() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cm.config.Host, cm.config.Port, cm.config.Username, cm.config.Password, cm.config.Database, cm.config.SSLMode)

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cm.config.MaxConnections)
	sqlDB.SetMaxIdleConns(cm.config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cm.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cm.config.ConnMaxIdleTime)

	return db, nil
}

// startBackgroundServices starts health checking and metrics collection
func (cm *ConnectionManager) startBackgroundServices() {
	// Start health checker
	cm.shutdownWg.Add(1)
	go cm.healthChecker.start(cm, &cm.shutdownWg)

	// Start metrics collection
	cm.shutdownWg.Add(1)
	go cm.collectMetrics(&cm.shutdownWg)

	cm.logger.Info(context.Background(), "Background services started")
}

// Run starts the connection manager
func (cm *ConnectionManager) Run() error {
	// Set up graceful shutdown
	signal.Notify(cm.shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	cm.logger.Info(context.Background(), "Connection manager started")

	// Run demonstration scenarios
	if err := cm.runDemonstrationScenarios(); err != nil {
		cm.logger.Error(context.Background(), "Demonstration scenarios failed", logging.ErrorField("error", err))
	}

	// Wait for shutdown signal
	<-cm.shutdownChan
	cm.logger.Info(context.Background(), "Shutdown signal received, cleaning up...")

	return cm.Shutdown()
}

// runDemonstrationScenarios runs various demonstration scenarios
func (cm *ConnectionManager) runDemonstrationScenarios() error {
	ctx := context.Background()

	// Scenario 1: Basic CRUD operations
	if err := cm.demonstrateBasicCRUD(ctx); err != nil {
		return fmt.Errorf("basic CRUD demonstration failed: %w", err)
	}

	// Scenario 2: High concurrency operations
	if err := cm.demonstrateHighConcurrency(ctx); err != nil {
		return fmt.Errorf("high concurrency demonstration failed: %w", err)
	}

	// Scenario 3: Connection pooling
	if err := cm.demonstrateConnectionPooling(ctx); err != nil {
		return fmt.Errorf("connection pooling demonstration failed: %w", err)
	}

	// Scenario 4: Advanced repository operations
	if err := cm.demonstrateAdvancedRepositoryOperations(ctx); err != nil {
		return fmt.Errorf("advanced repository operations demonstration failed: %w", err)
	}

	return nil
}

// demonstrateBasicCRUD demonstrates basic CRUD operations
func (cm *ConnectionManager) demonstrateBasicCRUD(ctx context.Context) error {
	cm.logger.Info(ctx, "Starting basic CRUD demonstration")

	// Create user
	user := &ExampleUser{
		Username: "john_doe",
		Email:    "john@example.com",
		FullName: "John Doe",
		IsActive: true,
	}

	if err := cm.userRepo.Create(ctx, user); err != nil {
		cm.logger.Error(ctx, "Failed to create user", logging.ErrorField("error", err))
		return fmt.Errorf("failed to create user: %w", err)
	}

	cm.logger.Info(ctx, "User created successfully",
		logging.String("id", user.ID.String()),
		logging.String("username", user.Username))

	// Read operations - find by ID
	foundUser, err := cm.userRepo.FindFirstByID(ctx, user.ID)
	if err != nil {
		cm.logger.Error(ctx, "Failed to find user by ID", logging.ErrorField("error", err))
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Read operations - find by username
	foundByUsername, err := cm.userRepo.FindByUsername(ctx, "john_doe")
	if err != nil {
		cm.logger.Error(ctx, "Failed to find user by username", logging.ErrorField("error", err))
		return fmt.Errorf("failed to find user by username: %w", err)
	}

	// Verify the found user matches
	if foundByUsername.ID != foundUser.ID {
		cm.logger.Warn(ctx, "User ID mismatch between different find methods")
	}

	// Update operations
	foundUser.FullName = "John Smith"
	if err := cm.userRepo.Update(ctx, foundUser); err != nil {
		cm.logger.Error(ctx, "Failed to update user", logging.ErrorField("error", err))
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Query operations - find all active users
	activeUsers, err := cm.userRepo.FindActiveUsers(ctx)
	if err != nil {
		cm.logger.Error(ctx, "Failed to find active users", logging.ErrorField("error", err))
		return fmt.Errorf("failed to find active users: %w", err)
	}

	// Count operations
	totalCount, err := cm.userRepo.CountAll(ctx)
	if err != nil {
		cm.logger.Error(ctx, "Failed to count users", logging.ErrorField("error", err))
		return fmt.Errorf("failed to count users: %w", err)
	}

	cm.logger.Info(ctx, "Basic CRUD demonstration completed",
		logging.Int("active_users_count", len(activeUsers)),
		logging.Int64("total_users_count", totalCount))

	return nil
}

// demonstrateHighConcurrency demonstrates high concurrency operations
func (cm *ConnectionManager) demonstrateHighConcurrency(ctx context.Context) error {
	cm.logger.Info(ctx, "Starting high concurrency demonstration")

	const numGoroutines = 20
	const operationsPerGoroutine = 5
	results := make(chan error, numGoroutines*operationsPerGoroutine)
	var wg sync.WaitGroup

	startTime := time.Now()

	// Start concurrent operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				user := &ExampleUser{
					Username: fmt.Sprintf("user_%d_%d", id, j),
					Email:    fmt.Sprintf("user_%d_%d@example.com", id, j),
					FullName: fmt.Sprintf("User %d-%d", id, j),
					IsActive: true,
				}

				if err := cm.userRepo.Create(ctx, user); err != nil {
					results <- fmt.Errorf("goroutine %d failed to create user %d: %w", id, j, err)
					return
				}

				results <- nil
			}
		}(i)
	}

	// Wait for all operations to complete
	wg.Wait()
	close(results)

	// Collect results
	var errors []error
	for err := range results {
		if err != nil {
			errors = append(errors, err)
		}
	}

	duration := time.Since(startTime)
	successRate := float64(numGoroutines*operationsPerGoroutine-len(errors)) / float64(numGoroutines*operationsPerGoroutine) * 100

	cm.logger.Info(ctx, "High concurrency demonstration completed",
		logging.Int("total_operations", numGoroutines*operationsPerGoroutine),
		logging.Float64("success_rate", successRate),
		logging.Duration("duration", duration),
		logging.Int("error_count", len(errors)))

	return nil
}

// demonstrateConnectionPooling demonstrates connection pooling
func (cm *ConnectionManager) demonstrateConnectionPooling(ctx context.Context) error {
	cm.logger.Info(ctx, "Starting connection pooling demonstration")

	// Use repository to count users
	count, err := cm.userRepo.CountAll(ctx)
	if err != nil {
		cm.logger.Error(ctx, "Failed to count users", logging.ErrorField("error", err))
		return fmt.Errorf("failed to count users: %w", err)
	}

	cm.logger.Info(ctx, "Connection pooling demonstration completed", logging.Int64("users_count", count))
	return nil
}

// demonstrateAdvancedRepositoryOperations demonstrates advanced repository features
func (cm *ConnectionManager) demonstrateAdvancedRepositoryOperations(ctx context.Context) error {
	cm.logger.Info(ctx, "Starting advanced repository operations demonstration")

	// Demonstrate transaction usage
	if err := cm.userRepo.WithTransaction(ctx, func(repo repository.Repository[ExampleUser]) error {
		// Create a user within transaction
		user := &ExampleUser{
			Username: "transaction_user",
			Email:    "transaction@example.com",
			FullName: "Transaction User",
			IsActive: true,
		}

		if err := repo.Create(ctx, user); err != nil {
			return fmt.Errorf("failed to create user in transaction: %w", err)
		}

		// Update user within same transaction
		user.FullName = "Updated Transaction User"
		if err := repo.Update(ctx, user); err != nil {
			return fmt.Errorf("failed to update user in transaction: %w", err)
		}

		cm.logger.Info(ctx, "Transaction completed successfully", logging.String("username", user.Username))
		return nil
	}); err != nil {
		cm.logger.Error(ctx, "Transaction demonstration failed", logging.ErrorField("error", err))
		return fmt.Errorf("transaction demonstration failed: %w", err)
	}

	// Demonstrate batch operations
	users := []ExampleUser{
		{Username: "batch_user_1", Email: "batch1@example.com", FullName: "Batch User 1", IsActive: true},
		{Username: "batch_user_2", Email: "batch2@example.com", FullName: "Batch User 2", IsActive: true},
		{Username: "batch_user_3", Email: "batch3@example.com", FullName: "Batch User 3", IsActive: true},
	}

	if err := cm.userRepo.CreateInBatches(ctx, users, 2); err != nil {
		cm.logger.Error(ctx, "Batch creation failed", logging.ErrorField("error", err))
		return fmt.Errorf("batch creation failed: %w", err)
	}

	// Demonstrate existence checks
	exists, err := cm.userRepo.ExistsByConditions(ctx, "username = ?", "batch_user_1")
	if err != nil {
		cm.logger.Error(ctx, "Existence check failed", logging.ErrorField("error", err))
		return fmt.Errorf("existence check failed: %w", err)
	}

	if exists {
		cm.logger.Info(ctx, "Batch user 1 exists as expected")
	}

	// Demonstrate pagination
	var paginatedUsers []ExampleUser
	if err := cm.userRepo.FindAllWithOffset(ctx, 5, 0, &paginatedUsers); err != nil {
		cm.logger.Error(ctx, "Pagination failed", logging.ErrorField("error", err))
		return fmt.Errorf("pagination failed: %w", err)
	}

	cm.logger.Info(ctx, "Advanced repository operations demonstration completed",
		logging.Int("paginated_users_count", len(paginatedUsers)))
	return nil
}

// getConnectionFromPool gets a connection from the pool
func (cm *ConnectionManager) getConnectionFromPool() (*gorm.DB, error) {
	select {
	case conn := <-cm.connectionPool.connections:
		cm.metrics.incrementActiveConnections()
		return conn, nil
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for connection from pool")
	}
}

// returnConnectionToPool returns a connection to the pool
func (cm *ConnectionManager) returnConnectionToPool(conn *gorm.DB) {
	cm.metrics.decrementActiveConnections()

	select {
	case cm.connectionPool.connections <- conn:
		// Connection returned successfully
	default:
		// Pool is full, close the connection
		if sqlDB, err := conn.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

// Shutdown gracefully shuts down the connection manager
func (cm *ConnectionManager) Shutdown() error {
	cm.logger.Info(context.Background(), "Starting graceful shutdown...")

	// Stop background services
	close(cm.healthChecker.stopChan)

	// Wait for background services to finish
	cm.shutdownWg.Wait()

	// Close connection pool
	cm.closeConnectionPool()

	// Close primary connection
	if cm.primaryDB != nil {
		if sqlDB, err := cm.primaryDB.DB(); err == nil {
			sqlDB.Close()
		}
	}

	cm.logger.Info(context.Background(), "Graceful shutdown completed")
	return nil
}

// closeConnectionPool closes all connections in the pool
func (cm *ConnectionManager) closeConnectionPool() {
	close(cm.connectionPool.connections)

	for conn := range cm.connectionPool.connections {
		if sqlDB, err := conn.DB(); err == nil {
			sqlDB.Close()
		}
	}
}

// start starts the health checker
func (hc *HealthChecker) start(cm *ConnectionManager, wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performHealthCheck(cm)
		case <-hc.stopChan:
			return
		}
	}
}

// performHealthCheck performs a health check on the database
func (hc *HealthChecker) performHealthCheck(cm *ConnectionManager) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.timeout)
	defer cancel()

	start := time.Now()

	// Test the connection with a repository operation
	if _, err := cm.userRepo.CountAll(ctx); err != nil {
		hc.setHealthStatus(false)
		cm.logger.Error(ctx, "Health check failed", logging.ErrorField("error", err))
		return
	}

	duration := time.Since(start)
	hc.setHealthStatus(true)

	cm.logger.Info(ctx, "Health check passed", logging.Duration("duration", duration))
}

// setHealthStatus sets the health status
func (hc *HealthChecker) setHealthStatus(status bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.healthStatus = status
}

// collectMetrics collects connection metrics
func (cm *ConnectionManager) collectMetrics(wg *sync.WaitGroup) {
	defer wg.Done()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cm.logger.Info(context.Background(), "Connection metrics",
				logging.Int64("active_connections", cm.metrics.getActiveConnections()),
				logging.Int64("total_connections", cm.metrics.getTotalConnections()),
				logging.Int64("failed_connections", cm.metrics.getFailedConnections()))
		case <-cm.shutdownChan:
			return
		}
	}
}

// incrementActiveConnections increments active connections count
func (cm *ConnectionMetrics) incrementActiveConnections() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.ActiveConnections++
}

// decrementActiveConnections decrements active connections count
func (cm *ConnectionMetrics) decrementActiveConnections() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.ActiveConnections > 0 {
		cm.ActiveConnections--
	}
}

// incrementTotalConnections increments total connections count
func (cm *ConnectionMetrics) incrementTotalConnections() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.TotalConnections++
}

// getActiveConnections gets active connections count
func (cm *ConnectionMetrics) getActiveConnections() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.ActiveConnections
}

// getTotalConnections gets total connections count
func (cm *ConnectionMetrics) getTotalConnections() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.TotalConnections
}

// getFailedConnections gets failed connections count
func (cm *ConnectionMetrics) getFailedConnections() int64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.FailedConnections
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvOrDefaultInt gets environment variable as int or returns default value
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := fmt.Sscanf(value, "%d", &defaultValue); err == nil && intValue == 1 {
			return defaultValue
		}
	}
	return defaultValue
}

// main function demonstrates the connection manager
func main() {
	// Create connection manager
	cm := NewConnectionManager()

	// Initialize the system
	if err := cm.Initialize(); err != nil {
		log.Fatalf("Failed to initialize connection manager: %v", err)
	}

	// Run the example
	if err := cm.Run(); err != nil {
		log.Fatalf("Connection manager example failed: %v", err)
	}

	log.Println("Connection manager example completed successfully")
}
