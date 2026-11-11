package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"podcast/config"
	"podcast/pkg/types"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// fallbackMaxStep ReAct 兜底执行器的最大步数（仅在直调失败时使用，防止失控）
const fallbackMaxStep = 5

// maxStepResultLen 单步结果写入上下文的最大长度（字符），防止单工具返回巨量数据撑爆上下文
const maxStepResultLen = 8000

// maxSummaryResultLen 最终答案摘要中单步结果的最大长度（字符）
const maxSummaryResultLen = 4000

// truncateResult 截断超长结果（按 rune，避免截断多字节字符）
func truncateResult(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "\n...（结果过长已截断）"
}

// PlanExecuteAgent Plan-and-Execute 模式的 Agent
// 执行策略：计划步骤优先直接调用工具（零额外 LLM 开销），
// 参数不合法/工具报错时降级到小 ReAct 循环兜底；
// 全部步骤执行后按结果质量决定是否重规划（上限 MaxReplan）
type PlanExecuteAgent struct {
	planner   *Planner
	executor  *react.Agent // ReAct 兜底执行器（仅直调失败时使用）
	chatModel model.ToolCallingChatModel
	tools     []tool.BaseTool
	toolMap   map[string]tool.BaseTool
	maxReplan int

	// 回调相关
	mu             sync.Mutex
	currentPlan    *Plan
	executedSteps  []PlanStep
	messageHandler func(ctx context.Context, stage string, data map[string]any)
}

// PlanExecuteAgentConfig Agent 配置
type PlanExecuteAgentConfig struct {
	ChatModel     model.ToolCallingChatModel
	Tools         []tool.BaseTool
	MaxReplan     int // 最大重规划次数
	MessageHandle func(ctx context.Context, stage string, data map[string]any)
	StrategySrc   *types.ToolStrategySource // 工具策略来源（内部配置 + mcpProxy），可为 nil
}

// NewPlanExecuteAgent 创建 Plan-Execute Agent
func NewPlanExecuteAgent(config *PlanExecuteAgentConfig) (*PlanExecuteAgent, error) {
	if config.MaxReplan == 0 {
		config.MaxReplan = 3
	}

	// 工具名索引（用于步骤直调）
	toolMap := make(map[string]tool.BaseTool, len(config.Tools))
	for _, t := range config.Tools {
		info, err := t.Info(context.Background())
		if err != nil {
			return nil, fmt.Errorf("获取工具 %v 信息失败: %w", t, err)
		}
		toolMap[info.Name] = t
	}

	// ReAct 兜底执行器：仅在直调失败时使用，限制小步数防止失控
	executor, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: config.ChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: config.Tools,
			// 截断工具结果，防止单个工具返回巨量数据撑爆 ReAct 上下文
			ToolCallMiddlewares: []compose.ToolMiddleware{
				{
					Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
						return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
							out, err := next(ctx, input)
							if err == nil && out != nil {
								out.Result = truncateResult(out.Result, maxStepResultLen)
							}
							return out, err
						}
					},
				},
			},
		},
		MaxStep: fallbackMaxStep,
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{
				schema.SystemMessage(fmt.Sprintf(executorPersonaTemplate, time.Now().Format("2006-01-02 15:04:05"))),
			}, input...)
		},
		StreamToolCallChecker: func(ctx context.Context, sr *schema.StreamReader[*schema.Message]) (bool, error) {
			defer sr.Close()
			for {
				msg, err := sr.Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return false, err
				}
				if len(msg.ToolCalls) > 0 {
					return true, nil
				}
			}
			return false, nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("创建 react.Agent 兜底执行器失败: %w", err)
	}

	return &PlanExecuteAgent{
		planner:        NewPlanner(config.ChatModel, config.StrategySrc),
		executor:       executor,
		chatModel:      config.ChatModel,
		tools:          config.Tools,
		toolMap:        toolMap,
		maxReplan:      config.MaxReplan,
		messageHandler: config.MessageHandle,
	}, nil
}

