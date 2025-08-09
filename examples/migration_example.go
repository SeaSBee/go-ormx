// Package examples provides migration examples using golang-migrate
package main

import (
	"context"
	"fmt"
	"log"

	gormx "go-ormx/ormx"
	"go-ormx/ormx/config"
)

// Example demonstrates the new golang-migrate based migration system
func MigrationExample() {
	fmt.Println("=== Migration Example (golang-migrate) ===")

	// Create logger
	logger := &SimpleLogger{}

	// Create client configuration
	clientConfig := gormx.Config{
		Database: &config.Config{
			Type:     config.PostgreSQL,
			Host:     "localhost",
			Port:     5432,
			Database: "migration_test",
			Username: "postgres",
			Password: "password",
			SSLMode:  "disable",
		},
		Logger: logger,
		Options: gormx.ClientOptions{
			EnableMigrations: true,
			AutoMigrate:      false, // We'll run migrations manually
		},
	}

	// Create client
	client, err := gormx.NewClient(clientConfig)
	if err != nil {
		log.Fatal("Failed to create client:", err)
	}
	defer client.Close()

	ctx := context.Background()

	// Example 1: Get migration status
	fmt.Println("\n1. Getting migration status...")
	status, err := client.GetMigrationStatus(ctx)
	if err != nil {
		log.Printf("Failed to get migration status: %v", err)
	} else {
		fmt.Printf("✓ Current version: %d\n", status.Version)
		fmt.Printf("✓ Dirty: %t\n", status.Dirty)
		fmt.Printf("✓ Applied count: %d\n", status.AppliedCount)
		fmt.Printf("✓ Pending count: %d\n", status.PendingCount)
	}

	// Example 2: Run migrations
	fmt.Println("\n2. Running migrations...")
	if err := client.Migrate(ctx); err != nil {
		log.Printf("Migration failed: %v", err)
	} else {
		fmt.Println("✓ Migrations completed successfully")
	}

	// Example 3: Get updated status
	fmt.Println("\n3. Getting updated migration status...")
	status, err = client.GetMigrationStatus(ctx)
	if err != nil {
		log.Printf("Failed to get migration status: %v", err)
	} else {
		fmt.Printf("✓ Current version: %d\n", status.Version)
		fmt.Printf("✓ Applied count: %d\n", status.AppliedCount)
	}

	// Example 4: Demonstrate migration file structure
	fmt.Println("\n4. Migration File Structure:")
	fmt.Println("   migrations/")
	fmt.Println("   ├── postgresql/")
	fmt.Println("   │   ├── 000001_create_users_table.up.sql")
	fmt.Println("   │   ├── 000001_create_users_table.down.sql")
	fmt.Println("   │   ├── 000002_create_user_sessions_table.up.sql")
	fmt.Println("   │   └── 000002_create_user_sessions_table.down.sql")
	fmt.Println("   └── mysql/")
	fmt.Println("       ├── 000001_create_users_table.up.sql")
	fmt.Println("       ├── 000001_create_users_table.down.sql")
	fmt.Println("       ├── 000002_create_user_sessions_table.up.sql")
	fmt.Println("       └── 000002_create_user_sessions_table.down.sql")

	// Example 5: Migration file format
	fmt.Println("\n5. Migration File Format:")
	fmt.Println("   -- 000001_create_users_table.up.sql")
	fmt.Println("   CREATE TABLE users (")
	fmt.Println("       id UUID PRIMARY KEY DEFAULT gen_random_uuid(),")
	fmt.Println("       email VARCHAR(255) UNIQUE NOT NULL,")
	fmt.Println("       created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP")
	fmt.Println("   );")
	fmt.Println("   ")
	fmt.Println("   -- 000001_create_users_table.down.sql")
	fmt.Println("   DROP TABLE users;")

	// Example 6: Advanced migration features
	fmt.Println("\n6. Advanced Migration Features:")
	fmt.Println("   ✓ Version-based migrations")
	fmt.Println("   ✓ Up/Down migrations")
	fmt.Println("   ✓ Database-specific migrations")
	fmt.Println("   ✓ Transaction safety")
	fmt.Println("   ✓ Rollback support")
	fmt.Println("   ✓ Status tracking")
	fmt.Println("   ✓ Force version setting")

	// Example 7: Migration commands
	fmt.Println("\n7. Available Migration Commands:")
	fmt.Println("   client.Migrate(ctx)           // Run all pending migrations")
	fmt.Println("   client.MigrateTo(ctx, 2)      // Migrate to specific version")
	fmt.Println("   client.Rollback(ctx)          // Rollback last migration")
	fmt.Println("   client.RollbackTo(ctx, 1)     // Rollback to specific version")
	fmt.Println("   client.Force(ctx, 3)          // Force set version")
	fmt.Println("   client.Drop(ctx)              // Drop all tables (dangerous!)")
	fmt.Println("   client.GetMigrationStatus(ctx) // Get current status")

	// Example 8: Production considerations
	fmt.Println("\n8. Production-Grade Features:")
	fmt.Println("   ✓ Uses golang-migrate/migrate library")
	fmt.Println("   ✓ Industry-standard migration format")
	fmt.Println("   ✓ Support for PostgreSQL and MySQL")
	fmt.Println("   ✓ File-based migrations (version controlled)")
	fmt.Println("   ✓ Atomic migrations (transaction-based)")
	fmt.Println("   ✓ Comprehensive error handling")
	fmt.Println("   ✓ Detailed logging and monitoring")
}

