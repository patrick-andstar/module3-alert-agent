package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"module3-alert-agent/internal/model"
	"module3-alert-agent/internal/pipeline"
)

type ToolContext struct {
	Events              map[string]model.Event
	Whitelist           *pipeline.WhitelistCache
	FalsePositiveStore  FalsePositiveStore
	FalsePositives      []model.FalsePositiveRecord
	Recalls             []RecallResult
	TopK                int
	StructuredThreshold float64
	Now                 func() time.Time
}

type EventIDInput struct {
	EventID string `json:"event_id" jsonschema:"description=事件 ID"`
}

type EventInput struct {
	Event model.Event `json:"event" jsonschema:"description=告警事件"`
}

type MarkFalsePositiveInput struct {
	Event   model.Event `json:"event" jsonschema:"description=告警事件"`
	Reason  string      `json:"reason" jsonschema:"description=误报原因"`
	TTLDays int         `json:"ttl_days" jsonschema:"description=误报记录有效天数"`
}

type SearchHistoryOutput struct {
	Results []RecallResult `json:"results"`
}

type EventDetailOutput struct {
	Event model.Event `json:"event"`
}

type WhitelistOutput struct {
	Matched bool `json:"matched"`
}

type MarkFalsePositiveOutput struct {
	Recommended bool `json:"recommended"`
}

func BuildTools(ctx ToolContext) ([]tool.BaseTool, error) {
	search, err := utils.InferTool("SearchFalsePositiveHistory", "结构化召回当前事件的 Top-K 历史误报候选", func(c context.Context, input EventIDInput) (SearchHistoryOutput, error) {
		return SearchFalsePositiveHistory(c, ctx, input)
	})
	if err != nil {
		return nil, err
	}
	detail, err := utils.InferTool("GetEventDetail", "按 event_id 获取告警事件完整 JSON 上下文", func(c context.Context, input EventIDInput) (EventDetailOutput, error) {
		return GetEventDetail(c, ctx, input)
	})
	if err != nil {
		return nil, err
	}
	whitelist, err := utils.InferTool("QueryWhitelist", "查询当前事件是否命中白名单规则", func(c context.Context, input EventInput) (WhitelistOutput, error) {
		return QueryWhitelist(c, ctx, input)
	})
	if err != nil {
		return nil, err
	}
	mark, err := utils.InferTool("MarkAsFalsePositive", "建议标记误报，不直接写入误报库", func(c context.Context, input MarkFalsePositiveInput) (MarkFalsePositiveOutput, error) {
		return MarkAsFalsePositive(c, ctx, input)
	})
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{search, detail, whitelist, mark}, nil
}

func SearchFalsePositiveHistory(_ context.Context, ctx ToolContext, _ EventIDInput) (SearchHistoryOutput, error) {
	return SearchHistoryOutput{
		Results: ctx.Recalls,
	}, nil
}

func GetEventDetail(_ context.Context, ctx ToolContext, input EventIDInput) (EventDetailOutput, error) {
	event, ok := ctx.Events[input.EventID]
	if !ok {
		return EventDetailOutput{}, fmt.Errorf("event %q not found", input.EventID)
	}
	return EventDetailOutput{Event: event}, nil
}

func QueryWhitelist(_ context.Context, ctx ToolContext, input EventInput) (WhitelistOutput, error) {
	if ctx.Whitelist == nil {
		return WhitelistOutput{}, nil
	}
	return WhitelistOutput{Matched: ctx.Whitelist.Match(input.Event)}, nil
}

func MarkAsFalsePositive(_ context.Context, ctx ToolContext, input MarkFalsePositiveInput) (MarkFalsePositiveOutput, error) {
	return MarkFalsePositiveOutput{Recommended: input.Reason != ""}, nil
}
