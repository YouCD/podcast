package common

import (
	"context"
	"strings"
	"time"

	"podcast/internal/ai/llm"
	"podcast/pkg"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

func ModelGenerate(ctx context.Context, model model.ToolCallingChatModel, input []*schema.Message) (string, error) {
	msg, err := model.Generate(ctx, input)
	if err != nil {
		return "", err
	}
	return msg.Content, nil
}

func RunModelGenerate(ctx context.Context, msgName string, input []*schema.Message, responseFormat openai.ChatCompletionResponseFormatType, attemptTotal int) (string, *types.LLMInfo, error) {
	var llm_info *types.LLMInfo
	var lastErr error
	var llmResult string
	var success bool

	pool := llm.GetLLMPool()

	for attempt := 0; attempt < attemptTotal; attempt++ {
		// 获取 LLM 实例
		llmInfo, err := pool.Get(ctx)
		if err != nil {
			lastErr = err
			log.WithCtx(ctx).Warnw("failed to get llm from pool", "error", err)
			continue // 池子空了，稍等重试
		}
		llm_info = llmInfo
		if attempt > 0 {
			// 指数退避：2s, 4s, 8s
			backoff := time.Duration(1<<attempt) * time.Second
			log.WithCtx(ctx).Infow(msgName,
				"provider", llmInfo,
				"attempt", attempt+1,
				"backoff", backoff)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				lastErr = ctx.Err()
				goto Done // 超时退出
			}
		}

		chatModel, err := pkg.NewChatModel(ctx, llmInfo, responseFormat)
		if err != nil {
			pool.Put(ctx, llmInfo)
			lastErr = err
			continue
		}

		msg, err := chatModel.Generate(ctx, input)
		// 成功
		if err == nil {
			success = true
			pool.Put(ctx, llmInfo)
			llmResult = strings.TrimSpace(msg.Content)
			break
		}

		pool.Put(ctx, llmInfo) // 归还实例（即使出错）

		log.WithCtx(ctx).Errorw(msgName,
			"provider", llmInfo,
			"attempt", attempt+1,
			"err", err,
		)

		// 标记该 provider 进入冷却期
		pool.MarkRateLimited(ctx, llmInfo)
		lastErr = err
		continue // 继续循环，会自动获取其他 provider
	}

Done:
	if success {
		return llmResult, llm_info, nil // 始终返回 nil，单个失败不阻塞整体流程
	}
	return "", llm_info, lastErr
}