// Run 执行 Plan-Execute 流程
// history 为多轮对话历史（不含当前问题，按时间正序）
func (a *PlanExecuteAgent) Run(ctx context.Context, input *schema.Message, history []*schema.Message) (*schema.Message, error) {
	query := input.Content

	// 通知开始规划
	a.emitEvent(ctx, "planning_start", map[string]any{
		"query": query,
	})

	// 1. 获取工具信息
	toolsInfo, err := a.getToolInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取工具信息失败: %w", err)
	}

	// 2. 创建初始计划
	plan, err := a.planner.CreatePlan(ctx, query, toolsInfo, history)
	if err != nil {
		return nil, fmt.Errorf("创建计划失败: %w", err)
	}

	a.mu.Lock()
	a.currentPlan = plan
	a.executedSteps = nil
	a.mu.Unlock()

	// 通知计划创建完成
	a.emitEvent(ctx, "planning_complete", map[string]any{
		"plan": plan,
	})

	// 3. 执行计划（含重规划循环）
	plan, err = a.executeWithReplan(ctx, query, plan)
	if err != nil {
		return nil, err
	}

	// 4. 生成最终答案
	finalAnswer, err := a.generateFinalAnswer(ctx, query, history)
	if err != nil {
		return nil, fmt.Errorf("生成最终答案失败: %w", err)
	}

	// 通知完成
	a.emitEvent(ctx, "execution_complete", map[string]any{
		"answer": finalAnswer,
	})

	return schema.AssistantMessage(finalAnswer, nil), nil
}

// Stream 流式执行并返回结果
// history 为多轮对话历史（不含当前问题，按时间正序）
func (a *PlanExecuteAgent) Stream(ctx context.Context, input *schema.Message, history []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	query := input.Content

	// 获取工具信息
	toolsInfo, err := a.getToolInfo(ctx)
	if err != nil {
		return nil, err
	}

	// 创建计划
	plan, err := a.planner.CreatePlan(ctx, query, toolsInfo, history)
	if err != nil {
		return nil, err
	}

	a.mu.Lock()
	a.currentPlan = plan
	a.executedSteps = nil
	a.mu.Unlock()

	// 发送计划信息
	if a.messageHandler != nil {
		planJSON, _ := json.Marshal(plan)
		a.messageHandler(ctx, "plan_created", map[string]any{
			"plan": string(planJSON),
		})
	}

	// 执行计划（含重规划循环）
	plan, err = a.executeWithReplan(ctx, query, plan)
	if err != nil {
		return nil, err
	}

	// 生成最终答案（流式）
	return a.streamFinalAnswer(ctx, query, history)
}

// executeWithReplan 执行计划步骤，并按结果质量决定是否重规划
// 返回最终计划（已完成步骤带 Result）
func (a *PlanExecuteAgent) executeWithReplan(ctx context.Context, query string, plan *Plan) (*Plan, error) {
	replanCount := 0
	for {
		// 执行未完成步骤（空计划直接跳过，用于简单问题直接回答）
		for i := range plan.Steps {
			if plan.Steps[i].Status == "completed" {
				continue
			}
			step := &plan.Steps[i]

			a.emitEvent(ctx, "step_start", map[string]any{
				"step_id":     step.ID,
				"description": step.Description,
				"tool_name":   step.ToolName,
				"reason":      step.Reason,
			})

			previousResults := a.resultsBefore(plan, i)
			result, err := a.executeStep(ctx, query, step, previousResults)
			if err != nil {
				log.WithCtx(ctx).Errorf("步骤 %d 执行失败: %v", step.ID, err)
				result = step.Result
			}

			a.mu.Lock()
			a.executedSteps = append(a.executedSteps, *step)
			a.mu.Unlock()

			a.emitEvent(ctx, "step_result", map[string]any{
				"step_id": step.ID,
				"status":  step.Status,
				"result":  result,
				"tool":    step.ToolName,
			})
		}

		// 检查是否需要重规划
		if !a.checkNeedReplan(plan) || replanCount >= a.maxReplan {
			break
		}

		// 重规划：携带已完成步骤，只补未完成的
		replanCount++
		newPlan, err := a.planner.Replan(ctx, plan, a.GetExecutedSteps(), query)
		if err != nil {
			log.WithCtx(ctx).Warnf("重规划失败: %v", err)
			break
		}

		a.mu.Lock()
		a.currentPlan = newPlan
		a.mu.Unlock()

		a.emitEvent(ctx, "replanned", map[string]any{
			"plan": newPlan,
		})

		plan = newPlan
	}
	return plan, nil
}

