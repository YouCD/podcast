package types

import "net/url"

// LLMInfo 接口定义

type LLMInfo struct {
	ID      int64
	ApiKey  string `json:"api_key"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}

func (m *LLMInfo) GetModelName() string {
	return m.Model
}
func (m *LLMInfo) GetBaseURL() string {
	parse, _ := url.Parse(m.BaseURL)
	return parse.Host
}

type QA struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type LLMResult struct {
	Subtitle       string `json:"subtitle"`
	ContentSummary string `json:"contentSummary"`
	QA             []*QA  `json:"qa"`
}

//nolint:tagliatelle
type UserQuery struct {
	StartTime  string   `json:"start_time"`
	EndTime    string   `json:"end_time"`
	Queries    []string `json:"queries"`
	Categories string   `json:"categories"`
}
