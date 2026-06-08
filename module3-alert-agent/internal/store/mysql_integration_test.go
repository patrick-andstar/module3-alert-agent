package store_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"module3-alert-agent/internal/store"
)

func TestMySQLSchemaAppliesToConfiguredDatabase(t *testing.T) {
	if os.Getenv("RUN_MYSQL_INTEGRATION") != "1" {
		t.Skip("set RUN_MYSQL_INTEGRATION=1 to run MySQL integration validation")
	}

	cfg := store.MySQLConfig{
		Host:     getenvOrDefault("MYSQL_HOST", "127.0.0.1"),
		Port:     getenvIntOrDefault("MYSQL_PORT", 3306),
		User:     getenvOrDefault("MYSQL_USER", "root"),
		Password: os.Getenv("MYSQL_PASSWORD"),
		Database: getenvOrDefault("MYSQL_DATABASE", "dlp_agent"),
	}

	if cfg.Password == "" {
		t.Fatal("MYSQL_PASSWORD is required for integration validation")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminDB, err := store.Open(ctx, store.MySQLConfig{
		Host:     cfg.Host,
		Port:     cfg.Port,
		User:     cfg.User,
		Password: cfg.Password,
		Database: "mysql",
	})
	if err != nil {
		t.Fatalf("connect mysql admin DB: %v", err)
	}
	defer adminDB.Close()

	if _, err := adminDB.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+cfg.Database); err != nil {
		t.Fatalf("create database %s: %v", cfg.Database, err)
	}

	db, err := store.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("connect target DB: %v", err)
	}
	defer db.Close()

	schemaPath := filepath.Clean("../../sql/schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}

	for _, stmt := range splitSQLStatements(string(schema)) {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("apply schema statement %q: %v", stmt, err)
		}
	}

	rows, err := db.QueryContext(ctx, `
SELECT table_name
FROM information_schema.tables
WHERE table_schema = ?
ORDER BY table_name`, cfg.Database)
	if err != nil {
		t.Fatalf("query tables: %v", err)
	}
	defer rows.Close()

	found := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan table name: %v", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table names: %v", err)
	}

	for _, name := range []string{"alert_logs", "false_positive_library", "whitelist_rules"} {
		if !found[name] {
			t.Fatalf("expected table %s to exist after applying schema; found=%v", name, found)
		}
	}
}

func getenvOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscan(value, &parsed); err != nil {
		return fallback
	}
	return parsed
}

func splitSQLStatements(schema string) []string {
	parts := strings.Split(schema, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		statements = append(statements, stmt)
	}
	return statements
}