// resultsBefore 返回计划中 index 之前各步骤的结果（供当前步骤作上下文）
func (a *PlanExecuteAgent) resultsBefore(plan *Plan, index int) []string {
	var results []string
	for i := 0; i < index; i++ {
		results = append(results, plan.Steps[i].Result)
	}
	return results
}

// buildExecutionSummary 基于所有已执行步骤构建执行结果摘要
// （跨重规划迭代累积，避免重规划后丢失前期检索证据）
func (a *PlanExecuteAgent) buildExecutionSummary() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	var summary strings.Builder
	summary.WriteString("## 执行结果摘要\n\n")
	if len(a.executedSteps) == 0 {
		summary.WriteString("（未执行任何步骤，请基于自身知识直接回答，不确定时明确说明）\n")
		return summary.String()
	}
	for _, step := range a.executedSteps {
		summary.WriteString(fmt.Sprintf("### 步骤 %d: %s\n", step.ID, step.Description))
		if step.ToolName != "" {
			summary.WriteString(fmt.Sprintf("- 工具: %s\n", step.ToolName))
		}
		if step.Reason != "" {
			summary.WriteString(fmt.Sprintf("- 原因: %s\n", step.Reason))
		}
		if step.Result != "" && step.Result != "无需调用工具" {
			summary.WriteString(fmt.Sprintf("- 结果: %s\n\n", truncateResult(step.Result, maxSummaryResultLen)))
		} else {
			summary.WriteString("- 结果: （无有效结果）\n\n")
		}
	}
	return summary.String()
}

// checkNeedReplan 检查执行结果是否值得重规划：
// 1. 存在失败步骤；2. 超过一半步骤的结果无效（空/未检索到）
func (a *PlanExecuteAgent) checkNeedReplan(plan *Plan) bool {
	if len(plan.Steps) == 0 {
		return false
	}
	for _, step := range plan.Steps {
		if step.Status == "failed" {
			return true
		}
	}
	useless := 0
	for _, step := range plan.Steps {
		if isUselessResult(step.Result) {
			useless++
		}
	}
	return useless > len(plan.Steps)/2
}

// isUselessResult 判断步骤结果是否无效（空或各工具的“无结果”提示）
func isUselessResult(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return true
	}
	for _, marker := range []string{"未检索到", "无结果", "未找到", "no results"} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// executeStep 执行单个计划步骤：
// 1) 优先直接调用计划指定工具（零额外 LLM 开销）
// 2) 直调失败（参数不合法/工具报错/未知工具）时降级到 ReAct 兜底
func (a *PlanExecuteAgent) executeStep(ctx context.Context, query string, step *PlanStep, previousResults []string) (string, error) {
	// 更新状态为运行中
	step.Status = "running"

	// 无工具步骤
	if step.ToolName == "none" || step.ToolName == "" {
		step.Status = "completed"
		step.Result = "无需调用工具"
		return step.Result, nil
	}

	// 1) 直接调用计划指定工具
	if t, ok := a.toolMap[step.ToolName]; ok {
		if result, err := invokeTool(ctx, t, step.ToolArgs); err == nil {
			step.Status = "completed"
			step.Result = truncateResult(result, maxStepResultLen)
			return step.Result, nil
		} else {
			log.WithCtx(ctx).Warnf("步骤 %d 直调 %s 失败: %v，降级 ReAct 兜底", step.ID, step.ToolName, err)
		}
	} else {
		// 未知工具：ReAct 兜底也没有该工具，直接失败以触发重规划
		log.WithCtx(ctx).Warnf("步骤 %d 计划了未知工具 %s，快速失败", step.ID, step.ToolName)
		step.Status = "failed"
		step.Result = fmt.Sprintf("工具 %s 不存在，请改用可用工具", step.ToolName)
		return step.Result, fmt.Errorf("未知工具: %s", step.ToolName)
	}

	// 2) ReAct 兜底（小步数）
	executionPrompt := a.buildExecutionPrompt(query, step, previousResults)
	response, err := a.executor.Generate(ctx, []*schema.Message{schema.UserMessage(executionPrompt)})
	if err != nil {
		step.Status = "failed"
		step.Result = fmt.Sprintf("执行失败: %v", err)
		return "", err
	}

	step.Status = "completed"
	step.Result = truncateResult(response.Content, maxStepResultLen)
	return step.Result, nil
}

