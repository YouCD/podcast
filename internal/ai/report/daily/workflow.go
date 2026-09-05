package daily

import (
	"context"
	"fmt"
	"time"

	"podcast/internal/ai/llm"
	"podcast/internal/database/models"
	"podcast/pkg/types"

	"github.com/cloudwego/eino/compose"
	"github.com/youcd/toolkit/log"
)

// graphState 全局状态（用于节点间共享数据）
type graphState struct {
	Report      *models.Report
	startDate   time.Time
	endDate     time.Time
	rssContents []*models.RssContent
	isNewReport bool
	llmPool     *llm.LLMPool
}

func init() {
	// 注册 graphState 的合并函数
	// 由于所有节点共享同一个 state 对象，合并时只需返回其中一个即可
	compose.RegisterValuesMergeFunc(func(states []*graphState) (*graphState, error) {
		if len(states) == 0 {
			return nil, fmt.Errorf("no states to merge")
		}
		// 返回第一个 state（所有 state 实际上是同一个对象）
		return states[0], nil
	})
}

func buildDailyWorkflow(cfg *types.Podcast, llmPool *llm.LLMPool) *compose.Graph[int, *graphState] {
	graph := compose.NewGraph[int, *graphState]()

	// 节点 1: 创建报告对象
	_ = graph.AddLambdaNode("create_report", compose.InvokableLambda(func(ctx context.Context, reportID int) (*graphState, error) {
		state, err := createReport(ctx, reportID)
		if err != nil {
			return nil, err
		}
		state.llmPool = llmPool
		return state, nil
	}))

	// 节点 2: 获取报告所需的所有Rss内容
	_ = graph.AddLambdaNode("rss_contents", compose.InvokableLambda(rssContents))

	// 节点 3: 生成内容摘要
	_ = graph.AddLambdaNode("generate_content", compose.InvokableLambda(generateContent))

	// 节点 4: 生成Html内容
	_ = graph.AddLambdaNode("generate_html", compose.InvokableLambda(generateHTML))

	// 节点 5: 生成Podcast播客内容
	_ = graph.AddLambdaNode("generate_podcast_content", compose.InvokableLambda(generatePodcastContent))

	// 节点 6: 生成Podcast播客音频
	_ = graph.AddLambdaNode("generate_podcast_video", compose.InvokableLambda(func(ctx context.Context, state *graphState) (*graphState, error) {
		return generatePodcastVideo(ctx, cfg, state)
	}))

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

	// 7. generate_html 和 generate_podcast_video 都连接到 save_report
	// save_report 会等待两个分支都完成后执行
	// 即使播客生成失败（节点内部处理错误返回 nil），工作流也能继续执行到 save_report
	_ = graph.AddEdge("generate_html", "save_report")
	_ = graph.AddEdge("generate_podcast_video", "save_report")
	_ = graph.AddEdge("save_report", compose.END)

	return graph
}

func New(ctx context.Context, cfg *types.Podcast, llmPool *llm.LLMPool) (compose.Runnable[int, *graphState], error) {
	// 构建工作流
	workflow := buildDailyWorkflow(cfg, llmPool)
	// 编译（启用流式处理和回调）
	// 使用 AllPredecessor 模式，确保 save_report 等待所有前驱节点完成后再执行
	runnable, err := workflow.Compile(ctx, compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return nil, fmt.Errorf("编译工作流失败: %w", err)
	}

	return runnable, nil
}
