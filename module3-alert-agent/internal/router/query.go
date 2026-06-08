package router

import "module3-alert-agent/internal/model"

type AlertQuery = model.AlertQuery

type AlertQueryResult struct {
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	Data     []model.Event `json:"data"`
}

func normalizeAlertQuery(query AlertQuery) AlertQuery {
	return model.NormalizeAlertQuery(query)
}

func matchesAlertQuery(event model.Event, query AlertQuery) bool {
	return model.MatchesAlertQuery(event, query)
}
