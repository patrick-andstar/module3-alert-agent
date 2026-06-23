package agent_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"module3-alert-agent/internal/agent"
)

func TestLoadSystemPromptReturnsDefaultWhenPathIsEmpty(t *testing.T) {
	prompt, err := agent.LoadSystemPrompt("")
	if err != nil {
		t.Fatalf("LoadSystemPrompt returned error: %v", err)
	}
	if prompt == "" {
		t.Fatal("default prompt is empty")
	}
}

func TestDefaultSystemPromptDocumentsStructuredVerdictRules(t *testing.T) {
	prompt, err := agent.LoadSystemPrompt("")
	if err != nil {
		t.Fatalf("LoadSystemPrompt returned error: %v", err)
	}
	for _, needle := range []string{
		"agent_verdict",
		"recall_score",
		"不要直接写入误报库",
		"真实告警保持原风险等级",
		"自然语言字段必须使用中文",
		"不要翻译文件名",
	} {
		if !strings.Contains(prompt, needle) {
			t.Fatalf("default prompt missing %q: %s", needle, prompt)
		}
	}
}

func TestLoadSystemPromptReadsAndTrimsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "system.md")
	if err := os.WriteFile(path, []byte("\ncustom prompt\n"), 0o600); err != nil {
		t.Fatalf("write prompt: %v", err)
	}

	prompt, err := agent.LoadSystemPrompt(path)
	if err != nil {
		t.Fatalf("LoadSystemPrompt returned error: %v", err)
	}
	if prompt != "custom prompt" {
		t.Fatalf("prompt = %q, want trimmed file content", prompt)
	}
}
