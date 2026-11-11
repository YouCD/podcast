package types

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ToolStrategy 单个工具的使用策略（用于动态生成 Agent 规划/执行提示词）
type ToolStrategy struct {
	// Name 工具名（与注册的工具名一致）
	Name string `yaml:"name"`
	// Summary 一句话定位（用于"工具使用优先级"列表）
	Summary string `yaml:"summary"`
	// Strategy 详细使用策略（用于"工具详解"段落）
	Strategy string `yaml:"strategy"`
}

// AgentToolsConfig Agent 工具策略配置（工具配置文件）
// 配置文件中的条目顺序即提示词中的优先级顺序
type AgentToolsConfig struct {
	Tools []ToolStrategy `yaml:"tools"`
}

// LoadAgentToolsConfig 加载工具策略配置。
// path 为空或文件不存在时返回空配置（提示词退化为仅工具基础描述），不视为错误；
// 文件存在但解析失败时返回错误。
func LoadAgentToolsConfig(path string) (*AgentToolsConfig, error) {
	cfg := &AgentToolsConfig{}
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("读取工具配置文件失败 %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析工具配置文件失败 %s: %w", path, err)
	}
	return cfg, nil
}

// StrategyFor 按工具名查找使用策略，未配置时返回 nil
func (c *AgentToolsConfig) StrategyFor(name string) *ToolStrategy {
	if c == nil {
		return nil
	}
	for i := range c.Tools {
		if c.Tools[i].Name == name {
			return &c.Tools[i]
		}
	}
	return nil
}

// ToolStrategySource 工具策略的统一来源：
// 内部工具来自 AgentToolsConfig（agent_tools.yaml），
// 外部 MCP 代理工具来自 mcpProxy 配置段（types.Mcp 的 summary/strategy 字段）。
type ToolStrategySource struct {
	Internal *AgentToolsConfig
	MCPProxy map[string]*Mcp
}

// Resolve 按工具名解析使用策略。
// 外部工具（名称形如 <proxyKey>_<tool>）优先从 mcpProxy 配置读取（取最长前缀匹配）；
// 内部工具从 AgentToolsConfig 读取；均未配置时返回 nil。
func (s *ToolStrategySource) Resolve(name string) *ToolStrategy {
	if s == nil {
		return nil
	}
	// 1. 外部 MCP 代理工具：<proxyKey>_ 前缀匹配（取最长 key）
	bestKey := ""
	var best *Mcp
	for key, m := range s.MCPProxy {
		if m == nil || !strings.HasPrefix(name, key+"_") {
			continue
		}
		if len(key) > len(bestKey) {
			bestKey = key
			best = m
		}
	}
	if best != nil && best.Strategy != "" {
		return &ToolStrategy{Name: name, Strategy: best.Strategy}
	}
	// 2. 内部工具
	return s.Internal.StrategyFor(name)
}

// IsProxyTool 判断工具名是否属于某个 MCP 代理（<proxyKey>_ 前缀）
func (s *ToolStrategySource) IsProxyTool(name string) bool {
	if s == nil {
		return false
	}
	for key := range s.MCPProxy {
		if strings.HasPrefix(name, key+"_") {
			return true
		}
	}
	return false
}
