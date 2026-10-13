package daily

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"podcast/internal/ai/common"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

// PodcastDialogue 播客对话单元
type PodcastDialogue struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}

// parsePodcastJSON 解析LLM输出的JSON格式播客脚本
func parsePodcastJSON(jsonStr string) (string, error) {
	// 尝试解析为 []map[string]string 格式
	var dialogues []map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &dialogues); err != nil {
		return "", fmt.Errorf("JSON解析失败: %w", err)
	}

	// 转换为纯文本格式 [S1] 内容\n[S2] 内容
	var lines []string
	for _, dialogue := range dialogues {
		// 找到key（S1或S2）和value（内容）
		for speaker, text := range dialogue {
			if speaker == "S1" || speaker == "S2" {
				lines = append(lines, fmt.Sprintf("[%s] %s", speaker, text))
				break
			}
		}
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("未解析到任何对话内容")
	}

	return strings.Join(lines, "\n"), nil
}

func generatePodcastContent(ctx context.Context, state *graphState) (*graphState, error) {
	if state.Report.PodcastContent != "" {
		log.WithCtx(ctx).Info("播客内容已存在")
		return state, nil
	}
	if state.isNewReport {
		if state.Report.Content == "" {
			// 内容为空时，记录警告但不返回错误，让工作流继续执行
			log.WithCtx(ctx).Warn("播客内容生成跳过：报告内容为空")
			return state, nil
		}
	}
	log.WithCtx(ctx).Info("播客内容生成开始")
	format, err := newGenPodcastSummaryTemplate().Format(ctx, map[string]any{"content": state.Report.Content})
	if err != nil {
		// 格式化失败时，记录错误但不中断工作流
		log.WithCtx(ctx).Errorw("播客内容格式化失败", "error", err)
		return state, nil
	}

	// 校验失败时把错误原因反馈给模型自我修正；最终失败只跳过播客，不中断工作流
	podcastText, llmInfo, err := common.RunModelGenerateReAct(ctx, state.llmInfo, "generatePodcastContent", format,
		func(output string) (string, error) {
			// 解析JSON格式的播客脚本，转换为纯文本格式
			text, perr := parsePodcastJSON(output)
			if perr == nil {
				return text, nil
			}
			// 回退：接受纯文本对话格式
			if strings.HasPrefix(strings.TrimSpace(output), "[S1]") || strings.HasPrefix(strings.TrimSpace(output), "[S2]") {
				return output, nil
			}
			return "", fmt.Errorf("播客脚本格式错误(%v)，请返回JSON对话数组或[S1]/[S2]开头的纯文本", perr)
		},
		3, 5, 0, true, openai.ChatCompletionResponseFormatTypeText)
	if err != nil {
		log.WithCtx(ctx).Errorw("播客内容生成失败，跳过", "provider", llmInfo, "llmResult", podcastText, "err", err)
		return state, nil
	}

	state.Report.PodcastContent = podcastText
	log.WithCtx(ctx).Infow("播客内容生成结束", "llmInfo", llmInfo, "lines", len(strings.Split(podcastText, "\n")))
	return state, nil
}
