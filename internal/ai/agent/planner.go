package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"podcast/config"
	"podcast/pkg/types"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// PlanStep 表示计划中的一个步骤
type PlanStep struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	ToolName    string `json:"tool_name"`
	ToolArgs    string `json:"tool_args"`
	Reason      string `json:"reason"`
	Status      string `json:"status"` // pending, running, completed, failed, skipped
	Result      string `json:"result,omitempty"`
}

// Plan 表示完整的执行计划
type Plan struct {
	Query       string     `json:"query"`
	Steps       []PlanStep `json:"steps"`
	CurrentStep int        `json:"current_step"`
	IsComplete  bool       `json:"is_complete"`
}

// Planner 规划器
type Planner struct {
	chatModel   model.ToolCallingChatModel
	strategySrc *types.ToolStrategySource // 工具策略来源（内部配置 + mcpProxy），可为 nil
}

// NewPlanner 创建规划器
func NewPlanner(chatModel model.ToolCallingChatModel, strategySrc *types.ToolStrategySource) *Planner {
	return &Planner{
		chatModel:   chatModel,
		strategySrc: strategySrc,
	}
}

// CreatePlan 根据用户问题创建执行计划
// history 为多轮对话历史（不含当前问题），用于解析指代/省略
func (p *Planner) CreatePlan(ctx context.Context, query string, toolsInfo []*schema.ToolInfo, history []*schema.Message) (*Plan, error) {
	// 构建工具描述（动态：已配置策略的工具带详细策略，其余仅基础描述）
	toolsDesc := p.buildToolsDescription(toolsInfo)
	// 构建工具使用优先级列表（动态：仅包含已配置策略的工具）
	priority := p.buildPrioritySection(toolsInfo)

	// 构建规划提示词
	planningPrompt := fmt.Sprintf(planningPromptTemplate, buildHistorySection(history), query, toolsDesc)

	messages := []*schema.Message{
		schema.SystemMessage(fmt.Sprintf(planningSystemPromptTemplate, time.Now().Format("2006-01-02 15:04:05"), priority)),
		schema.UserMessage(planningPrompt),
	}

	// 调用 LLM 生成计划
	response, err := p.chatModel.Generate(ctx, messages, qwen.WithExtraHeader(config.GetUA()))
	if err != nil {
		return nil, err
	}

	// 解析计划
	plan, err := p.parsePlanResponse(response.Content, query)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// Replan 根据执行结果重新规划
func (p *Planner) Replan(ctx context.Context, currentPlan *Plan, executedSteps []PlanStep, query string) (*Plan, error) {
	// 构建已执行步骤的摘要
	var executedSummary strings.Builder
	for _, step := range executedSteps {
		executedSummary.WriteString(fmt.Sprintf("步骤 %d: %s\n工具: %s\n结果: %s\n状态: %s\n\n",
			step.ID, step.Description, step.ToolName, step.Result, step.Status))
	}

	replanPrompt := fmt.Sprintf(replanningPromptTemplate, query, executedSummary.String())

	messages := []*schema.Message{
		schema.SystemMessage(fmt.Sprintf(replanningSystemPromptTemplate, time.Now().Format("2006-01-02 15:04:05"))),
		schema.UserMessage(replanPrompt),
	}

	response, err := p.chatModel.Generate(ctx, messages, qwen.WithExtraHeader(config.GetUA()))
	if err != nil {
		return nil, err
	}

	// 解析新计划
	plan, err := p.parsePlanResponse(response.Content, query)
	if err != nil {
		return nil, err
	}

	// 携带旧计划中已完成的步骤（按 ID + 工具名匹配），避免重规划后重复执行
	completed := make(map[int]PlanStep, len(currentPlan.Steps))
	for _, s := range currentPlan.Steps {
		if s.Status == "completed" {
			completed[s.ID] = s
		}
	}
	for i := range plan.Steps {
		if old, ok := completed[plan.Steps[i].ID]; ok && plan.Steps[i].ToolName == old.ToolName {
			plan.Steps[i].Status = "completed"
			plan.Steps[i].Result = old.Result
		}
	}

	return plan, nil
}

// buildHistorySection 构建对话历史段落（用于规划/最终答案提示词）
// 无历史时返回空字符串
func buildHistorySection(history []*schema.Message) string {
	if len(history) == 0 {
		return "（无）"
	}
	var b strings.Builder
	for _, m := range history {
		role := "用户"
		switch m.Role {
		case schema.Assistant:
			role = "助手"
		case schema.System:
			role = "系统"
		}
		content := strings.TrimSpace(m.Content)
		if runes := []rune(content); len(runes) > 500 {
			content = string(runes[:500]) + "..."
		}
		if content == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("[%s] %s\n", role, content))
	}
	return strings.TrimSpace(b.String())
}

