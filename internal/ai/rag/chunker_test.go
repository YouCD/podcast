package rag

import (
	"strings"
	"testing"
)

func TestChunkContent_ShortContent(t *testing.T) {
	chunks := ChunkContent("hello world", 800, 100)
	if len(chunks) != 1 || chunks[0] != "hello world" {
		t.Errorf("短内容应单段: %+v", chunks)
	}
}

func TestChunkContent_Empty(t *testing.T) {
	if chunks := ChunkContent("   \n  ", 800, 100); len(chunks) != 0 {
		t.Errorf("空内容应返回空: %+v", chunks)
	}
}

func TestChunkContent_SingleChunkAtLimit(t *testing.T) {
	// 正好 chunkSize 长度
	content := strings.Repeat("a", 800)
	chunks := ChunkContent(content, 800, 100)
	if len(chunks) != 1 {
		t.Fatalf("800 字应单段, got %d", len(chunks))
	}
}

func TestChunkContent_MultiChunkWithOverlap(t *testing.T) {
	// 2500 字无边界字符（全是 'a'）→ 硬切
	content := strings.Repeat("a", 2500)
	chunks := ChunkContent(content, 800, 100)

	// 期望: [0:800), [700:1500), [1400:2200), [2100:2500)
	if len(chunks) != 4 {
		t.Fatalf("2500 字应切 4 段, got %d: %v", len(chunks), lengths(chunks))
	}
	if len([]rune(chunks[0])) != 800 || len([]rune(chunks[1])) != 800 {
		t.Errorf("前两段应为 800 字: %v", lengths(chunks))
	}
	if len([]rune(chunks[3])) != 400 {
		t.Errorf("末段应为 400 字: %v", lengths(chunks))
	}

	// 验证重叠：chunk[i+1] 的开头 100 字 == chunk[i] 的结尾 100 字
	for i := 0; i+1 < len(chunks); i++ {
		tail := string([]rune(chunks[i])[700:])
		head := string([]rune(chunks[i+1])[:100])
		if tail != head {
			t.Errorf("第 %d/%d 段重叠不匹配", i, i+1)
		}
	}
}

func TestChunkContent_BoundaryPreferred(t *testing.T) {
	// 750 个 'a' + "。" + 200 个 'b'：第一刀应在句号处（751）而非硬切 800
	content := strings.Repeat("a", 750) + "。" + strings.Repeat("b", 200)
	chunks := ChunkContent(content, 800, 100)
	if len(chunks) < 2 {
		t.Fatalf("应至少 2 段: %v", lengths(chunks))
	}
	if !strings.HasSuffix(chunks[0], "。") {
		t.Errorf("第一段应在句子边界结束: %q", tail10(chunks[0]))
	}
}

func TestChunkContent_Chinese(t *testing.T) {
	// 中文按 rune 计长
	content := strings.Repeat("测", 2000)
	chunks := ChunkContent(content, 800, 100)
	total := 0
	for _, c := range chunks {
		total += len([]rune(c))
	}
	if total != 2000+2*100 { // 3 段：800+800+400，重叠 2×100
		t.Errorf("总长应为 2200（含重叠）, got %d: %v", total, lengths(chunks))
	}
}

func TestChunkContent_InvalidParams(t *testing.T) {
	if chunks := ChunkContent("abc", 0, 100); len(chunks) != 0 {
		t.Errorf("chunkSize<=0 应返回空")
	}
	if chunks := ChunkContent("abc", 100, 100); len(chunks) != 0 {
		t.Errorf("overlap>=chunkSize 应返回空")
	}
}

func lengths(chunks []string) []int {
	out := make([]int, len(chunks))
	for i, c := range chunks {
		out[i] = len([]rune(c))
	}
	return out
}

func tail10(s string) string {
	r := []rune(s)
	if len(r) > 10 {
		return string(r[len(r)-10:])
	}
	return s
}
