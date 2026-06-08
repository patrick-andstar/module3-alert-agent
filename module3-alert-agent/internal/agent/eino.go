package agent

import (
	"context"
	"fmt"
	"time"

	einomodel "github.com/cloudwego/eino-ext/components/model/ark"
	volcmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"module3-alert-agent/internal/config"
)

type EinoRuntime struct {
	ChatModel *einomodel.ChatModel
}

func NewEinoRuntime(ctx context.Context, cfg config.ArkConfig) (*EinoRuntime, error) {
	if cfg.ChatModelEndpoint == "" {
		return nil, fmt.Errorf("ark.chat_model_endpoint is required")
	}
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ark.api_key is required")
	}

	chatModel, err := einomodel.NewChatModel(ctx, NewArkChatModelConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("create ark chat model: %w", err)
	}

	return &EinoRuntime{
		ChatModel: chatModel,
	}, nil
}

func NewArkChatModelConfig(cfg config.ArkConfig) *einomodel.ChatModelConfig {
	maxTokens := 512
	temperature := float32(0.1)
	timeout := 60 * time.Second
	retryTimes := 0
	return &einomodel.ChatModelConfig{
		Model:       cfg.ChatModelEndpoint,
		APIKey:      cfg.APIKey,
		MaxTokens:   &maxTokens,
		Temperature: &temperature,
		Timeout:     &timeout,
		RetryTimes:  &retryTimes,
		Thinking: &volcmodel.Thinking{
			Type: volcmodel.ThinkingTypeDisabled,
		},
	}
}
