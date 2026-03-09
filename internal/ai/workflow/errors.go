package workflow

import (
	"fmt"
)

var (
	ErrNotLLMResult        = fmt.Errorf("not llm result")
	ErrRSSSourceNotContent = fmt.Errorf("rss source not content")
)
