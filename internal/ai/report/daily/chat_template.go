package daily

import (
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func newGenMarkdownSummaryTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(PromptMarkdownSummary),
		schema.UserMessage(`RSS 内容列表:
{content}`),
	)
}

func newGenHtmlSummaryTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.GoTemplate,
		schema.SystemMessage(PromptMarkdown2Html),
		schema.UserMessage(`RSS 内容列表:
{{.content}}`),
	)
}

func newGenPodcastSummaryTemplate() prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage(PromptMarkdown2Podcast),
		schema.UserMessage(`内容如下:
{content}`),
	)
}
