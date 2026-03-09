package daily

import (
	"context"
	"fmt"
	"podcast/internal/database/models"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/youcd/toolkit/log"
)

// graphState 全局状态（用于节点间共享数据）
type graphState struct {
	report      *models.Report
	startDate   time.Time
	endDate     time.Time
	rssContents []*models.RssContent
	isNewReport bool
}

func buildDailyWorkflow() *compose.Graph[int, *graphState] {
	graph := compose.NewGraph[int, *graphState]()

	// 节点 1: 创建报告对象
	_ = graph.AddLambdaNode("create_report", compose.InvokableLambda(createReport))

	// 节点 2: 获取报告所需的所有Rss内容
	_ = graph.AddLambdaNode("rss_contents", compose.InvokableLambda(rssContents))

	// 节点 3: 生成内容摘要
	_ = graph.AddLambdaNode("generate_content", compose.InvokableLambda(generateContent))

	// 节点 4: 生成Html内容
	_ = graph.AddLambdaNode("generate_html", compose.InvokableLambda(generateHTML))

	// 节点 5: 生成Podcast播客内容
	_ = graph.AddLambdaNode("generate_podcast_content", compose.InvokableLambda(generatePodcastContent))

	// 节点 6: 生成Podcast播客音频
	_ = graph.AddLambdaNode("generate_podcast_video", compose.InvokableLambda(generatePodcastVideo))

	// 节点 7: 保存report
	_ = graph.AddLambdaNode("save_report", compose.InvokableLambda(saveReport))

	// 编排
	// 【关键】定义节点间的边（依赖关系）
	// 1. create_report -> rss_contents
	_ = graph.AddEdge(compose.START, "create_report")
	// 2. create_report -> rss_contents
	_ = graph.AddEdge("create_report", "rss_contents")

	// 这两条边定义了 generate_markdown 和 generate_podcast 都依赖于 rss_contents
	// 当 rss_contents 完成后，这两个节点就可以并发执行了。
	// 3. rss_contents -> generate_markdown
	_ = graph.AddEdge("rss_contents", "generate_content")
	// 4. rss_contents -> generate_podcast

	// 5. generate_markdown -> generate_html
	_ = graph.AddEdge("generate_content", "generate_html")
	_ = graph.AddEdge("generate_content", "generate_podcast_content")
	// 6. generate_podcast_content -> generate_podcast_video
	_ = graph.AddEdge("generate_podcast_content", "generate_podcast_video")

	// 7. generate_podcast_video -> save_report
	_ = graph.AddEdge("generate_podcast_video", "save_report")
	_ = graph.AddEdge("save_report", compose.END)

	return graph
}
func New(ctx context.Context) (compose.Runnable[int, *graphState], error) {
	// 构建工作流
	workflow := buildDailyWorkflow()
	// 编译（启用流式处理和回调）
	runnable, err := workflow.Compile(ctx)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return nil, fmt.Errorf("编译工作流失败: %w", err)
	}

	return runnable, nil
}
