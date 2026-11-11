package agent

import (
	"strings"
	"testing"

	"podcast/pkg/types"

	"github.com/cloudwego/eino/schema"
)

func testToolsInfo() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{Name: "get_current_time", Desc: "获取当前时间"},
		{Name: "pgvector_search", Desc: "语义检索内部知识库"},
		{Name: "web_search_bailian", Desc: "外部网络搜索工具（MCP 下发描述）"},
		{Name: "dgraph_query", Desc: "查询知识图谱关系"},
	}
}

func testStrategySrc() *types.ToolStrategySource {
	return &types.ToolStrategySource{
		Internal: &types.AgentToolsConfig{
			Tools: []types.ToolStrategy{
				{Name: "pgvector_search", Summary: "内部文档库，权重最高", Strategy: "优先使用语义检索"},
				{Name: "dgraph_query", Summary: "知识图谱", Strategy: "关系查询时使用"},
			},
		},
		MCPProxy: map[string]*types.Mcp{
			"web_search": {URL: "http://x", Strategy: "内部库无果时使用外部搜索"},
		},
	}
}

func TestBuildToolsDescription_Dynamic(t *testing.T) {
	p := &Planner{strategySrc: testStrategySrc()}
	desc := p.buildToolsDescription(testToolsInfo())

	// 有策略的工具带详细策略
	if !strings.Contains(desc, "### pgvector_search") || !strings.Contains(desc, "优先使用语义检索") {
		t.Fatalf("pgvector_search 应带策略:\n%s", desc)
	}
	// 外部工具策略来自 mcpProxy
	if !strings.Contains(desc, "### web_search_bailian") || !strings.Contains(desc, "内部库无果时使用外部搜索") {
		t.Fatalf("外部工具应带 mcpProxy 策略:\n%s", desc)
	}
	// 无策略的工具仅基础描述
	if !strings.Contains(desc, "- **get_current_time**: 获取当前时间") {
		t.Fatalf("无策略工具应仅基础描述:\n%s", desc)
	}
}

func TestBuildToolsDescription_NoStrategy(t *testing.T) {
	p := &Planner{strategySrc: nil}
	desc := p.buildToolsDescription(testToolsInfo())

	// 无策略来源时全部为基础描述
	for _, name := range []string{"get_current_time", "pgvector_search", "web_search_bailian", "dgraph_query"} {
		if !strings.Contains(desc, "- **"+name+"**:") {
			t.Fatalf("工具 %s 应有基础描述:\n%s", name, desc)
		}
	}
	if strings.Contains(desc, "###") {
		t.Fatalf("无策略来源不应有 ### 段落:\n%s", desc)
	}
}

func TestBuildToolsDescription_Ordering(t *testing.T) {
	p := &Planner{strategySrc: testStrategySrc()}
	desc := p.buildToolsDescription(testToolsInfo())

	// 内部工具按配置顺序在前（pgvector_search 先于 dgraph_query），其余按注册顺序在后
	pgIdx := strings.Index(desc, "### pgvector_search")
	dgIdx := strings.Index(desc, "### dgraph_query")
	webIdx := strings.Index(desc, "### web_search_bailian")
	ctIdx := strings.Index(desc, "get_current_time")
	if pgIdx < 0 || dgIdx < 0 || webIdx < 0 || ctIdx < 0 {
		t.Fatalf("缺少工具段落:\n%s", desc)
	}
	// 注册序：get_current_time(0) < web_search_bailian(2)，故其余段为 get_current_time 先于 web_search_bailian
	if !(pgIdx < dgIdx) || !(dgIdx < ctIdx) || !(ctIdx < webIdx) {
		t.Fatalf("顺序错误（内部配置序 → 其余注册序）:\n%s", desc)
	}
}

func TestBuildPrioritySection(t *testing.T) {
	p := &Planner{strategySrc: testStrategySrc()}
	section := p.buildPrioritySection(testToolsInfo())

	// 仅包含有策略的工具，内部用 summary，外部用截断描述
	if !strings.Contains(section, "1. **pgvector_search**：内部文档库，权重最高") {
		t.Fatalf("优先级列表错误:\n%s", section)
	}
	if !strings.Contains(section, "2. **dgraph_query**：知识图谱") {
		t.Fatalf("优先级列表错误:\n%s", section)
	}
	if !strings.Contains(section, "web_search_bailian") || !strings.Contains(section, "外部网络搜索工具") {
		t.Fatalf("外部工具应使用自带描述:\n%s", section)
	}
	// 无策略工具不进入优先级列表
	if strings.Contains(section, "get_current_time") {
		t.Fatalf("无策略工具不应进入优先级列表:\n%s", section)
	}
}

func TestBuildPrioritySection_Empty(t *testing.T) {
	p := &Planner{strategySrc: nil}
	section := p.buildPrioritySection(testToolsInfo())
	if section != "（无特殊优先级，按工具描述使用）" {
		t.Fatalf("空策略来源应返回占位文本, got %q", section)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("hello", 10); got != "hello" {
		t.Fatalf("短串不应截断: %q", got)
	}
	if got := truncateRunes("中文测试字符串", 4); got != "中文测试..." {
		t.Fatalf("中文截断错误: %q", got)
	}
	// 边界：不截断多字节字符
	if got := truncateRunes("abcdef", 3); got != "abc..." {
		t.Fatalf("截断错误: %q", got)
	}
}
