package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	einoagent "github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"module3-alert-agent/internal/config"
	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

type RuntimeAnalyzer struct {
	runtime    *EinoRuntime
	whitelist  *pipeline.WhitelistCache
	store      FalsePositiveStore
	fpSource   FalsePositiveSource
	cfg        config.AgentConfig
	system     string
	now        func() time.Time
	newAgentFn func(context.Context, ToolContext) (*react.Agent, error)
}

type FalsePositiveSource interface {
	ListFalsePositiveRecordsForAnalysis(context.Context) ([]model.FalsePositiveRecord, error)
}

type EventAnalyzer interface {
	Analyze(context.Context, model.Event) (model.Event, error)
}

type LimitedAnalyzer struct {
	inner EventAnalyzer
	sem   chan struct{}
}

func NewLimitedAnalyzer(inner EventAnalyzer, maxConcurrency int) *LimitedAnalyzer {
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	return &LimitedAnalyzer{
		inner: inner,
		sem:   make(chan struct{}, maxConcurrency),
	}
}

func (a *LimitedAnalyzer) Analyze(ctx context.Context, event model.Event) (model.Event, error) {
	if a == nil || a.inner == nil {
		return event, fmt.Errorf("inner analyzer is required")
	}
	select {
	case a.sem <- struct{}{}:
		defer func() { <-a.sem }()
	case <-ctx.Done():
		return event, ctx.Err()
	}
	return a.inner.Analyze(ctx, event)
}

func NewRuntimeAnalyzer(runtime *EinoRuntime, whitelist *pipeline.WhitelistCache, store FalsePositiveStore, fpSource FalsePositiveSource, cfg config.AgentConfig) *RuntimeAnalyzer {
	analyzer := &RuntimeAnalyzer{
		runtime:   runtime,
		whitelist: whitelist,
		store:     store,
		fpSource:  fpSource,
		cfg:       cfg,
		system:    DefaultDecisionSystemPrompt,
		now:       time.Now,
	}
	analyzer.newAgentFn = analyzer.newAgent
	return analyzer
}

func (a *RuntimeAnalyzer) SetSystemPrompt(prompt string) {
	if strings.TrimSpace(prompt) != "" {
		a.system = prompt
	}
}

func (a *RuntimeAnalyzer) Analyze(ctx context.Context, event model.Event) (model.Event, error) {
	if a == nil || a.runtime == nil || a.runtime.ChatModel == nil {
		return event, fmt.Errorf("agent runtime is required")
	}

	falsePositives := []model.FalsePositiveRecord{}
	var err error
	if a.fpSource != nil {
		falsePositives, err = a.fpSource.ListFalsePositiveRecordsForAnalysis(ctx)
		if err != nil {
			return event, fmt.Errorf("list false positives: %w", err)
		}
	}
	recalls := StructuredRecall(event, falsePositives, StructuredRecallConfig{
		TopK:      coalesceZero(a.cfg.TopKRecall, 5),
		Threshold: coalesceZeroFloat(a.cfg.StructuredRecallThreshold, 0.60),
		Now:       a.now,
	})

	toolCtx := ToolContext{
		Events:              map[string]model.Event{event.EventID: event},
		Whitelist:           a.whitelist,
		FalsePositiveStore:  a.store,
		FalsePositives:      falsePositives,
		Recalls:             recalls,
		TopK:                coalesceZero(a.cfg.TopKRecall, 5),
		StructuredThreshold: coalesceZeroFloat(a.cfg.StructuredRecallThreshold, 0.60),
		Now:                 a.now,
	}
	reactAgent, err := a.newAgentFn(ctx, toolCtx)
	if err != nil {
		return event, err
	}

	message, err := a.generateDecision(ctx, reactAgent, event, recalls)
	if err != nil {
		return event, fmt.Errorf("generate decision: %w", err)
	}

	decision, err := ParseDecision(message.Content)
	if err != nil {
		return event, err
	}
	topScore := topRecallScore(recalls)

	return ApplyDecision(event, decision, a.store, DecisionConfig{
		ConfidenceThreshold:   coalesceZeroFloat(a.cfg.ConfidenceThreshold, 0.8),
		StrongRecallThreshold: coalesceZeroFloat(a.cfg.StrongRecallThreshold, 0.75),
		FalsePositiveTTLDays:  coalesceZero(a.cfg.FalsePositiveTTLDays, 30),
		Now:                   a.now,
		HasRecall:             len(recalls) > 0,
		RecallScore:           topScore,
	})
}

