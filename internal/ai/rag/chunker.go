package rag

import "strings"

const (
	// DefaultChunkSize 每个分段的最大字符数（按 rune 计算）
	DefaultChunkSize = 800
	// DefaultChunkOverlap 相邻分段的重叠字符数
	DefaultChunkOverlap = 100
)

// isChunkBoundary 判断 rune 是否为分段边界（段落/句子结束符）
func isChunkBoundary(r rune) bool {
	switch r {
	case '\n', '。', '！', '？', '!', '?', '；', ';', '.':
		return true
	}
	return false
}

// ChunkContent 将内容切分为多个分段：
//   - 每段最多 chunkSize 个字符（rune）
//   - 相邻段重叠 overlapSize 个字符，保证跨段信息的连续性
//   - 优先在段落/句子边界处切割，避免截断词语
//
// 返回空切片表示没有可索引的内容。
func ChunkContent(content string, chunkSize, overlapSize int) []string {
	if chunkSize <= 0 || overlapSize < 0 || overlapSize >= chunkSize {
		return nil
	}
	runes := []rune(strings.TrimSpace(content))
	if len(runes) == 0 {
		return nil
	}

	var chunks []string
	start := 0
	for start < len(runes) {
		end := start + chunkSize
		if end >= len(runes) {
			if chunk := strings.TrimSpace(string(runes[start:])); chunk != "" {
				chunks = append(chunks, chunk)
			}
			break
		}

		// 在 [end-overlapSize, end) 区间内找最近的句子边界
		cut := -1
		for i := end - 1; i >= end-overlapSize; i-- {
			if isChunkBoundary(runes[i]) {
				cut = i + 1
				break
			}
		}
		if cut == -1 {
			cut = end // 无边界则硬切
		}

		if chunk := strings.TrimSpace(string(runes[start:cut])); chunk != "" {
			chunks = append(chunks, chunk)
		}

		if cut >= len(runes) {
			break
		}
		// 下一段起点 = 本段终点 - 重叠量（chunkSize > overlap 保证 start 严格递增）
		start = cut - overlapSize
	}
	return chunks
}