// invokeTool 直接调用工具（要求实现 tool.InvokableTool）
func invokeTool(ctx context.Context, t tool.BaseTool, args string) (string, error) {
	if it, ok := t.(tool.InvokableTool); ok {
		return it.InvokableRun(ctx, args)
	}
	return "", fmt.Errorf("工具 %v 不支持直接调用", t)
}

// buildExecutionPrompt 构建 ReAct 兜底执行提示
func (a *PlanExecuteAgent) buildExecutionPrompt(query string, step *PlanStep, previousResults []string) string {
	var prompt strings.Builder

	prompt.WriteString(fmt.Sprintf("用户问题: %s\n\n", query))
	prompt.WriteString("请执行以下任务:\n\n")
	prompt.WriteString(fmt.Sprintf("任务描述: %s\n", step.Description))
	prompt.WriteString(fmt.Sprintf("推荐工具: %s\n", step.ToolName))

	if step.ToolArgs != "" && step.ToolArgs != "{}" {
		prompt.WriteString(fmt.Sprintf("推荐参数: %s\n", step.ToolArgs))
	}

	if step.Reason != "" {
		prompt.WriteString(fmt.Sprintf("执行原因: %s\n", step.Reason))
	}

	if len(previousResults) > 0 {
		prompt.WriteString("\n前序步骤结果（供参考）:\n")
		for i, r := range previousResults {
			runes := []rune(r)
			if len(runes) > 1000 {
				r = string(runes[:1000]) + "..."
			}
			prompt.WriteString(fmt.Sprintf("- 步骤 %d: %s\n", i+1, r))
		}
	}

	prompt.WriteString("\n请使用适当的工具完成这个任务，并提供详细的执行结果。")

	return prompt.String()
}

// getToolInfo 获取所有工具的信息
func (a *PlanExecuteAgent) getToolInfo(ctx context.Context) ([]*schema.ToolInfo, error) {
	var toolsInfo []*schema.ToolInfo
	for _, t := range a.tools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		toolsInfo = append(toolsInfo, info)
	}
	return toolsInfo, nil
}

// GetCurrentPlan 获取当前计划
func (a *PlanExecuteAgent) GetCurrentPlan() *Plan {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.currentPlan
}

// GetExecutedSteps 获取已执行的步骤
func (a *PlanExecuteAgent) GetExecutedSteps() []PlanStep {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.executedSteps
}

// generateFinalAnswer 生成最终答案
// history 为多轮对话历史（不含当前问题），可为 nil
func (a *PlanExecuteAgent) generateFinalAnswer(ctx context.Context, query string, history []*schema.Message) (string, error) {
	prompt := fmt.Sprintf(finalAnswerPromptTemplate, time.Now().Format("2006-01-02 15:04:05"), buildHistorySection(history), query, a.buildExecutionSummary())

	messages := []*schema.Message{
		schema.SystemMessage(finalAnswerSystemPrompt),
		schema.UserMessage(prompt),
	}

	response, err := a.chatModel.Generate(ctx, messages, qwen.WithExtraHeader(config.GetUA()))
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// streamFinalAnswer 流式生成最终答案
// history 为多轮对话历史（不含当前问题），可为 nil
func (a *PlanExecuteAgent) streamFinalAnswer(ctx context.Context, query string, history []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
	prompt := fmt.Sprintf(finalAnswerPromptTemplate, time.Now().Format("2006-01-02 15:04:05"), buildHistorySection(history), query, a.buildExecutionSummary())

	messages := []*schema.Message{
		schema.SystemMessage(finalAnswerSystemPrompt),
		schema.UserMessage(prompt),
	}

	return a.chatModel.Stream(ctx, messages)
}

// emitEvent 发送事件
func (a *PlanExecuteAgent) emitEvent(ctx context.Context, stage string, data map[string]any) {
	if a.messageHandler != nil {
		a.messageHandler(ctx, stage, data)
	}
}
