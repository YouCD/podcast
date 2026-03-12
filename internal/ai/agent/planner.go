package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	chatModel model.ToolCallingChatModel
}

// NewPlanner 创建规划器
func NewPlanner(chatModel model.ToolCallingChatModel) *Planner {
	return &Planner{
		chatModel: chatModel,
	}
}

// CreatePlan 根据用户问题创建执行计划
func (p *Planner) CreatePlan(ctx context.Context, query string, toolsInfo []*schema.ToolInfo) (*Plan, error) {
	// 构建工具描述
	toolsDesc := p.buildToolsDescription(toolsInfo)

	// 构建规划提示词
	planningPrompt := fmt.Sprintf(planningPromptTemplate, query, toolsDesc)

	messages := []*schema.Message{
		schema.SystemMessage(planningSystemPrompt),
		schema.UserMessage(planningPrompt),
	}

	// 调用 LLM 生成计划
	response, err := p.chatModel.Generate(ctx, messages)
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
		schema.SystemMessage(replanningSystemPrompt),
		schema.UserMessage(replanPrompt),
	}

	response, err := p.chatModel.Generate(ctx, messages)
	if err != nil {
		return nil, err
	}

	// 解析新计划
	plan, err := p.parsePlanResponse(response.Content, query)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// buildToolsDescription 构建工具描述
func (p *Planner) buildToolsDescription(toolsInfo []*schema.ToolInfo) string {
	var desc strings.Builder
	for _, info := range toolsInfo {
		desc.WriteString(fmt.Sprintf("- **%s**: %s\n", info.Name, info.Desc))
	}
	return desc.String()
}

// parsePlanResponse 解析 LLM 返回的计划
func (p *Planner) parsePlanResponse(response string, query string) (*Plan, error) {
	// 尝试从响应中提取 JSON
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		// 如果无法提取 JSON，创建一个简单的单步计划
		return &Plan{
			Query: query,
			Steps: []PlanStep{
				{
					ID:          1,
					Description: "直接回答用户问题",
					ToolName:    "none",
					ToolArgs:    "{}",
					Reason:      "无法解析复杂计划，直接处理",
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

// extractJSON 从文本中提取 JSON
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
				return text[start : i+1]
			}
		}
	}
	return ""
}
