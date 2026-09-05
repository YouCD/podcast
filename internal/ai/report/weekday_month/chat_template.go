package weekday_month

import (
	"context"

	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

func newGenQueryReWritingTemplate(ctx context.Context) prompt.ChatTemplate {
	usePrompt := PromptRewriting
	keyInfoDao := dao.NewKeyInfoDao(models.GetDb())
	genre, _ := keyInfoDao.FindByKeynameAndGenre(ctx, "userQueryRewriting", 3)
	if genre != nil {
		log.WithCtx(ctx).Debugw("QueryAnalysis", "自定义问题重写提示词", genre.Data)
		usePrompt = genre.Data
	}

	return prompt.FromMessages(
		schema.GoTemplate,
		schema.SystemMessage(usePrompt),
		schema.UserMessage(`当前日期: {{.date}}
用户提问: {{.userQuery}}`),
	)
}

func newAnalysisTemplate(ctx context.Context, queryStr string) prompt.ChatTemplate {
	usePrompt := PromptReportAnalysis
	keyInfoDao := dao.NewKeyInfoDao(models.GetDb())
	genre, _ := keyInfoDao.FindByKeynameAndGenre(ctx, queryStr, 3)
	if genre != nil {
		log.WithCtx(ctx).Debugw("QueryAnalysis", "自定义问题重写提示词", genre.Data)
		usePrompt = genre.Data
	}

	return prompt.FromMessages(
		schema.GoTemplate,
		schema.SystemMessage(usePrompt),
		schema.UserMessage(`用户提问: {{.userQuery}}
当前的日期： {{.date}}
所有的内容：
{{.content}}
	`),
	)
}
