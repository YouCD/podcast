package types

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAgentToolsConfig_Nonexistent(t *testing.T) {
	cfg, err := LoadAgentToolsConfig("/nonexistent/agent_tools.yaml")
	if err != nil {
		t.Fatalf("文件不存在不应报错: %v", err)
	}
	if cfg == nil || len(cfg.Tools) != 0 {
		t.Fatalf("文件不存在应返回空配置, got %+v", cfg)
	}
}

func TestLoadAgentToolsConfig_EmptyPath(t *testing.T) {
	cfg, err := LoadAgentToolsConfig("")
	if err != nil {
		t.Fatalf("空路径不应报错: %v", err)
	}
	if len(cfg.Tools) != 0 {
		t.Fatalf("空路径应返回空配置, got %+v", cfg)
	}
}

func TestLoadAgentToolsConfig_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent_tools.yaml")
	content := `
tools:
  - name: tool_a
    summary: 工具A
    strategy: "策略A"
  - name: tool_b
    summary: 工具B
    strategy: "策略B"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadAgentToolsConfig(path)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(cfg.Tools) != 2 {
		t.Fatalf("期望 2 个工具, got %d", len(cfg.Tools))
	}
	if cfg.Tools[0].Name != "tool_a" || cfg.Tools[0].Summary != "工具A" || cfg.Tools[0].Strategy != "策略A" {
		t.Fatalf("工具A解析错误: %+v", cfg.Tools[0])
	}
}

func TestLoadAgentToolsConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "agent_tools.yaml")
	if err := os.WriteFile(path, []byte("tools: [unclosed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAgentToolsConfig(path); err == nil {
		t.Fatal("非法 YAML 应报错")
	}
}

func TestStrategyFor(t *testing.T) {
	cfg := &AgentToolsConfig{
		Tools: []ToolStrategy{{Name: "a", Summary: "A"}, {Name: "b", Summary: "B"}},
	}
	if ts := cfg.StrategyFor("a"); ts == nil || ts.Summary != "A" {
		t.Fatalf("StrategyFor(a) 错误: %+v", ts)
	}
	if ts := cfg.StrategyFor("c"); ts != nil {
		t.Fatalf("StrategyFor(c) 应为 nil, got %+v", ts)
	}
	var nilCfg *AgentToolsConfig
	if ts := nilCfg.StrategyFor("a"); ts != nil {
		t.Fatalf("nil 配置 StrategyFor 应为 nil, got %+v", ts)
	}
}

func TestToolStrategySource_Resolve_Internal(t *testing.T) {
	src := &ToolStrategySource{
		Internal: &AgentToolsConfig{
			Tools: []ToolStrategy{{Name: "pgvector_search", Summary: "内部库", Strategy: "策略"}},
		},
	}
	ts := src.Resolve("pgvector_search")
	if ts == nil || ts.Strategy != "策略" {
		t.Fatalf("内部工具解析错误: %+v", ts)
	}
	if ts := src.Resolve("unknown"); ts != nil {
		t.Fatalf("未配置工具应为 nil, got %+v", ts)
	}
}

func TestToolStrategySource_Resolve_Proxy(t *testing.T) {
	src := &ToolStrategySource{
		Internal: &AgentToolsConfig{},
		MCPProxy: map[string]*Mcp{
			"web_search": {URL: "http://x", Strategy: "外部策略"},
		},
	}
	// 外部工具名 = <proxyKey>_<remoteTool>
	ts := src.Resolve("web_search_bailian_web_search")
	if ts == nil || ts.Strategy != "外部策略" {
		t.Fatalf("外部工具解析错误: %+v", ts)
	}
	if !src.IsProxyTool("web_search_bailian_web_search") {
		t.Fatal("IsProxyTool 应识别外部工具")
	}
	if src.IsProxyTool("pgvector_search") {
		t.Fatal("IsProxyTool 不应误判内部工具")
	}
}

func TestToolStrategySource_Resolve_LongestPrefix(t *testing.T) {
	src := &ToolStrategySource{
		Internal: &AgentToolsConfig{},
		MCPProxy: map[string]*Mcp{
			"web":        {URL: "http://a", Strategy: "短前缀策略"},
			"web_search": {URL: "http://b", Strategy: "长前缀策略"},
		},
	}
	ts := src.Resolve("web_search_tool1")
	if ts == nil || ts.Strategy != "长前缀策略" {
		t.Fatalf("应取最长前缀匹配, got %+v", ts)
	}
}

func TestToolStrategySource_Resolve_Nil(t *testing.T) {
	var src *ToolStrategySource
	if ts := src.Resolve("a"); ts != nil {
		t.Fatalf("nil 来源 Resolve 应为 nil, got %+v", ts)
	}
	if src.IsProxyTool("a") {
		t.Fatal("nil 来源 IsProxyTool 应为 false")
	}
}
