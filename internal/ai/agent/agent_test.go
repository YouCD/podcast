package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ==================== Mock 基础设施 ====================

// mockResponse 队列式 LLM 响应
type mockResponse struct {
	content string
	err     error
}

// mockChatModel 实现 model.ToolCallingChatModel，按队列顺序返回预设响应
type mockChatModel struct {
	mu        sync.Mutex
	responses []mockResponse
	calls     int
}

func newMockChatModel(responses ...string) *mockChatModel {
	q := make([]mockResponse, 0, len(responses))
	for _, r := range responses {
		q = append(q, mockResponse{content: r})
	}
	return &mockChatModel{responses: q}
}

func (m *mockChatModel) next() mockResponse {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if len(m.responses) == 0 {
		return mockResponse{err: errors.New("mockChatModel: 响应队列已空")}
	}
	r := m.responses[0]
	m.responses = m.responses[1:]
	return r
}

func (m *mockChatModel) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func (m *mockChatModel) Generate(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
	r := m.next()
	if r.err != nil {
		return nil, r.err
	}
	return &schema.Message{Role: schema.Assistant, Content: r.content}, nil
}

func (m *mockChatModel) Stream(_ context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	r := m.next()
	if r.err != nil {
		return nil, r.err
	}
	return schema.StreamReaderFromArray([]*schema.Message{{Role: schema.Assistant, Content: r.content}}), nil
}

func (m *mockChatModel) WithTools(_ []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

// mockTool 实现 tool.InvokableTool
type mockTool struct {
	name      string
	desc      string
	run       func(ctx context.Context, argsInJSON string) (string, error)
	mu        sync.Mutex
	callCount int
}

func newMockTool(name, desc string, run func(ctx context.Context, argsInJSON string) (string, error)) *mockTool {
	return &mockTool{name: name, desc: desc, run: run}
}

func (t *mockTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: t.name, Desc: t.desc}, nil
}

func (t *mockTool) InvokableRun(ctx context.Context, argsInJSON string, _ ...tool.Option) (string, error) {
	t.mu.Lock()
	t.callCount++
	t.mu.Unlock()
	// 与真实工具一致：先解析 JSON 参数，非法参数直接报错
	if !json.Valid([]byte(argsInJSON)) {
		return "", fmt.Errorf("invalid tool args: %s", argsInJSON)
	}
	return t.run(ctx, argsInJSON)
}

func (t *mockTool) CallCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.callCount
}

// newTestAgent 构建带 mock 工具的 PlanExecuteAgent
func newTestAgent(t *testing.T, cm *mockChatModel, tools ...tool.BaseTool) *PlanExecuteAgent {
	t.Helper()
	a, err := NewPlanExecuteAgent(&PlanExecuteAgentConfig{
		ChatModel: cm,
		Tools:     tools,
		MaxReplan: 3,
	})
	if err != nil {
		t.Fatalf("NewPlanExecuteAgent: %v", err)
	}
	return a
}

