package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"module3-alert-agent/internal/agent"
	"module3-alert-agent/internal/config"
	"module3-alert-agent/internal/router"
	"module3-alert-agent/internal/store"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := store.Open(ctx, store.MySQLConfig{
		Host:     cfg.MySQL.Host,
		Port:     cfg.MySQL.Port,
		User:     cfg.MySQL.User,
		Password: cfg.MySQL.Password,
		Database: cfg.MySQL.Database,
	})
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}

	service, err := store.NewMySQLService(ctx, db, cfg.Pipeline.DedupWindows)
	if err != nil {
		log.Fatalf("initialize mysql service: %v", err)
	}
	service.SetAnalysisTimeout(time.Duration(cfg.Agent.AnalysisTimeoutSec) * time.Second)
	service.SetMaxRecallRecords(cfg.Agent.MaxRecallRecords)
	runtime, err := agent.NewEinoRuntime(ctx, cfg.Ark)
	if err != nil {
		log.Fatalf("initialize eino runtime: %v", err)
	}
	systemPrompt, err := agent.LoadSystemPrompt(cfg.Agent.SystemPromptPath)
	if err != nil {
		log.Fatalf("load system prompt: %v", err)
	}
	analyzer := agent.NewRuntimeAnalyzer(runtime, service.WhitelistCache(), service, service, cfg.Agent)
	analyzer.SetSystemPrompt(systemPrompt)
	service.SetAnalyzer(agent.NewLimitedAnalyzer(analyzer, cfg.Agent.MaxConcurrency))

	h := router.BuildWithOptions(fmt.Sprintf(":%d", cfg.Server.Port), service, router.Options{
		AdminToken: cfg.Security.AdminToken,
	})
	h.OnShutdown = append(h.OnShutdown, func(context.Context) {
		service.Close()
		_ = db.Close()
	})
	h.Spin()
}
