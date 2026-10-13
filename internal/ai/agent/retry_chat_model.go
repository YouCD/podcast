package agent

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"podcast/pkg"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

type retryChatModel struct {
	base           *qwen.ChatModel
	maxRetry       int
	backoff        func(attempt int) time.Duration
	llmInfo        *types.LLMInfo
	responseFormat openai.ChatCompletionResponseFormatType
}

func defaultBackoff(attempt int) time.Duration {
	// 简单指数退避 + jitter
	base := 500 * time.Millisecond
	d := base * time.Duration(1<<attempt)
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	return d + jitter
}

// IsRetryableError 判断是否值得重试的错误（主要是 429 / rate limit 相关）
func isRetryableRateLimit(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "429") ||
		strings.Contains(s, "rate limit") ||
		strings.Contains(s, "too many requests") ||
		strings.Contains(s, "quota exceeded") ||
		strings.Contains(s, "limit reached") ||
		strings.Contains(s, "requests per") // dashscope 常见
}

// NewRetryChatModel 使用指定模型创建带重试能力的 ChatModel（限流时指数退避重试同一模型）
func NewRetryChatModel(ctx context.Context, llmInfo *types.LLMInfo, maxRetry int, backoff func(int) time.Duration, responseFormat openai.ChatCompletionResponseFormatType) (*retryChatModel, error) {
	if maxRetry < 0 {
		maxRetry = 0
	}
	if backoff == nil {
		backoff = defaultBackoff
	}

	chatModel, err := pkg.NewChatModel(ctx, llmInfo, responseFormat, true)
	if err != nil {
		return nil, err
	}

	return &retryChatModel{
		base:           chatModel,
		maxRetry:       maxRetry,
		backoff:        backoff,
		llmInfo:        llmInfo,
		responseFormat: responseFormat,
	}, nil
}

func (r *retryChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	var lastErr error

	for attempt := 0; attempt <= r.maxRetry; attempt++ {
		// 每次重试都重新调用底层 Stream
		outStream, err := r.base.Stream(ctx, in, opts...)
		if err != nil {
			lastErr = err
			if !isRetryableRateLimit(err) {
				// 非限流错误，直接返回
				return nil, err
			}
			if attempt == r.maxRetry {
				// 达到最大重试次数
				return nil, fmt.Errorf("rate limit retry exhausted after %d attempts: %w", r.maxRetry, lastErr)
			}

			wait := r.backoff(attempt)
			log.WithCtx(ctx).Warnw("Rate limit, retrying stream...", "attempt", attempt+1, "wait", wait, "err", err)

			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		// 成功拿到 StreamReader，开始包装
		// 保留 qwen 原有的 ToolCall index 修复逻辑
		var lastIndex *int

		wrapped := schema.StreamReaderWithConvert(outStream, func(msg *schema.Message) (*schema.Message, error) {
			if len(msg.ToolCalls) > 0 {
				firstToolCall := msg.ToolCalls[0]

				if msg.ResponseMeta == nil || len(msg.ResponseMeta.FinishReason) == 0 {
					lastIndex = firstToolCall.Index
					return msg, nil
				}

				if firstToolCall.Index == nil && len(msg.ResponseMeta.FinishReason) != 0 {
					firstToolCall.Index = lastIndex
					msg.ToolCalls[0] = firstToolCall
				}
			}
			return msg, nil
		})

		return wrapped, nil
	}

	// 理论上不会走到这里
	return nil, fmt.Errorf("unexpected: retry loop finished without return: %w", lastErr)
}

func (r *retryChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetry; attempt++ {
		msg, err := r.base.Generate(ctx, in, opts...)
		if err == nil {
			return msg, nil
		}
		lastErr = err

		if !isRetryableRateLimit(err) {
			return nil, err
		}
		if attempt == r.maxRetry {
			return nil, fmt.Errorf("rate limit retry exhausted: %w", lastErr)
		}

		wait := r.backoff(attempt)
		select {
		case <-time.After(wait):
			log.WithCtx(ctx).Warnw("Retrying Generate", "attempt", attempt, "err", err)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return nil, lastErr
}

func (r *retryChatModel) BindTools(tools []*schema.ToolInfo) error {
	return r.base.BindTools(tools)
}

func (r *retryChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	// 先调用 base 的 WithTools，得到新的 cli
	newBase, err := r.base.WithTools(tools)
	if err != nil {
		return nil, err
	}

	// newBase 是 *qwen.ChatModel 类型，再包装一次 retry 层
	return &retryChatModel{
		base:           newBase.(*qwen.ChatModel), // 类型断言，假设能成功
		maxRetry:       r.maxRetry,
		backoff:        r.backoff,
		llmInfo:        r.llmInfo,
		responseFormat: r.responseFormat,
	}, nil
}

func (r *retryChatModel) GetType() string {
	return r.base.GetType()
}

func (r *retryChatModel) GetModelName() string {
	if namer, ok := any(r.base).(interface{ GetModelName() string }); ok {
		return namer.GetModelName()
	}
	return "unknown-retry"
}
