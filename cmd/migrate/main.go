package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <database_path> [migration_file]")
	}

	dbPath := os.Args[1]
	migrationFile := "../../migrations/001_add_refresh_tokens.sql"
	if len(os.Args) > 2 {
		migrationFile = os.Args[2]
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Read migration file
	migrationPath, err := filepath.Abs(migrationFile)
	if err != nil {
		log.Fatalf("Failed to resolve migration path: %v", err)
	}

	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	// Execute migration
	fmt.Printf("Applying migration: %s\n", migrationPath)
	fmt.Printf("To database: %s\n", dbPath)

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	fmt.Println("✓ Migration applied successfully!")

	// Verify tables were created
	tables := []string{"refresh_tokens", "jwt_blacklist", "auth_audit_log", "schema_migrations"}
	for _, table := range tables {
		var count int
		err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s'", table)).Scan(&count)
		if err != nil {
			log.Printf("Warning: Could not verify table %s: %v", table, err)
			continue
		}
		if count > 0 {
			fmt.Printf("✓ Table '%s' created\n", table)
		} else {
			fmt.Printf("✗ Table '%s' NOT created\n", table)
		}
	}

	// Check if jwt_version column was added to users table
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('users') WHERE name='jwt_version'").Scan(&count)
	if err == nil && count > 0 {
		fmt.Println("✓ JWT columns added to 'users' table")
	} else {
		fmt.Println("✗ JWT columns NOT added to 'users' table (may already exist)")
	}
}
