package common

import (
	"context"
	"fmt"
	"podcast/config"
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

func ModelGenerate(ctx context.Context, model model.ToolCallingChatModel, input []*schema.Message) (string, error) {
	msg, err := model.Generate(ctx, input)
	if err != nil {
		return "", err
	}
	return msg.Content, nil
}

func RunModelGenerate(ctx context.Context, llmInfo *types.LLMInfo, msgName string, input []*schema.Message, responseFormat openai.ChatCompletionResponseFormatType, attemptTotal int, timeout time.Duration, enableThinking bool) (string, *types.LLMInfo, error) {
	var lastErr error
	var llmResult string
	var success bool

	for attempt := 0; attempt < attemptTotal; attempt++ {
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

		chatModel, err := pkg.NewChatModel(ctx, llmInfo, responseFormat, enableThinking)
		if err != nil {
			lastErr = err
			continue
		}

		generateCtx := ctx
		var generateCancel context.CancelFunc
		if timeout > 0 {
			generateCtx, generateCancel = context.WithTimeout(ctx, timeout)
		}

		msg, err := chatModel.Generate(generateCtx, input, qwen.WithExtraHeader(config.GetUA()))
		if generateCancel != nil {
			generateCancel()
		}
		// 成功
		if err == nil {
			success = true
			llmResult = strings.TrimSpace(msg.Content)
			break
		}

		log.WithCtx(ctx).Errorw(
			msgName,
			"provider", llmInfo,
			"attempt", attempt+1,
			"err", err,
		)
		lastErr = err
		continue // 继续重试同一模型
	}

Done:
	if success {
		return llmResult, llmInfo, nil // 始终返回 nil，单个失败不阻塞整体流程
	}
	return "", llmInfo, lastErr
}

// RunModelGenerateReAct 带错误反馈的迭代生成（ReAct 式自我修正循环）：
// 第一轮只发 initialInput；校验失败后，把失败输出与失败原因追加到对话中作为反馈，
// 再调模型自我修正，最多 maxAttempts 轮（含首轮）。
//
// validate 通过时返回 (要保存的值, nil)，未通过时返回 ("", 错误)，
// 错误信息会作为反馈内容发给模型。
//
// 传输层失败（网络/限流/超时）在同一轮内指数退避重试 transportRetries 次，
// 不消耗反馈轮次；全部传输重试失败则直接返回错误。
func RunModelGenerateReAct(
	ctx context.Context,
	llmInfo *types.LLMInfo,
	msgName string,
	initialInput []*schema.Message,
	validate func(output string) (string, error),
	maxAttempts int,
	transportRetries int,
	timeout time.Duration,
	enableThinking bool,
	responseFormat openai.ChatCompletionResponseFormatType,
) (string, *types.LLMInfo, error) {
	chatModel, err := pkg.NewChatModel(ctx, llmInfo, responseFormat, enableThinking)
	if err != nil {
		return "", llmInfo, err
	}

	messages := make([]*schema.Message, 0, len(initialInput)+maxAttempts*2)
	messages = append(messages, initialInput...)

	var lastOutput string
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var msg *schema.Message
		for t := 0; t < transportRetries; t++ {
			if t > 0 {
				// 指数退避：2s, 4s, 8s, 16s
				backoff := time.Duration(1<<t) * time.Second
				log.WithCtx(ctx).Infow(msgName, "provider", llmInfo, "attempt", attempt, "transport", t+1, "backoff", backoff)
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return lastOutput, llmInfo, ctx.Err()
				}
			}

			generateCtx := ctx
			var cancel context.CancelFunc
			if timeout > 0 {
				generateCtx, cancel = context.WithTimeout(ctx, timeout)
			}
			msg, err = chatModel.Generate(generateCtx, messages, qwen.WithExtraHeader(config.GetUA()))
			if cancel != nil {
				cancel()
			}
			if err == nil {
				break
			}
			log.WithCtx(ctx).Errorw(msgName, "provider", llmInfo, "attempt", attempt, "transport", t+1, "err", err)
		}
		if err != nil {
			return lastOutput, llmInfo, err
		}

		output := strings.TrimSpace(msg.Content)
		lastOutput = output

		saved, verr := validate(output)
		if verr == nil {
			return saved, llmInfo, nil
		}
		lastErr = verr
		log.WithCtx(ctx).Warnw(msgName, "provider", llmInfo, "attempt", attempt, "err", verr, "output_head", runePrefix(output, 200))

		if attempt >= maxAttempts {
			break
		}
		// ReAct 反馈：把失败输出与失败原因追加到对话，让模型自我修正
		messages = append(messages,
			schema.AssistantMessage(truncateRune(output, 1000), nil),
			schema.UserMessage(fmt.Sprintf(
				"你上一次的输出未通过校验：%s\n上一次输出的开头片段：%s\n请根据错误修正，严格按照原始要求重新输出。",
				verr.Error(), runePrefix(output, 500))),
		)
	}

	return lastOutput, llmInfo, lastErr
}

// runePrefix 取字符串开头 n 个 rune。
func runePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// truncateRune 截断到前 n 个 rune，截断时追加省略标记。
func truncateRune(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "……(截断)"
}
