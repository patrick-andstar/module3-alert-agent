package agent_test

import (
	"context"
	"testing"

	volcmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/config"
)

func TestNewEinoRuntimeRejectsMissingArkConfig(t *testing.T) {
	_, err := agent.NewEinoRuntime(context.Background(), config.ArkConfig{})
	if err == nil {
		t.Fatal("NewEinoRuntime returned nil error, want validation error")
	}
}

func TestNewEinoRuntimeBuildsArkComponents(t *testing.T) {
	runtime, err := agent.NewEinoRuntime(context.Background(), config.ArkConfig{
		ChatModelEndpoint: "ep-chat",
		APIKey:            "test-key",
	})
	if err != nil {
		t.Fatalf("NewEinoRuntime returned error: %v", err)
	}
	if runtime.ChatModel == nil {
		t.Fatal("ChatModel is nil")
	}
}

func TestNewArkChatModelConfigDisablesLongThinkingForAgentLatency(t *testing.T) {
	cfg := agent.NewArkChatModelConfig(config.ArkConfig{
		ChatModelEndpoint: "ep-chat",
		APIKey:            "test-key",
	})

	if cfg.Model != "ep-chat" || cfg.APIKey != "test-key" {
		t.Fatalf("config model/api key = %q/%q, want propagated values", cfg.Model, cfg.APIKey)
	}
	if cfg.MaxTokens == nil || *cfg.MaxTokens > 512 {
		t.Fatalf("MaxTokens = %#v, want bounded output tokens", cfg.MaxTokens)
	}
	if cfg.Temperature == nil || *cfg.Temperature > 0.2 {
		t.Fatalf("Temperature = %#v, want low variance", cfg.Temperature)
	}
	if cfg.Thinking == nil || cfg.Thinking.Type != volcmodel.ThinkingTypeDisabled {
		t.Fatalf("Thinking = %#v, want disabled for ReAct latency", cfg.Thinking)
	}
	if cfg.RetryTimes == nil || *cfg.RetryTimes != 0 {
		t.Fatalf("RetryTimes = %#v, want no SDK retries under request timeout", cfg.RetryTimes)
	}
}
