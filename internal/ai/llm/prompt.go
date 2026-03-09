package llm

const (
	PromptConversationTitle = `你是一个聊天系统的标题生成器。
请根据用户提供的对话片段，生成一个简短、准确的会话标题。

要求：
- 不超过 15 个字
- 不要标点符号
- 不要引号
- 只输出标题文本

用户对话片段：
%#v
`
)
