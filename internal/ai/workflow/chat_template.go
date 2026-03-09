package workflow

import (
	"context"
	"podcast/internal/database/dao"
	"podcast/internal/database/models"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	"github.com/youcd/toolkit/log"
)

// 创建分类模板
func newCategorizationTemplate(ctx context.Context) prompt.ChatTemplate {
	// 使用的内容变量
	usePrompt := PromptLLMCategorization
	var keyInfoDao = dao.NewKeyInfoDao(models.GetDb())
	p, _ := keyInfoDao.FindByKeynameAndGenre(ctx, "categorization", 1)
	if p != nil {
		if p.Data != "" {
			// 如果成功获取到提示词，则使用获取到的提示词
			usePrompt = p.Data
			log.WithCtx(ctx).Debugw("使用自定义提示词进行LLM分类", "key_name", "categorization")
		}
	}

	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(usePrompt),
		schema.UserMessage(`标题: {title}
内容: {content}`),
	)
}

// 创建Rss内容分析模板
func newRssAnalyzeTemplate(ctx context.Context, categories string) prompt.ChatTemplate {
	usePrompt := PromptLLMResult
	var keyInfoDao = dao.NewKeyInfoDao(models.GetDb())
	// 从prompt_dao获取分类为"llm_result"的提示词
	p, _ := keyInfoDao.FindByKeynameAndGenre(ctx, categories, 1)
	if p != nil {
		// 如果成功获取到提示词，则使用获取到的提示词
		usePrompt = p.Data
		// item.Score = 80 // 设置分数,代表使用自定义提示词
	}

	return prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage(usePrompt),
		schema.UserMessage(`发布时间：{{.date}}
Rss内容：{{.content}}`),
	)
}

// 创建清理模板
func newCleanTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(PromptCleanContent),
		schema.UserMessage(`Rss内容：
{content}`),
	)
}

func newRssDgraphTemplate(categories string) prompt.ChatTemplate {
	/*
		财经 / 行业洞察
		政治 / 国际新闻
		科技 / AI
	*/
	var usePrompt string
	switch categories {
	case "财经", "行业洞察":
		usePrompt = PromptDgraphB
	case "政治", "国际新闻":
		usePrompt = PromptDgraphC
	case "科技", "AI":
		usePrompt = PromptDgraphA
	}
	// 从prompt_dao获取分类为"llm_result"的提示词
	return prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage(usePrompt),
		schema.UserMessage(`发布时间：{{.date}}
RSS标题：{{.title}}
Rss内容：{{.content}}`),
	)
}