// ==================== 纯函数测试 ====================

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"纯JSON", `{"steps":[]}`, `{"steps":[]}`},
		{"中文包裹", `好的，计划如下：{"steps":[{"id":1}]}`, `{"steps":[{"id":1}]}`},
		{"无JSON", `没有JSON内容`, ``},
		{"嵌套花括号", `前缀 {"a":{"b":1}} 后缀`, `{"a":{"b":1}}`},
		{"非法JSON回退", `{"broken":`, ``},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := extractJSON(c.input); got != c.want {
				t.Errorf("extractJSON(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}

func TestParsePlanResponse_Fallback(t *testing.T) {
	p := NewPlanner(newMockChatModel(), nil)
	plan, err := p.parsePlanResponse("抱歉，我无法生成计划", "什么是量子计算")
	if err != nil {
		t.Fatalf("parsePlanResponse: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("期望 1 个回退步骤，得到 %d", len(plan.Steps))
	}
	step := plan.Steps[0]
	if step.ToolName != "pgvector_search" {
		t.Errorf("回退步骤工具应为 pgvector_search，实际 %q", step.ToolName)
	}
	if !strings.Contains(step.ToolArgs, "什么是量子计算") {
		t.Errorf("回退步骤参数应包含原始问题，实际 %q", step.ToolArgs)
	}
}

func TestParsePlanResponse_Valid(t *testing.T) {
	p := NewPlanner(newMockChatModel(), nil)
	resp := `{"steps":[{"id":1,"description":"查知识库","tool_name":"pgvector_search","tool_args":"{\"query\":\"x\"}"},{"id":2,"description":"查图谱","tool_name":"dgraph_query"}]}`
	plan, err := p.parsePlanResponse(resp, "test")
	if err != nil {
		t.Fatalf("parsePlanResponse: %v", err)
	}
	if len(plan.Steps) != 2 {
		t.Fatalf("期望 2 步，得到 %d", len(plan.Steps))
	}
	if plan.Steps[0].Status != "pending" || plan.Steps[1].Status != "pending" {
		t.Errorf("步骤状态应初始化为 pending")
	}
}

func TestParsePlanResponse_EmptySteps(t *testing.T) {
	p := NewPlanner(newMockChatModel(), nil)
	plan, err := p.parsePlanResponse(`{"steps": []}`, "你好")
	if err != nil {
		t.Fatalf("parsePlanResponse: %v", err)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("空计划应保留 0 步，得到 %d", len(plan.Steps))
	}
}

func TestBuildHistorySection(t *testing.T) {
	if got := buildHistorySection(nil); got != "（无）" {
		t.Errorf("无历史应返回（无），实际 %q", got)
	}
	history := []*schema.Message{
		schema.UserMessage("上上个问题"),
		&schema.Message{Role: schema.Assistant, Content: strings.Repeat("答", 600)}, // 应被截断到 500 rune
	}
	got := buildHistorySection(history)
	if !strings.Contains(got, "[用户] 上上个问题") {
		t.Errorf("应包含用户消息: %q", got)
	}
	if !strings.Contains(got, "[助手]") {
		t.Errorf("应包含助手消息: %q", got)
	}
	if !strings.HasSuffix(strings.Fields(got)[len(strings.Fields(got))-1], "...") && !strings.Contains(got, "...") {
		t.Errorf("长消息应被截断并带省略号: %q", got)
	}
}

func TestIsUselessResult(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"", true},
		{"   ", true},
		{"知识库中未检索到相关文档。建议更换关键词", true},
		{"无结果", true},
		{"未找到匹配项", true},
		{"no results found", true},
		{"正常结果内容", false},
	}
	for _, c := range cases {
		if got := isUselessResult(c.s); got != c.want {
			t.Errorf("isUselessResult(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestCheckNeedReplan(t *testing.T) {
	a := newTestAgent(t, newMockChatModel())

	// 空计划：不需要重规划
	if a.checkNeedReplan(&Plan{Steps: []PlanStep{}}) {
		t.Error("空计划不应触发重规划")
	}

	// 失败步骤：触发
	plan := &Plan{Steps: []PlanStep{{ID: 1, Status: "failed", Result: "执行失败: x"}}}
	if !a.checkNeedReplan(plan) {
		t.Error("失败步骤应触发重规划")
	}

	// 全部有效结果：不触发
	plan = &Plan{Steps: []PlanStep{
		{ID: 1, Status: "completed", Result: "结果A"},
		{ID: 2, Status: "completed", Result: "结果B"},
	}}
	if a.checkNeedReplan(plan) {
		t.Error("全部有效结果不应触发重规划")
	}

	// 超过一半无效：触发
	plan = &Plan{Steps: []PlanStep{
		{ID: 1, Status: "completed", Result: "结果A"},
		{ID: 2, Status: "completed", Result: ""},
		{ID: 3, Status: "completed", Result: "无结果"},
	}}
	if !a.checkNeedReplan(plan) {
		t.Error("超过一半无效结果应触发重规划")
	}

	// 单步骤且无效：触发
	plan = &Plan{Steps: []PlanStep{{ID: 1, Status: "completed", Result: "未检索到相关内容"}}}
	if !a.checkNeedReplan(plan) {
		t.Error("单步骤无效结果应触发重规划")
	}
}

func TestBuildExecutionPrompt(t *testing.T) {
	a := newTestAgent(t, newMockChatModel())
	step := &PlanStep{
		ID:          1,
		Description: "检索量子计算",
		ToolName:    "pgvector_search",
		ToolArgs:    `{"query":"量子计算"}`,
		Reason:      "查内部知识库",
	}
	prompt := a.buildExecutionPrompt("量子计算是什么", step, []string{strings.Repeat("前", 1200)})
	if !strings.Contains(prompt, "量子计算是什么") {
		t.Error("应包含原始用户问题")
	}
	if !strings.Contains(prompt, "pgvector_search") {
		t.Error("应包含推荐工具")
	}
	if !strings.Contains(prompt, `{"query":"量子计算"}`) {
		t.Error("应包含推荐参数")
	}
	if !strings.Contains(prompt, "...") {
		t.Error("前序结果超长应被截断")
	}
}

func TestInvokeTool(t *testing.T) {
	good := newMockTool("t", "d", func(ctx context.Context, args string) (string, error) { return "ok:" + args, nil })
	if r, err := invokeTool(context.Background(), good, `{"a":1}`); err != nil || r != `ok:{"a":1}` {
		t.Errorf("invokeTool(good) = %q, %v", r, err)
	}

	// 非 InvokableTool
	plain := &plainTool{}
	if _, err := invokeTool(context.Background(), plain, "{}"); err == nil {
		t.Error("非 InvokableTool 应返回错误")
	}
}

type plainTool struct{}

func (p *plainTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "plain"}, nil
}

// ==================== Planner 测试 ====================

func TestPlannerCreatePlan(t *testing.T) {
	cm := newMockChatModel(`{"steps":[{"id":1,"description":"检索","tool_name":"pgvector_search","tool_args":"{\"query\":\"x\"}","reason":"r"}]}`)
	p := NewPlanner(cm, nil)
	plan, err := p.CreatePlan(context.Background(), "x", nil, nil)
	if err != nil {
		t.Fatalf("CreatePlan: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].ToolName != "pgvector_search" {
		t.Errorf("计划解析错误: %+v", plan)
	}
	if cm.callCount() != 1 {
		t.Errorf("CreatePlan 应调用 LLM 1 次，实际 %d", cm.callCount())
	}
}

func TestPlannerReplanMergesCompleted(t *testing.T) {
	// 重规划响应：步骤 1（工具 a）+ 新步骤 3（工具 c）
	cm := newMockChatModel(`{"steps":[{"id":1,"description":"d1","tool_name":"a"},{"id":3,"description":"d3","tool_name":"c"}]}`)
	p := NewPlanner(cm, nil)

	currentPlan := &Plan{Steps: []PlanStep{
		{ID: 1, ToolName: "a", Status: "completed", Result: "R1"},
		{ID: 2, ToolName: "b", Status: "failed", Result: "执行失败"},
	}}

	newPlan, err := p.Replan(context.Background(), currentPlan, currentPlan.Steps, "query")
	if err != nil {
		t.Fatalf("Replan: %v", err)
	}
	if len(newPlan.Steps) != 2 {
		t.Fatalf("期望 2 步，得到 %d", len(newPlan.Steps))
	}
	// 步骤 1 已完成，应携带旧结果
	if newPlan.Steps[0].Status != "completed" || newPlan.Steps[0].Result != "R1" {
		t.Errorf("已完成步骤应携带旧状态/结果: %+v", newPlan.Steps[0])
	}
	// 步骤 3 为新步骤
	if newPlan.Steps[1].Status != "pending" {
		t.Errorf("新步骤应为 pending: %+v", newPlan.Steps[1])
	}
}

// ==================== PlanExecuteAgent 流程测试 ====================

const planOneStep = `{"steps":[{"id":1,"description":"d","tool_name":"%s","tool_args":"%s","reason":"r"}]}`

func TestPlanExecute_Run_DirectTool(t *testing.T) {
	tool := newMockTool("good_tool", "good", func(ctx context.Context, args string) (string, error) {
		return "TOOL_RESULT:" + args, nil
	})
	cm := newMockChatModel(
		fmt.Sprintf(planOneStep, "good_tool", `{\"q\":\"x\"}`),
		"FINAL_ANSWER",
	)
	a := newTestAgent(t, cm, tool)

	msg, err := a.Run(context.Background(), schema.UserMessage("test query"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if msg.Content != "FINAL_ANSWER" {
		t.Errorf("最终答案错误: %q", msg.Content)
	}
	if tool.CallCount() != 1 {
		t.Errorf("工具应直调 1 次，实际 %d", tool.CallCount())
	}
	// LLM 调用：规划 1 次 + 最终答案 1 次（直调不额外消耗 LLM）
	if cm.callCount() != 2 {
		t.Errorf("LLM 应调用 2 次（规划+最终答案），实际 %d", cm.callCount())
	}
	// 步骤状态
	steps := a.GetExecutedSteps()
	if len(steps) != 1 || steps[0].Status != "completed" {
		t.Errorf("步骤状态错误: %+v", steps)
	}
}

func TestPlanExecute_Run_ReActFallback(t *testing.T) {
	tool := newMockTool("good_tool", "good", func(ctx context.Context, args string) (string, error) {
		return "SHOULD_NOT_REACH", nil
	})
	cm := newMockChatModel(
		fmt.Sprintf(planOneStep, "good_tool", `{invalid json`), // 非法参数 → 直调失败
		"FALLBACK_RESULT", // ReAct 兜底响应
		"FINAL_ANSWER",
	)
	a := newTestAgent(t, cm, tool)

	msg, err := a.Run(context.Background(), schema.UserMessage("test"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if msg.Content != "FINAL_ANSWER" {
		t.Errorf("最终答案错误: %q", msg.Content)
	}
	// 直调因参数非法失败（工具仍被调用 1 次但报错）
	if tool.CallCount() != 1 {
		t.Errorf("直调应尝试 1 次，实际 %d", tool.CallCount())
	}
	// ReAct 兜底由 mock model 直接返回
	steps := a.GetExecutedSteps()
	if len(steps) != 1 || steps[0].Status != "completed" {
		t.Errorf("步骤应完成: %+v", steps)
	}
	if steps[0].Result != "FALLBACK_RESULT" {
		t.Errorf("步骤结果应来自 ReAct 兜底: %q", steps[0].Result)
	}
}

func TestPlanExecute_Run_Replan(t *testing.T) {
	badTool := newMockTool("bad_tool", "bad", func(ctx context.Context, args string) (string, error) {
		return "", errors.New("tool broken")
	})
	goodTool := newMockTool("good_tool", "good", func(ctx context.Context, args string) (string, error) {
		return "GOOD_RESULT", nil
	})

	cm := newMockChatModel(
		fmt.Sprintf(planOneStep, "bad_tool", `{}`), // 计划 A：坏工具
		"", // ReAct 兜底失败（见下方 error 注入）
		fmt.Sprintf(planOneStep, "good_tool", `{}`), // 重规划后的计划 B
		"FINAL_ANSWER",
	)
	// 第二次 LLM 调用（ReAct 兜底）返回错误，迫使步骤失败
	cm.mu.Lock()
	cm.responses[1] = mockResponse{err: errors.New("react model error")}
	cm.mu.Unlock()

	a := newTestAgent(t, cm, badTool, goodTool)

	msg, err := a.Run(context.Background(), schema.UserMessage("test"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if msg.Content != "FINAL_ANSWER" {
		t.Errorf("最终答案错误: %q", msg.Content)
	}
	if badTool.CallCount() != 1 {
		t.Errorf("坏工具应直调 1 次，实际 %d", badTool.CallCount())
	}
	if goodTool.CallCount() != 1 {
		t.Errorf("重规划后好工具应直调 1 次，实际 %d", goodTool.CallCount())
	}
	// LLM：规划 + ReAct兜底(失败) + 重规划 + 最终答案 = 4
	if cm.callCount() != 4 {
		t.Errorf("LLM 应调用 4 次，实际 %d", cm.callCount())
	}
}

func TestPlanExecute_Run_EmptyPlan(t *testing.T) {
	cm := newMockChatModel(`{"steps": []}`, "GREETING_ANSWER")
	tool := newMockTool("t", "d", func(ctx context.Context, args string) (string, error) { return "x", nil })
	a := newTestAgent(t, cm, tool)

	msg, err := a.Run(context.Background(), schema.UserMessage("你好"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if msg.Content != "GREETING_ANSWER" {
		t.Errorf("最终答案错误: %q", msg.Content)
	}
	if tool.CallCount() != 0 {
		t.Errorf("空计划不应调用工具，实际 %d", tool.CallCount())
	}
	if cm.callCount() != 2 {
		t.Errorf("LLM 应调用 2 次（规划+最终答案），实际 %d", cm.callCount())
	}
}

func TestPlanExecute_Run_UnknownToolFastFail(t *testing.T) {
	// 计划了不存在的工具 → 快速失败（不走 ReAct 兜底）→ 触发重规划
	cm := newMockChatModel(
		fmt.Sprintf(planOneStep, "ghost_tool", `{}`),
		`{"steps":[]}`, // 重规划：空计划，直接回答
		"FINAL",
	)
	a := newTestAgent(t, cm)

	msg, err := a.Run(context.Background(), schema.UserMessage("test"), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if msg.Content != "FINAL" {
		t.Errorf("最终答案错误: %q", msg.Content)
	}
	steps := a.GetExecutedSteps()
	if len(steps) != 1 || steps[0].Status != "failed" {
		t.Errorf("未知工具步骤应快速失败: %+v", steps)
	}
}

// ==================== PgVectorSearchTool 参数校验测试 ====================

func TestPgVectorSearchTool_ValidateArgs(t *testing.T) {
	tt := &PgVectorSearchTool{} // nil embedder：校验失败应在调用 embedder 前返回

	// 空 query
	if _, err := tt.InvokableRun(context.Background(), `{"top_k":5,"reason":"r"}`); err == nil {
		t.Error("空 query 应报错")
	}
	// 非法 JSON
	if _, err := tt.InvokableRun(context.Background(), `{broken`); err == nil {
		t.Error("非法 JSON 应报错")
	}
	// 非法日期
	if _, err := tt.InvokableRun(context.Background(), `{"query":"x","top_k":5,"start_date":"2026/13/99","reason":"r"}`); err == nil {
		t.Error("非法 start_date 应报错")
	}
	if _, err := tt.InvokableRun(context.Background(), `{"query":"x","top_k":5,"end_date":"bad","reason":"r"}`); err == nil {
		t.Error("非法 end_date 应报错")
	}
}

func TestParseDate(t *testing.T) {
	if _, err := parseDate("2026-01-15"); err != nil {
		t.Errorf("parseDate(2026-01-15): %v", err)
	}
	if _, err := parseDate("2026-01-15 10:00:00"); err != nil {
		t.Errorf("parseDate(2026-01-15 10:00:00): %v", err)
	}
	if _, err := parseDate("garbage"); err == nil {
		t.Error("parseDate(garbage) 应报错")
	}
}
