package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Security SecurityConfig `yaml:"security"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Ark      ArkConfig      `yaml:"ark"`
	Pipeline PipelineConfig `yaml:"pipeline"`
	Agent    AgentConfig    `yaml:"agent"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type SecurityConfig struct {
	AdminToken string `yaml:"admin_token"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type ArkConfig struct {
	ChatModelEndpoint string `yaml:"chat_model_endpoint"`
	APIKey            string `yaml:"api_key"`
}

type PipelineConfig struct {
	DedupWindows map[string]int `yaml:"dedup_windows"`
}

type AgentConfig struct {
	SystemPromptPath          string  `yaml:"system_prompt_path"`
	MaxConcurrency            int     `yaml:"max_concurrency"`
	RecallStrategy            string  `yaml:"recall_strategy"`
	TopKRecall                int     `yaml:"top_k_recall"`
	StructuredRecallThreshold float64 `yaml:"structured_recall_threshold"`
	StrongRecallThreshold     float64 `yaml:"strong_recall_threshold"`
	ConfidenceThreshold       float64 `yaml:"confidence_threshold"`
	FalsePositiveTTLDays      int     `yaml:"false_positive_ttl_days"`
	AnalysisTimeoutSec        int     `yaml:"analysis_timeout_sec"`
	MaxRecallRecords          int     `yaml:"max_recall_records"`
	LLMRetryCount             int     `yaml:"llm_retry_count"`
}

func Load(path string) (Config, error) {
	var cfg Config
	content, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	applyEnvOverrides(&cfg)

	if err := validate(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if value := os.Getenv("MYSQL_HOST"); value != "" {
		cfg.MySQL.Host = value
	}
	if value := getenvInt("MYSQL_PORT"); value > 0 {
		cfg.MySQL.Port = value
	}
	if value := os.Getenv("MYSQL_USER"); value != "" {
		cfg.MySQL.User = value
	}
	if value := os.Getenv("MYSQL_PASSWORD"); value != "" {
		cfg.MySQL.Password = value
	}
	if value := os.Getenv("MYSQL_DATABASE"); value != "" {
		cfg.MySQL.Database = value
	}
	if value := os.Getenv("ARK_CHAT_MODEL"); value != "" {
		cfg.Ark.ChatModelEndpoint = value
	}
	if value := os.Getenv("ARK_API_KEY"); value != "" {
		cfg.Ark.APIKey = value
	}
	if value := os.Getenv("ADMIN_API_TOKEN"); value != "" {
		cfg.Security.AdminToken = value
	}
	if value := getenvInt("AGENT_ANALYSIS_TIMEOUT_SEC"); value > 0 {
		cfg.Agent.AnalysisTimeoutSec = value
	}
	if value := getenvInt("AGENT_MAX_RECALL_RECORDS"); value > 0 {
		cfg.Agent.MaxRecallRecords = value
	}
	if value := getenvInt("AGENT_LLM_RETRY_COUNT"); value > 0 {
		cfg.Agent.LLMRetryCount = value
	}
}

func getenvInt(key string) int {
	value := os.Getenv(key)
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 {
		return fmt.Errorf("server.port must be greater than 0")
	}
	if cfg.MySQL.Host == "" {
		return fmt.Errorf("mysql.host is required")
	}
	if cfg.MySQL.Port <= 0 {
		return fmt.Errorf("mysql.port must be greater than 0")
	}
	if cfg.MySQL.User == "" {
		return fmt.Errorf("mysql.user is required")
	}
	if strings.TrimSpace(cfg.MySQL.Password) == "your_password" {
		return fmt.Errorf("mysql.password must be set via MYSQL_PASSWORD or replaced with a real local value")
	}
	if cfg.MySQL.Database == "" {
		return fmt.Errorf("mysql.database is required")
	}
	if cfg.Ark.ChatModelEndpoint == "" || strings.TrimSpace(cfg.Ark.ChatModelEndpoint) == "set-via-env" {
		return fmt.Errorf("ark.chat_model_endpoint must be set via ARK_CHAT_MODEL")
	}
	if cfg.Ark.APIKey == "" {
		return fmt.Errorf("ark.api_key must be set via ARK_API_KEY")
	}
	for _, level := range []string{"critical", "high", "medium", "low", "info"} {
		if cfg.Pipeline.DedupWindows[level] <= 0 {
			return fmt.Errorf("pipeline.dedup_windows.%s must be greater than 0", level)
		}
	}
	if cfg.Agent.MaxConcurrency <= 0 {
		return fmt.Errorf("agent.max_concurrency must be greater than 0")
	}
	if cfg.Agent.TopKRecall <= 0 {
		return fmt.Errorf("agent.top_k_recall must be greater than 0")
	}
	if cfg.Agent.RecallStrategy == "" {
		cfg.Agent.RecallStrategy = "structured"
	}
	if cfg.Agent.RecallStrategy != "structured" {
		return fmt.Errorf("agent.recall_strategy must be structured")
	}
	if cfg.Agent.StructuredRecallThreshold <= 0 {
		cfg.Agent.StructuredRecallThreshold = 0.60
	}
	if cfg.Agent.StrongRecallThreshold <= 0 {
		cfg.Agent.StrongRecallThreshold = 0.75
	}
	if cfg.Agent.ConfidenceThreshold <= 0 {
		return fmt.Errorf("agent.confidence_threshold must be greater than 0")
	}
	if cfg.Agent.FalsePositiveTTLDays <= 0 {
		return fmt.Errorf("agent.false_positive_ttl_days must be greater than 0")
	}
	if cfg.Agent.AnalysisTimeoutSec <= 0 {
		cfg.Agent.AnalysisTimeoutSec = 30
	}
	if cfg.Agent.MaxRecallRecords <= 0 {
		cfg.Agent.MaxRecallRecords = 500
	}
	if cfg.Agent.LLMRetryCount < 0 {
		return fmt.Errorf("agent.llm_retry_count must not be negative")
	}
	return nil
}