// buildToolsDescription 构建工具描述（动态）：
// 已配置策略的工具（内部配置/mcpProxy）优先展示并附带详细策略，
// 其余工具仅展示基础描述（来自工具注册信息，外部 MCP 工具即 MCP 服务器下发的描述）。
func (p *Planner) buildToolsDescription(toolsInfo []*schema.ToolInfo) string {
	var desc strings.Builder
	for _, entry := range p.orderedTools(toolsInfo) {
		ts := p.strategyFor(entry.info.Name)
		if ts != nil && ts.Strategy != "" {
			desc.WriteString(fmt.Sprintf("### %s\n%s\n%s\n\n", entry.info.Name, entry.info.Desc, ts.Strategy))
		} else {
			desc.WriteString(fmt.Sprintf("- **%s**: %s\n", entry.info.Name, entry.info.Desc))
		}
	}
	return desc.String()
}

// buildPrioritySection 构建"工具使用优先级"列表（动态）：
// 仅包含已配置策略的工具；内部工具用配置的 summary，外部工具用工具自带描述（截断）。
func (p *Planner) buildPrioritySection(toolsInfo []*schema.ToolInfo) string {
	var lines []string
	n := 1
	for _, entry := range p.orderedTools(toolsInfo) {
		ts := p.strategyFor(entry.info.Name)
		if ts == nil || ts.Strategy == "" {
			continue
		}
		oneLine := ts.Summary
		if oneLine == "" {
			oneLine = truncateRunes(entry.info.Desc, 60)
		}
		lines = append(lines, fmt.Sprintf("%d. **%s**：%s", n, entry.info.Name, oneLine))
		n++
	}
	if len(lines) == 0 {
		return "（无特殊优先级，按工具描述使用）"
	}
	return strings.Join(lines, "\n")
}

// toolEntry 工具信息 + 是否为内部工具（用于排序）
type toolEntry struct {
	info     *schema.ToolInfo
	internal bool
}

// orderedTools 返回有序工具列表：内部工具（按策略配置顺序）在前，其余按注册顺序在后
func (p *Planner) orderedTools(toolsInfo []*schema.ToolInfo) []toolEntry {
	registered := make(map[string]*schema.ToolInfo, len(toolsInfo))
	for _, info := range toolsInfo {
		registered[info.Name] = info
	}

	var out []toolEntry
	seen := make(map[string]bool, len(toolsInfo))
	// 1. 内部工具：按 AgentToolsConfig 配置顺序
	if p.strategySrc != nil && p.strategySrc.Internal != nil {
		for i := range p.strategySrc.Internal.Tools {
			name := p.strategySrc.Internal.Tools[i].Name
			if info, ok := registered[name]; ok {
				out = append(out, toolEntry{info: info, internal: true})
				seen[name] = true
			}
		}
	}
	// 2. 其余工具（含外部 MCP 代理工具）：按注册顺序
	for _, info := range toolsInfo {
		if !seen[info.Name] {
			out = append(out, toolEntry{info: info})
			seen[info.Name] = true
		}
	}
	return out
}

// strategyFor 解析工具的策略（未配置返回 nil）
func (p *Planner) strategyFor(name string) *types.ToolStrategy {
	if p.strategySrc == nil {
		return nil
	}
	return p.strategySrc.Resolve(name)
}

// truncateRunes 按 rune 截断字符串（避免截断中文/多字节字符）
func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

// parsePlanResponse 解析 LLM 返回的计划
func (p *Planner) parsePlanResponse(response string, query string) (*Plan, error) {
	// 尝试从响应中提取 JSON
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		// 无法提取 JSON：回退为直接检索知识库，保证最终答案有据可依（而非裸 LLM 编造）
		return &Plan{
			Query: query,
			Steps: []PlanStep{
				{
					ID:          1,
					Description: "直接检索知识库",
					ToolName:    "pgvector_search",
					ToolArgs:    fmt.Sprintf(`{"query": %q, "top_k": 5, "reason": "计划解析失败，直接检索知识库"}`, query),
					Reason:      "无法解析复杂计划，直接检索知识库",
					Status:      "pending",
				},
			},
			CurrentStep: 0,
			IsComplete:  false,
		}, nil
	}

	var plan Plan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("解析计划失败: %w", err)
	}

	plan.Query = query
	plan.CurrentStep = 0
	plan.IsComplete = false

	// 初始化所有步骤状态
	for i := range plan.Steps {
		plan.Steps[i].Status = "pending"
	}

	return &plan, nil
}

// extractJSON 从文本中提取 JSON（支持 LLM 返回的混合中文+JSON 响应）
func extractJSON(text string) string {
	// 尝试找到 JSON 块
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}

	// 找到匹配的结束括号
	depth := 0
	for i := start; i < len(text); i++ {
		if text[i] == '{' {
			depth++
		} else if text[i] == '}' {
			depth--
			if depth == 0 {
				candidate := text[start : i+1]
				// 验证 JSON 合法性
				var temp interface{}
				if err := json.Unmarshal([]byte(candidate), &temp); err == nil {
					return candidate
				}
				// JSON 不合法，继续寻找下一个 { 开始的可能
				break
			}
		}
	}

	// 如果提取的 JSON 不合法，尝试整体解析
	var temp interface{}
	if err := json.Unmarshal([]byte(text), &temp); err == nil {
		return text
	}

	return ""
}
