package store_test

import (
	"testing"

	"module3-alert-agent/internal/store"
)

func TestMySQLDSNBuildsExpectedConnectionString(t *testing.T) {
	cfg := store.MySQLConfig{
		Host:     "127.0.0.1",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "dlp_agent",
	}

	dsn := store.DSN(cfg)
	want := "root:secret@tcp(127.0.0.1:3306)/dlp_agent?charset=utf8mb4&parseTime=True&loc=Local"
	if dsn != want {
		t.Fatalf("DSN = %q, want %q", dsn, want)
	}
}
