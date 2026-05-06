package daily

import (
	"testing"
)

func TestParsePodcastJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonInput     string
		expectError   bool
		expectedLines int
	}{
		{
			name: "正确格式 - 基本对话",
			jsonInput: `[
  {"S1": "哎老王，你听说没？"},
  {"S2": "嚯，你这消息够灵通的啊。"},
  {"S1": "还夸张？我朋友圈都炸了！"}
]`,
			expectError:   false,
			expectedLines: 3,
		},
		{
			name: "正确格式 - 带转义引号",
			jsonInput: `[
  {"S1": "他说：\"你好啊\""},
  {"S2": "真的吗？"}
]`,
			expectError:   false,
			expectedLines: 2,
		},
		{
			name: "错误格式 - 使用了speaker字段",
			jsonInput: `[
  {"speaker": "S1", "text": "你好"},
  {"speaker": "S2", "text": "嗨"}
]`,
			expectError:   true, // 这种格式不会被正确解析
			expectedLines: 0,
		},
		{
			name: "错误格式 - JSON语法错误",
			jsonInput: `[
  {"S1": "你好"},
  {"S2": "嗨",}
]`,
			expectError:   true,
			expectedLines: 0,
		},
		{
			name:          "错误格式 - 空数组",
			jsonInput:     `[]`,
			expectError:   true,
			expectedLines: 0,
		},
		{
			name: "正确格式 - 长对话",
			jsonInput: `[
  {"S1": "第一句"},
  {"S2": "第二句"},
  {"S1": "第三句"},
  {"S2": "第四句"},
  {"S1": "第五句"}
]`,
			expectError:   false,
			expectedLines: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parsePodcastJSON(tt.jsonInput)

			if tt.expectError && err == nil {
				t.Errorf("期望有错误，但实际没有")
			}

			if !tt.expectError && err != nil {
				t.Errorf("不期望有错误，但实际发生了: %v", err)
			}

			if !tt.expectError {
				lines := len(splitLines(result))
				if lines != tt.expectedLines {
					t.Errorf("期望%d行，实际%d行\n结果:\n%s", tt.expectedLines, lines, result)
				}

				// 检查格式是否正确
				for i, line := range splitLines(result) {
					if len(line) < 5 || line[:3] != "[S1" && line[:3] != "[S2" {
						t.Errorf("第%d行格式不正确: %s", i+1, line)
					}
				}
			}
		})
	}
}

func TestParsePodcastJSON_FormatValidation(t *testing.T) {
	// 测试转换后的格式是否符合要求
	jsonInput := `[
  {"S1": "哎老王，紧急呼叫！"},
  {"S2": "嚯，这什么动静，我刚泡的咖啡差点洒我芯片上。"},
  {"S1": "别提芯片了！我正刷手机呢。"}
]`

	result, err := parsePodcastJSON(jsonInput)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	lines := splitLines(result)

	// 检查每行的格式
	for i, line := range lines {
		if len(line) < 5 {
			t.Errorf("第%d行太短: %s", i+1, line)
			continue
		}

		// 检查是否以 [S1] 或 [S2] 开头
		if line[:3] != "[S1" && line[:3] != "[S2" {
			t.Errorf("第%d行不是以[S1]或[S2]开头: %s", i+1, line)
		}

		// 检查是否有内容
		if len(line) <= 5 {
			t.Errorf("第%d行没有内容: %s", i+1, line)
		}
	}

	// 检查总行数
	if len(lines) != 3 {
		t.Errorf("期望3行，实际%d行", len(lines))
	}

	t.Logf("转换结果:\n%s", result)
}

func splitLines(text string) []string {
	var lines []string
	for _, line := range splitByNewline(text) {
		if trimmed := trimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

// 简单的字符串分割和修剪函数（避免导入strings包）
func splitByNewline(s string) []string {
	var result []string
	current := ""
	for _, c := range s {
		if c == '\n' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