// Example demonstrates migration best practices
func MigrationBestPractices() {
	fmt.Println("\n=== Migration Best Practices ===")

	fmt.Println("\n1. File Naming Convention:")
	fmt.Println("   - Use numeric prefixes: 000001_, 000002_, etc.")
	fmt.Println("   - Use descriptive names: create_users_table")
	fmt.Println("   - Always include .up.sql and .down.sql files")
	fmt.Println("   - Organize by database type: postgresql/, mysql/")

	fmt.Println("\n2. Migration Content:")
	fmt.Println("   - Keep migrations atomic and focused")
	fmt.Println("   - Use IF NOT EXISTS / IF EXISTS clauses")
	fmt.Println("   - Always provide rollback (down) migration")
	fmt.Println("   - Test both up and down migrations")

	fmt.Println("\n3. Version Control:")
	fmt.Println("   - Never modify applied migrations")
	fmt.Println("   - Create new migrations for changes")
	fmt.Println("   - Use descriptive commit messages")
	fmt.Println("   - Review migrations before applying")

	fmt.Println("\n4. Database-Specific Considerations:")
	fmt.Println("   - PostgreSQL: Use UUID, JSONB, advanced features")
	fmt.Println("   - MySQL: Use CHAR(36) for UUID, JSON type")
	fmt.Println("   - Handle database-specific syntax differences")
	fmt.Println("   - Test on both database types")

	fmt.Println("\n5. Production Deployment:")
	fmt.Println("   - Run migrations during maintenance windows")
	fmt.Println("   - Monitor migration progress")
	fmt.Println("   - Have rollback plan ready")
	fmt.Println("   - Test migrations on staging first")

	fmt.Println("\n6. Error Handling:")
	fmt.Println("   - Check for dirty state after failures")
	fmt.Println("   - Use Force() to fix dirty state if needed")
	fmt.Println("   - Log all migration operations")
	fmt.Println("   - Handle partial failures gracefully")

	fmt.Println("\n7. Performance Considerations:")
	fmt.Println("   - Use CONCURRENTLY for index creation (PostgreSQL)")
	fmt.Println("   - Batch large data migrations")
	fmt.Println("   - Avoid blocking operations during peak hours")
	fmt.Println("   - Monitor migration duration")

	fmt.Println("\n8. Security:")
	fmt.Println("   - Use parameterized queries")
	fmt.Println("   - Avoid dynamic SQL generation")
	fmt.Println("   - Validate migration files")
	fmt.Println("   - Use least privilege database users")
}

// MigrationMain demonstrates the migration system
func MigrationMain() {
	// Run migration examples
	MigrationExample()
	MigrationBestPractices()
}
