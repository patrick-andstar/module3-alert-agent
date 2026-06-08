package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"module3-alert-agent/internal/config"
)

func TestLoadUsesDefaultsFromConfigFileAndEnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := []byte(`
server:
  port: 9090
mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "placeholder"
  database: "dlp_agent"
ark:
  chat_model_endpoint: "file-chat"
  api_key: "file-key"
pipeline:
  dedup_windows:
    critical: 30
    high: 60
    medium: 180
    low: 300
    info: 600
agent:
  system_prompt_path: "prompt/system.md"
  max_concurrency: 5
  recall_strategy: "structured"
  top_k_recall: 5
  structured_recall_threshold: 0.60
  strong_recall_threshold: 0.75
  confidence_threshold: 0.8
  false_positive_ttl_days: 30
`)
	if err := os.WriteFile(cfgPath, content, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("ARK_CHAT_MODEL", "env-chat")
	t.Setenv("ARK_API_KEY", "env-key")
	t.Setenv("MYSQL_HOST", "192.168.1.10")
	t.Setenv("MYSQL_PORT", "4406")
	t.Setenv("MYSQL_USER", "admin")
	t.Setenv("MYSQL_PASSWORD", "env-password")
	t.Setenv("MYSQL_DATABASE", "env_db")

	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Fatalf("Server.Port = %d, want 9090", cfg.Server.Port)
	}
	if cfg.MySQL.Host != "192.168.1.10" {
		t.Fatalf("MySQL.Host = %q, want env override", cfg.MySQL.Host)
	}
	if cfg.MySQL.Port != 4406 {
		t.Fatalf("MySQL.Port = %d, want env override", cfg.MySQL.Port)
	}
	if cfg.MySQL.User != "admin" {
		t.Fatalf("MySQL.User = %q, want env override", cfg.MySQL.User)
	}
	if cfg.MySQL.Password != "env-password" {
		t.Fatalf("MySQL.Password = %q, want env override", cfg.MySQL.Password)
	}
	if cfg.MySQL.Database != "env_db" {
		t.Fatalf("MySQL.Database = %q, want env override", cfg.MySQL.Database)
	}
	if cfg.Ark.ChatModelEndpoint != "env-chat" {
		t.Fatalf("Ark.ChatModelEndpoint = %q, want env-chat", cfg.Ark.ChatModelEndpoint)
	}
	if cfg.Ark.APIKey != "env-key" {
		t.Fatalf("Ark.APIKey = %q, want env-key", cfg.Ark.APIKey)
	}
	if cfg.Pipeline.DedupWindows["high"] != 60 {
		t.Fatalf("high dedup window = %d, want 60", cfg.Pipeline.DedupWindows["high"])
	}
	if cfg.Agent.ConfidenceThreshold != 0.8 {
		t.Fatalf("confidence threshold = %v, want 0.8", cfg.Agent.ConfidenceThreshold)
	}
	if cfg.Agent.RecallStrategy != "structured" {
		t.Fatalf("recall strategy = %q, want structured", cfg.Agent.RecallStrategy)
	}
	if cfg.Agent.StructuredRecallThreshold != 0.60 {
		t.Fatalf("structured recall threshold = %v, want 0.60", cfg.Agent.StructuredRecallThreshold)
	}
	if cfg.Agent.StrongRecallThreshold != 0.75 {
		t.Fatalf("strong recall threshold = %v, want 0.75", cfg.Agent.StrongRecallThreshold)
	}
	if cfg.Agent.SystemPromptPath != "prompt/system.md" {
		t.Fatalf("system prompt path = %q, want prompt/system.md", cfg.Agent.SystemPromptPath)
	}
}

func TestLoadRejectsMissingRequiredDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("server:\n  port: 0\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := config.Load(cfgPath)
	if err == nil {
		t.Fatal("Load returned nil error, want validation error")
	}
}

func TestLoadRejectsPlaceholderMySQLPassword(t *testing.T) {
	t.Setenv("MYSQL_PASSWORD", "")
	t.Setenv("ARK_CHAT_MODEL", "")
	t.Setenv("ARK_API_KEY", "")
	cfgPath := writeConfig(t, `
server:
  port: 9090
mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_password"
  database: "dlp_agent"
ark:
  chat_model_endpoint: "ep-chat"
  api_key: "test-key"
pipeline:
  dedup_windows:
    critical: 30
    high: 60
    medium: 180
    low: 300
    info: 600
agent:
  max_concurrency: 5
  top_k_recall: 5
  confidence_threshold: 0.8
  false_positive_ttl_days: 30
`)

	_, err := config.Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "mysql.password") {
		t.Fatalf("Load error = %v, want mysql.password placeholder validation", err)
	}
}

func TestLoadRejectsPlaceholderArkChatModel(t *testing.T) {
	t.Setenv("ARK_CHAT_MODEL", "")
	t.Setenv("ARK_API_KEY", "")
	cfgPath := writeConfig(t, `
server:
  port: 9090
mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: ""
  database: "dlp_agent"
ark:
  chat_model_endpoint: "set-via-env"
  api_key: "test-key"
pipeline:
  dedup_windows:
    critical: 30
    high: 60
    medium: 180
    low: 300
    info: 600
agent:
  max_concurrency: 5
  top_k_recall: 5
  confidence_threshold: 0.8
  false_positive_ttl_days: 30
`)

	_, err := config.Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "ark.chat_model_endpoint") {
		t.Fatalf("Load error = %v, want ark.chat_model_endpoint placeholder validation", err)
	}
}

func TestLoadRejectsMissingArkAPIKey(t *testing.T) {
	t.Setenv("ARK_CHAT_MODEL", "")
	t.Setenv("ARK_API_KEY", "")
	cfgPath := writeConfig(t, `
server:
  port: 9090
mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: ""
  database: "dlp_agent"
ark:
  chat_model_endpoint: "ep-chat"
  api_key: ""
pipeline:
  dedup_windows:
    critical: 30
    high: 60
    medium: 180
    low: 300
    info: 600
agent:
  max_concurrency: 5
  top_k_recall: 5
  confidence_threshold: 0.8
  false_positive_ttl_days: 30
`)

	_, err := config.Load(cfgPath)
	if err == nil || !strings.Contains(err.Error(), "ark.api_key") {
		t.Fatalf("Load error = %v, want ark.api_key validation", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return cfgPath
}