func (a *RuntimeAnalyzer) newAgent(ctx context.Context, toolCtx ToolContext) (*react.Agent, error) {
	tools, err := BuildTools(toolCtx)
	if err != nil {
		return nil, err
	}
	return react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: a.runtime.ChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
		MaxStep: 6,
	})
}

type decisionGenerator interface {
	Generate(context.Context, []*schema.Message, ...einoagent.AgentOption) (*schema.Message, error)
}

func (a *RuntimeAnalyzer) generateDecision(ctx context.Context, generator decisionGenerator, event model.Event, recalls []RecallResult) (*schema.Message, error) {
	messages := []*schema.Message{
		schema.SystemMessage(a.system),
		schema.UserMessage(BuildDecisionPrompt(event, recalls)),
	}
	attempts := coalesceZero(a.cfg.LLMRetryCount, 1) + 1
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		message, err := generator.Generate(ctx, messages)
		if err == nil {
			return message, nil
		}
		lastErr = err
		if attempt == attempts-1 {
			break
		}
		select {
		case <-time.After(time.Duration(attempt+1) * 200 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

func BuildDecisionPrompt(event model.Event, recalls []RecallResult) string {
	var b strings.Builder
	b.WriteString("判断以下告警是否为误报。只能输出 JSON。\n")
	b.WriteString("自然语言字段必须使用中文，尤其是 false_positive_reason 和 explanation；不要翻译文件名、进程名、域名、URL、路径、event_id、host_id、user_id、哈希值、规则 ID、API 名称和 JSON 字段名。\n")
	b.WriteString("当前事件:\n")
	writeJSON(&b, event)
	b.WriteString("\n误报召回历史:\n")
	writeJSON(&b, summarizeRecalls(recalls))
	b.WriteString("\n如果上下文足够，直接返回 JSON 决策，不要调用工具。只有事件或召回上下文缺失、冲突时才调用工具。")
	b.WriteString("\nMarkAsFalsePositive 只表示推荐；服务端决定是否根据召回证据和 confidence 更新误报库。返回 JSON 时，真实告警保持原风险，疑似误报只能建议有边界的降级。")
	return b.String()
}

type recallSummary struct {
	ID            int64   `json:"id"`
	HostID        string  `json:"host_id"`
	UserID        string  `json:"user_id"`
	SensitiveType string  `json:"sensitive_type"`
	RiskLevel     string  `json:"risk_level"`
	ProcessName   string  `json:"process_name"`
	ProcessPath   string  `json:"process_path"`
	Target        string  `json:"target"`
	Operation     string  `json:"operation"`
	Reason        string  `json:"reason"`
	RecallScore   float64 `json:"recall_score"`
}

func summarizeRecalls(recalls []RecallResult) []recallSummary {
	summaries := make([]recallSummary, 0, len(recalls))
	for _, recall := range recalls {
		record := recall.Record
		summaries = append(summaries, recallSummary{
			ID:            record.ID,
			HostID:        record.HostID,
			UserID:        record.UserID,
			SensitiveType: record.SensitiveType,
			RiskLevel:     record.RiskLevel,
			ProcessName:   record.ProcessName,
			ProcessPath:   record.ProcessPath,
			Target:        record.Target,
			Operation:     record.Operation,
			Reason:        record.Reason,
			RecallScore:   recallScore(recall),
		})
	}
	return summaries
}

func topRecallScore(recalls []RecallResult) float64 {
	if len(recalls) == 0 {
		return 0
	}
	return recallScore(recalls[0])
}

func recallScore(recall RecallResult) float64 {
	if recall.Score > 0 {
		return recall.Score
	}
	return recall.Similarity
}

func ParseDecision(content string) (Decision, error) {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return Decision{}, fmt.Errorf("agent response did not contain JSON object")
	}
	var decision Decision
	if err := json.Unmarshal([]byte(content[start:end+1]), &decision); err != nil {
		return Decision{}, fmt.Errorf("parse decision json: %w", err)
	}
	return decision, nil
}

func writeJSON(b *strings.Builder, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		b.WriteString("{}")
		return
	}
	b.Write(data)
}

func coalesceZero(value, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func coalesceZeroFloat(value, fallback float64) float64 {
	if value <= 0 {
		return fallback
	}
	return value
}
