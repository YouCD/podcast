package workflow

import (
	"context"
	"strings"
	"sync"
	"unicode"

	"podcast/internal/ai/common"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
	"golang.org/x/sync/semaphore"
)

func categorization(ctx context.Context, state *graphState) (*graphState, error) {
	if len(state.UniqueItems) == 0 {
		return state, ErrRSSSourceNotContent
	}

	log.WithCtx(ctx).Infof("[阶段4/7] 开始分类处理，共 %d 条", len(state.UniqueItems))

	// 创建信号量控制并发
	sem := semaphore.NewWeighted(state.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, item := range state.UniqueItems {
		wg.Add(1)
		// 获取信号量
		if err := sem.Acquire(ctx, 1); err != nil {
			log.WithCtx(ctx).Errorw("categorization: acquire semaphore failed", "error", err)
			wg.Done()
			continue
		}

		go func(item *types.RSSItem) {
			defer wg.Done()
			defer sem.Release(1)

			log.WithCtx(ctx).Debugw("[阶段4/7] 开始分类", "md5", item.MD5, "title", item.Title, "source", item.Source)

			msgs, err := newCategorizationTemplate(ctx).Format(ctx, map[string]any{
				"title":   item.Title,
				"content": item.Content,
			})
			if err != nil {
				log.WithCtx(ctx).Errorw("categorization", "md5", item.MD5, "content", item.Content, "error", err)
				mu.Lock()
				state.Categorization[item] = "其他"
				mu.Unlock()
				return
			}
			var category string
			var llmInfo *types.LLMInfo
			var retryCount int
			const maxRetry = 3
		Retry:
			category, llmInfo, err = common.RunModelGenerate(ctx, state.llmInfo, "categorization", msgs, openai.ChatCompletionResponseFormatTypeText, 3, state.llmTimeout, false)
			if err != nil {
				retryCount++
				if retryCount < maxRetry {
					log.WithCtx(ctx).Warnw("categorization retry", "provider", llmInfo, "md5", item.MD5, "attempt", retryCount, "error", err)
					goto Retry
				}
				log.WithCtx(ctx).Errorw("categorization failed after retries", "provider", llmInfo, "md5", item.MD5, "content", item.Content, "error", err)
				mu.Lock()
				state.Categorization[item] = "其他"
				mu.Unlock()
				return
			}
			// 归一化 LLM 原始输出（剥离前缀/装饰/空白，白名单校验，兜底「其他」）
			normalized := normalizeCategory(category)
			if normalized != category {
				log.WithCtx(ctx).Debugw("categorization normalized", "md5", item.MD5, "raw", category, "normalized", normalized)
			}
			mu.Lock()
			state.Categorization[item] = normalized
			mu.Unlock()
			log.WithCtx(ctx).Debugw("categorization", "provider", llmInfo, "md5", item.MD5, "content", item.Content, "category", normalized)
		}(item)
	}

	// 等待所有任务完成
	wg.Wait()

	log.WithCtx(ctx).Infof("[阶段4/7] 分类完成，共 %d 条", len(state.Categorization))
	return state, nil // 始终返回 nil，单个失败不阻塞整体流程
}

// allowedCategories 类别白名单（与分类提示词中的 Category List 保持一致），key 为小写形式
var allowedCategories = map[string]string{
	"ai": "AI", "技术文章": "技术文章", "体育": "体育", "信息安全": "信息安全",
	"军事": "军事", "国际新闻": "国际新闻", "影视": "影视", "政治": "政治",
	"汽车": "汽车", "社会": "社会", "科技": "科技", "财经": "财经",
	"软件工具": "软件工具", "效率方法": "效率方法", "行业洞察": "行业洞察",
	"low_quality": "low_quality",
}

// categoryAliases 常见非规范输出的映射（key 为小写形式）
var categoryAliases = map[string]string{
	"低质量": "low_quality", "low": "low_quality", "低_quality": "low_quality",
	"人工智能": "AI", "人工智能代理": "AI",
	"tech": "科技", "it": "科技", "technology": "科技",
	"social": "社会",
	"automotive": "汽车",
	"efficiency": "效率方法", "高效方法": "效率方法",
}

// categoryPrefixes LLM 易输出的「前缀:」形式（长前缀在前，避免短前缀部分匹配后残留）
var categoryPrefixes = []string{"输出结果", "输出类别", "输出", "类别", "分类", "output"}

// normalizeCategory 将 LLM 的原始类别输出归一化为白名单类别：
// 小写化 → 去全部空白（含换行）→ 去首尾 markdown 装饰 → 剥离「前缀:」→ 低质量变体归一 → 白名单校验，其余兜底「其他」。
// 保证落库的 categories 一定是白名单值，避免污染下游精确匹配逻辑（dgraph 白名单、自定义提示词查询、low_quality 判断）。
func normalizeCategory(raw string) string {
	s := strings.ToLower(raw)
	// 白名单类别名不含内部空白，去除所有空白字符（含换行）
	s = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	// 去除首尾 markdown 装饰与杂散符号
	s = strings.Trim(s, "*#`【】+＋()（）")
	// 剥离「前缀:」形式（全角/冒号均可，注意按 rune 而非字节切分）
	for {
		rs := []rune(s)
		matched := false
		for _, p := range categoryPrefixes {
			pr := []rune(p)
			if len(rs) >= len(pr)+1 && string(rs[:len(pr)]) == p && (rs[len(pr)] == '：' || rs[len(pr)] == ':') {
				rs = rs[len(pr)+1:]
				matched = true
				break
			}
		}
		if !matched {
			break
		}
		s = string(rs)
	}
	s = strings.Trim(s, "*#`【】+＋()（） ")
	if s == "" {
		return "其他"
	}
	if c, ok := allowedCategories[s]; ok {
		return c
	}
	if c, ok := categoryAliases[s]; ok {
		return c
	}
	// 含「低质量」字样的长尾变体（如「低质量文章。」）
	if strings.Contains(s, "低质量") {
		return "low_quality"
	}
	return "其他"
}
