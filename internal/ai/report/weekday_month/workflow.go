package weekday_month

import (
	"context"
	"fmt"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"
	"podcast/pkg/types"

	"github.com/cloudwego/eino/compose"
	"github.com/youcd/toolkit/log"
)

var (
	keyInfoDao                      = dao.NewKeyInfoDao(models.GetDb())
	rssContentDao dao.RssContentDao = dao.NewRssContentDao(models.GetDb())
)

// graphState 全局状态（用于节点间共享数据）
type graphState struct {
	userQueryStr string
	userQueryRaw string
	userQuery    *types.UserQuery
	contents     []*content
	llmResult    string
}

func buildWeekdayMonthWorkflow() *compose.Graph[string, *graphState] {
	graph := compose.NewGraph[string, *graphState]()

	// 节点 1: 重写用户查询
	_ = graph.AddLambdaNode("create_report", compose.InvokableLambda(rewriteUserQuery))

	// 节点 2: 获取RSS内容
	_ = graph.AddLambdaNode("get_rss_content", compose.InvokableLambda(getRssContent))

	// 节点 3: 分析
	_ = graph.AddLambdaNode("analysis", compose.InvokableLambda(analysis))
	_ = graph.AddLambdaNode("save", compose.InvokableLambda(save))

	_ = graph.AddEdge(compose.START, "create_report")
	_ = graph.AddEdge("create_report", "get_rss_content")
	_ = graph.AddEdge("get_rss_content", "analysis")
	_ = graph.AddEdge("analysis", "save")
	_ = graph.AddEdge("save", compose.END)

	return graph
}
func New(ctx context.Context) (compose.Runnable[string, *graphState], error) {
	// 构建工作流
	workflow := buildWeekdayMonthWorkflow()
	// 编译（启用流式处理和回调）
	runnable, err := workflow.Compile(ctx)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return nil, fmt.Errorf("编译工作流失败: %w", err)
	}

	return runnable, nil
}
