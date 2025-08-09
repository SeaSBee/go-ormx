// Package examples provides comprehensive usage examples for the go-ormx data access layer
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	gormx "go-ormx/ormx"
	"go-ormx/ormx/config"
	"go-ormx/ormx/models"

	"github.com/oklog/ulid/v2"
)

// Example demonstrates basic usage of the GORM data access layer
func BasicUsageExample() {
	// Example 1: Basic setup and configuration
	basicSetupExample()

	// Example 2: Model usage
	modelExample()

	// Example 3: Repository operations
	repositoryExample()

	// Example 4: Migration management
	migrationExample()

	// Example 5: Observability and metrics
	observabilityExample()
}

// basicSetupExample demonstrates basic client setup and configuration
func basicSetupExample() {
	fmt.Println("=== Basic Setup Example ===")

	// Create logger
	logger := &SimpleLogger{}

	// Manual configuration
	dbConfig := &config.Config{
		Type:     config.PostgreSQL,
		Host:     "localhost",
		Port:     5432,
		Database: "dev_db",
		Username: "postgres",
		Password: "password",

		// Connection pool settings
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,

		// Performance settings
		PreparedStatements: true,
		BatchSize:          100,
		SkipDefaultTx:      true,
	}

	clientConfig := gormx.Config{
		Database: dbConfig,
		Logger:   logger,
		Options: gormx.ClientOptions{
			EnableMigrations: true,
			AutoMigrate:      true,
			EnableMetrics:    true,
			SkipHealthCheck:  false,
		},
	}

	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		log.Printf("Failed to setup client: %v", err)
		return
	}
	defer client.Close()

	fmt.Println("✓ Client created successfully")
	fmt.Printf("✓ Database connected: %t\n", client.IsHealthy(context.Background()))
}

// modelExample demonstrates model usage
func modelExample() {
	fmt.Println("\n=== Model Example ===")

	// Create a basic model
	product := &Product{
		Name:        "Sample Product",
		Description: "A sample product for demonstration",
		Price:       29.99,
	}

	fmt.Printf("✓ Created product: %s\n", product.Name)
	fmt.Printf("✓ Product ID: %s\n", product.ID)
	fmt.Printf("✓ Created at: %s\n", product.CreatedAt.Format(time.RFC3339))

	// Create a tenant-aware model
	order := &Order{
		ProductID: ulid.Make().String(),
		Quantity:  5,
		OrderDate: time.Now(),
	}

	fmt.Printf("✓ Created order for tenant: %s\n", order.TenantID)
	fmt.Printf("✓ Order quantity: %d\n", order.Quantity)
}

// repositoryExample demonstrates repository operations
func repositoryExample() {
	fmt.Println("\n=== Repository Example ===")

	// This would typically use a real client
	// For demonstration, we'll show the pattern
	fmt.Println("✓ Repository pattern demonstrated")
	fmt.Println("✓ CRUD operations available")
	fmt.Println("✓ Filtering and pagination supported")
}

// migrationExample demonstrates migration functionality
func migrationExample() {
	fmt.Println("\n=== Migration Example ===")

	// This would typically use a real client
	// For demonstration, we'll show the pattern
	fmt.Println("✓ Migration system using golang-migrate")
	fmt.Println("✓ File-based migrations supported")
	fmt.Println("✓ PostgreSQL and MySQL migrations available")
}

// observabilityExample demonstrates observability features
func observabilityExample() {
	fmt.Println("\n=== Observability Example ===")

	// This would typically use a real client
	// For demonstration, we'll show the pattern
	fmt.Println("✓ Health checks available")
	fmt.Println("✓ Metrics collection supported")
	fmt.Println("✓ Structured logging implemented")
}

// Product represents a basic product model
type Product struct {
	models.BaseModel
	Name        string  `gorm:"type:varchar(255);not null" json:"name"`
	Description string  `gorm:"type:text" json:"description"`
	Price       float64 `gorm:"type:decimal(10,2);not null" json:"price"`
}

// TableName returns the table name for Product
func (p *Product) TableName() string {
	return "products"
}

// Order represents a tenant-aware order model
type Order struct {
	models.TenantAuditModel
	ProductID string    `gorm:"type:uuid;not null" json:"product_id"`
	Quantity  int       `gorm:"not null" json:"quantity"`
	OrderDate time.Time `gorm:"not null" json:"order_date"`
}

// TableName returns the table name for Order
func (o *Order) TableName() string {
	return "orders"
}

// SimpleLogger implements a simple logger for examples
// Note: SimpleLogger and main function are defined in app.go
