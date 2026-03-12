package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// InvokableTool 可调用工具接口
type InvokableTool interface {
	Info(ctx context.Context) (*schema.ToolInfo, error)
	InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error)
}

// Executor 执行器，负责执行计划中的各个步骤
type Executor struct {
	tools map[string]InvokableTool
}

// NewExecutor 创建执行器
func NewExecutor(tools []tool.BaseTool) *Executor {
	toolMap := make(map[string]InvokableTool)
	for _, t := range tools {
		// 尝试转换为 InvokableTool
		if invokable, ok := t.(InvokableTool); ok {
			info, err := invokable.Info(context.Background())
			if err != nil {
				log.WithCtx(context.Background()).Warnw("获取工具信息失败", "error", err)
				continue
			}
			toolMap[info.Name] = invokable
		}
	}
	return &Executor{
		tools: toolMap,
	}
}

// ExecuteStep 执行计划中的单个步骤
func (e *Executor) ExecuteStep(ctx context.Context, step *PlanStep) (string, error) {
	// 更新状态为运行中
	step.Status = "running"

	// 检查是否是无工具步骤
	if step.ToolName == "none" || step.ToolName == "" {
		step.Status = "completed"
		step.Result = "无需调用工具"
		return step.Result, nil
	}

	// 获取工具
	t, ok := e.tools[step.ToolName]
	if !ok {
		err := fmt.Errorf("工具 %s 不存在", step.ToolName)
		step.Status = "failed"
		step.Result = err.Error()
		return "", err
	}

	// 执行工具
	result, err := t.InvokableRun(ctx, step.ToolArgs)
	if err != nil {
		step.Status = "failed"
		step.Result = fmt.Sprintf("执行失败: %v", err)
		return "", err
	}

	// 更新状态
	step.Status = "completed"
	step.Result = result

	return result, nil
}

// ExecutePlan 执行完整计划
func (e *Executor) ExecutePlan(ctx context.Context, plan *Plan, callback func(step *PlanStep, result string)) error {
	for i := range plan.Steps {
		step := &plan.Steps[i]
		plan.CurrentStep = i

		result, err := e.ExecuteStep(ctx, step)
		if err != nil {
			log.WithCtx(ctx).Errorf("步骤 %d 执行失败: %v", step.ID, err)
			// 可以选择继续执行或停止
			// 这里选择继续执行后续步骤
		}

		// 回调通知
		if callback != nil {
			callback(step, result)
		}
	}

	plan.IsComplete = true
	return nil
}

// GetToolInfo 获取所有工具的信息
func (e *Executor) GetToolInfo(ctx context.Context) ([]*schema.ToolInfo, error) {
	var toolsInfo []*schema.ToolInfo
	for _, t := range e.tools {
		info, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}
		toolsInfo = append(toolsInfo, info)
	}
	return toolsInfo, nil
}

// ValidatePlan 验证计划中的工具是否可用
func (e *Executor) ValidatePlan(plan *Plan) error {
	for _, step := range plan.Steps {
		if step.ToolName == "none" || step.ToolName == "" {
			continue
		}
		if _, ok := e.tools[step.ToolName]; !ok {
			return fmt.Errorf("计划中引用的工具 '%s' 不存在", step.ToolName)
		}
	}
	return nil
}

// ParseToolArgs 解析工具参数
func ParseToolArgs(argsJSON string) (map[string]interface{}, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil, fmt.Errorf("解析工具参数失败: %w", err)
	}
	return args, nil
}
