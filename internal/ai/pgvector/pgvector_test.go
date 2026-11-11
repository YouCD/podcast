package pgvector

import (
	"strings"
	"testing"
)

func TestVectorToString(t *testing.T) {
	cases := []struct {
		name   string
		vector []float32
		want   string
	}{
		{"单元素", []float32{0.5}, "[0.500000]"},
		{"多元素", []float32{0.1, -0.25, 1.0}, "[0.100000,-0.250000,1.000000]"},
		{"空向量", []float32{}, "[]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := VectorToString(c.vector); got != c.want {
				t.Errorf("VectorToString = %q, want %q", got, c.want)
			}
		})
	}
}

func TestStringToVector(t *testing.T) {
	got, err := stringToVector("[0.1,-0.25,1.0]")
	if err != nil {
		t.Fatalf("stringToVector: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("期望 3 维，得到 %d: %v", len(got), got)
	}
	if got[0] != 0.1 || got[1] != -0.25 || got[2] != 1.0 {
		t.Errorf("解析错误: %v", got)
	}
}

func TestVectorRoundTrip(t *testing.T) {
	original := []float32{0.123456, -0.789, 0.0, 2.5}
	s := VectorToString(original)
	back, err := stringToVector(s)
	if err != nil {
		t.Fatalf("stringToVector: %v", err)
	}
	if len(back) != len(original) {
		t.Fatalf("维度不一致: %d vs %d", len(back), len(original))
	}
	for i := range original {
		diff := back[i] - original[i]
		if diff < 0 {
			diff = -diff
		}
		if diff > 1e-5 {
			t.Errorf("索引 %d 往返不一致: %f vs %f", i, back[i], original[i])
		}
	}
}

func TestVectorToStringNonEmpty(t *testing.T) {
	s := VectorToString([]float32{1, 2, 3})
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		t.Errorf("向量字符串应以方括号包裹: %q", s)
	}
}
