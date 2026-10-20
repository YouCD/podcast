package workflow

import "testing"

func TestNormalizeCategory(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// 白名单原值直通
		{"AI", "AI"},
		{"科技", "科技"},
		{"low_quality", "low_quality"},
		{"行业洞察", "行业洞察"},
		// 大小写/空白
		{"ai", "AI"},
		{"  AI  ", "AI"},
		{"AI\n", "AI"},
		{"AI                +", "AI"}, // 含换行/加号的多行脏值
		// 前缀形式
		{"输出：AI", "AI"},
		{"输出: 技术文章", "技术文章"},
		{"输出:技术文章", "技术文章"},
		{"类别：科技", "科技"},
		{"分类: 软件工具", "软件工具"},
		{"输出结果: AI", "AI"},
		{"输出类别: 人工智能", "AI"},
		{"Output: AI", "AI"},
		{"output：AI", "AI"},
		{"# 类别：信息安全", "信息安全"},
		// markdown 装饰
		{"**AI**", "AI"},
		{"# AI", "AI"},
		{"【国际新闻】", "国际新闻"},
		{"**其他**", "其他"},
		// 低质量变体
		{"低质量", "low_quality"},
		{"输出：低质量", "low_quality"},
		{"输出: 低质量", "low_quality"},
		{"low", "low_quality"},
		{"低_quality", "low_quality"},
		{"降低质量", "low_quality"},
		{"低质量文章。", "low_quality"},
		// 英文/近义别名
		{"人工智能", "AI"},
		{"人工智能代理", "AI"},
		{"tech", "科技"},
		{"IT", "科技"},
		{"social", "社会"},
		{"automotive", "汽车"},
		{"efficiency", "效率方法"},
		{"高效方法", "效率方法"},
		// 白名单外 → 其他
		{"交通", "其他"},
		{"法律", "其他"},
		{"娱乐", "其他"},
		{"汽车、行业洞察", "其他"},
		{"输出：其他主题匹配", "其他"},
		{"输出：人形机器人", "其他"},
		{"© THE END", "其他"},
		{"**unຫ)kinˆ", "其他"},
		{"s ?", "其他"},
		{"\\", "其他"},
		{"", "其他"},
		{"   ", "其他"},
		{"这篇文章应该归类为AI，因为内容涉及模型发布", "其他"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := normalizeCategory(tc.in); got != tc.want {
				t.Errorf("normalizeCategory(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestNormalizeCategory_AllInWhitelist 归一化结果必须落在白名单内
func TestNormalizeCategory_AllInWhitelist(t *testing.T) {
	dirty := []string{
		"输出：AI", "类别：科技", "**AI**", "# AI", "【国际新闻】", "Output: AI",
		"输出：低质量", "低_quality", "low", "降低质量", "低质量文章。",
		"tech", "social", "automotive", "efficiency", "IT", "Chinese",
		"交通", "法律", "政策", "安全", "游戏", "© THE END", "s ?", "",
		"The author", "Copy Link", "人工智能", "汽车、行业洞察", "输出：寒露",
	}
	// 白名单值集合（allowedCategories 的 value 为规范类别名）
	allowed := map[string]bool{}
	for _, v := range allowedCategories {
		allowed[v] = true
	}
	for _, d := range dirty {
		got := normalizeCategory(d)
		if !allowed[got] && got != "其他" {
			t.Errorf("normalizeCategory(%q) = %q 不在白名单内", d, got)
		}
	}
}
