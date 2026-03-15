package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// PlanExecuteAgent Plan-and-Execute 模式的 Agent
// 使用 react.Agent 作为执行器
type PlanExecuteAgent struct {
	planner   *Planner
	executor  *react.Agent // 使用 eino 框架的 react.Agent 作为执行器
	chatModel model.ToolCallingChatModel
	tools     []tool.BaseTool

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
}

// NewPlanExecuteAgent 创建 Plan-Execute Agent
func NewPlanExecuteAgent(config *PlanExecuteAgentConfig) (*PlanExecuteAgent, error) {
	if config.MaxReplan == 0 {
		config.MaxReplan = 3
	}

	// 创建 react.Agent 作为执行器
	executor, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: config.ChatModel,
		ToolsConfig:      compose.ToolsNodeConfig{Tools: config.Tools},
		MaxStep:          20,
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
		return nil, fmt.Errorf("创建 react.Agent 执行器失败: %w", err)
	}

	return &PlanExecuteAgent{
		planner:        NewPlanner(config.ChatModel),
		executor:       executor,
		chatModel:      config.ChatModel,
		tools:          config.Tools,
		messageHandler: config.MessageHandle,
	}, nil
}

// Run 执行 Plan-Execute 流程
func (a *PlanExecuteAgent) Run(ctx context.Context, input *schema.Message) (*schema.Message, error) {
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
	plan, err := a.planner.CreatePlan(ctx, query, toolsInfo)
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

	// 3. 执行计划 - 使用 react.Agent 执行每个步骤
	var allResults []string
	for i := range plan.Steps {
		step := &plan.Steps[i]

		// 通知步骤开始
		a.emitEvent(ctx, "step_start", map[string]any{
			"step":  step,
			"index": i,
		})

		result, err := a.executeStepWithReAct(ctx, step)
		if err != nil {
			log.WithCtx(ctx).Errorf("步骤 %d 执行失败: %v", step.ID, err)
			// 记录失败但继续执行
		}

		a.mu.Lock()
		a.executedSteps = append(a.executedSteps, *step)
		a.mu.Unlock()

		allResults = append(allResults, result)

		// 通知步骤完成
		a.emitEvent(ctx, "step_complete", map[string]any{
			"step":   step,
			"result": result,
			"index":  i,
		})
	}

	// 4. 生成最终答案
	finalAnswer, err := a.generateFinalAnswer(ctx, query, plan, allResults)
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
func (a *PlanExecuteAgent) Stream(ctx context.Context, input *schema.Message) (*schema.StreamReader[*schema.Message], error) {
	query := input.Content

	// 获取工具信息
	toolsInfo, err := a.getToolInfo(ctx)
	if err != nil {
		return nil, err
	}

	// 创建计划
	plan, err := a.planner.CreatePlan(ctx, query, toolsInfo)
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

	// 执行计划
	var allResults []string
	for i := range plan.Steps {
		step := &plan.Steps[i]

		// 发送步骤开始
		if a.messageHandler != nil {
			a.messageHandler(ctx, "step_start", map[string]any{
				"step_id":     step.ID,
				"description": step.Description,
				"tool_name":   step.ToolName,
				"reason":      step.Reason,
			})
		}

		result, err := a.executeStepWithReAct(ctx, step)
		if err != nil {
			result = fmt.Sprintf("执行失败: %v", err)
		}

		a.mu.Lock()
		a.executedSteps = append(a.executedSteps, *step)
		a.mu.Unlock()

		allResults = append(allResults, result)

		// 发送步骤结果
		if a.messageHandler != nil {
			var args interface{}
			if step.ToolArgs != "" {
				err := json.Unmarshal([]byte(step.ToolArgs), &args)
				if err != nil {
					args = step.ToolArgs
				}
			}
			a.messageHandler(ctx, "step_result", map[string]any{
				"step_id":   step.ID,
				"status":    step.Status,
				"result":    result,
				"tool_args": args,
			})
		}
	}

	// 生成最终答案（流式）
	return a.streamFinalAnswer(ctx, query, plan, allResults)
}

// executeStepWithReAct 使用 react.Agent 执行单个步骤
func (a *PlanExecuteAgent) executeStepWithReAct(ctx context.Context, step *PlanStep) (string, error) {
	// 更新状态为运行中
	step.Status = "running"

	// 检查是否是无工具步骤
	if step.ToolName == "none" || step.ToolName == "" {
		step.Status = "completed"
		step.Result = "无需调用工具"
		return step.Result, nil
	}

	// 构建执行提示，让 react.Agent 理解当前任务
	executionPrompt := a.buildExecutionPrompt(step)

	// 使用 react.Agent 执行
	msg := schema.UserMessage(executionPrompt)
	response, err := a.executor.Generate(ctx, []*schema.Message{msg})
	if err != nil {
		step.Status = "failed"
		step.Result = fmt.Sprintf("执行失败: %v", err)
		return "", err
	}

	// 更新状态
	step.Status = "completed"
	step.Result = response.Content

	return response.Content, nil
}

// buildExecutionPrompt 构建执行提示
func (a *PlanExecuteAgent) buildExecutionPrompt(step *PlanStep) string {
	var prompt strings.Builder

	prompt.WriteString("请执行以下任务:\n\n")
	prompt.WriteString(fmt.Sprintf("任务描述: %s\n", step.Description))
	prompt.WriteString(fmt.Sprintf("推荐工具: %s\n", step.ToolName))

	if step.ToolArgs != "" && step.ToolArgs != "{}" {
		prompt.WriteString(fmt.Sprintf("推荐参数: %s\n", step.ToolArgs))
	}

	if step.Reason != "" {
		prompt.WriteString(fmt.Sprintf("执行原因: %s\n", step.Reason))
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

// PlanExecuteAgentWithReplan 支持重规划的 Agent
type PlanExecuteAgentWithReplan struct {
	*PlanExecuteAgent
	maxReplan int
}

// NewPlanExecuteAgentWithReplan 创建支持重规划的 Agent
func NewPlanExecuteAgentWithReplan(config *PlanExecuteAgentConfig, maxReplan int) (*PlanExecuteAgentWithReplan, error) {
	agent, err := NewPlanExecuteAgent(config)
	if err != nil {
		return nil, err
	}
	return &PlanExecuteAgentWithReplan{
		PlanExecuteAgent: agent,
		maxReplan:        maxReplan,
	}, nil
}

// ExecuteWithReplan 执行并支持重规划
func (a *PlanExecuteAgentWithReplan) ExecuteWithReplan(ctx context.Context, query string) (string, error) {
	toolsInfo, err := a.getToolInfo(ctx)
	if err != nil {
		return "", err
	}

	plan, err := a.planner.CreatePlan(ctx, query, toolsInfo)
	if err != nil {
		return "", err
	}

	var allResults []string
	replanCount := 0

	for replanCount <= a.maxReplan {
		allResults = nil

		// 执行当前计划
		for i := range plan.Steps {
			if plan.Steps[i].Status == "completed" {
				continue // 跳过已完成的步骤
			}

			result, err := a.executeStepWithReAct(ctx, &plan.Steps[i])
			if err != nil {
				result = fmt.Sprintf("执行失败: %v", err)
			}
			allResults = append(allResults, result)
		}

		// 检查是否需要重规划
		needReplan := a.checkNeedReplan(plan, allResults)
		if !needReplan || replanCount >= a.maxReplan {
			break
		}

		// 重规划
		replanCount++
		newPlan, err := a.planner.Replan(ctx, plan, a.executedSteps, query)
		if err != nil {
			log.WithCtx(ctx).Warnf("重规划失败: %v", err)
			break
		}
		plan = newPlan
	}

	// 生成最终答案
	return a.generateFinalAnswer(ctx, query, plan, allResults)
}

// StreamWithReplan 流式执行并支持重规划
func (a *PlanExecuteAgentWithReplan) StreamWithReplan(ctx context.Context, query string) (*schema.StreamReader[*schema.Message], error) {
	// 简化版本：先执行，再流式返回结果
	result, err := a.ExecuteWithReplan(ctx, query)
	if err != nil {
		return nil, err
	}

	// 创建一个包含结果的 StreamReader
	msg := schema.AssistantMessage(result, nil)
	stream := schema.StreamReaderFromArray([]*schema.Message{msg})
	return stream, nil
}

// checkNeedReplan 检查是否需要重规划
func (a *PlanExecuteAgentWithReplan) checkNeedReplan(plan *Plan, results []string) bool {
	// 检查是否有失败的步骤
	for _, step := range plan.Steps {
		if step.Status == "failed" {
			return true
		}
	}

	// 检查结果是否为空或无效
	emptyCount := 0
	for _, result := range results {
		if result == "" || result == "无结果" {
			emptyCount++
		}
	}

	// 如果超过一半的结果为空，可能需要重规划
	return emptyCount > len(results)/2
}

// generateFinalAnswer 生成最终答案
func (a *PlanExecuteAgent) generateFinalAnswer(ctx context.Context, query string, plan *Plan, results []string) (string, error) {
	// 构建执行摘要
	var summary strings.Builder
	summary.WriteString("## 执行计划摘要\n\n")
	for i, step := range plan.Steps {
		summary.WriteString(fmt.Sprintf("### 步骤 %d: %s\n", step.ID, step.Description))
		summary.WriteString(fmt.Sprintf("- 工具: %s\n", step.ToolName))
		summary.WriteString(fmt.Sprintf("- 原因: %s\n", step.Reason))
		if i < len(results) {
			summary.WriteString(fmt.Sprintf("- 结果: %s\n\n", results[i]))
		}
	}

	// 构建最终答案生成提示
	prompt := fmt.Sprintf(finalAnswerPromptTemplate, query, summary.String())

	messages := []*schema.Message{
		schema.SystemMessage(finalAnswerSystemPrompt),
		schema.UserMessage(prompt),
	}

	response, err := a.chatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return response.Content, nil
}

// streamFinalAnswer 流式生成最终答案
func (a *PlanExecuteAgent) streamFinalAnswer(ctx context.Context, query string, plan *Plan, results []string) (*schema.StreamReader[*schema.Message], error) {
	// 构建执行摘要
	var summary strings.Builder
	summary.WriteString("## 执行计划摘要\n\n")
	for i, step := range plan.Steps {
		summary.WriteString(fmt.Sprintf("### 步骤 %d: %s\n", step.ID, step.Description))
		summary.WriteString(fmt.Sprintf("- 工具: %s\n", step.ToolName))
		summary.WriteString(fmt.Sprintf("- 原因: %s\n", step.Reason))
		if i < len(results) {
			summary.WriteString(fmt.Sprintf("- 结果: %s\n\n", results[i]))
		}
	}

	prompt := fmt.Sprintf(finalAnswerPromptTemplate, query, summary.String())

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
