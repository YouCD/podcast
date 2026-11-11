package agent

import (
	"errors"
	"strings"
	"testing"
)

func TestTruncateResult(t *testing.T) {
	// 不超长：原样返回
	if got := truncateResult("短文本", 100); got != "短文本" {
		t.Errorf("短文本不应被截断: %q", got)
	}
	// 超长：截断（按 rune）
	long := strings.Repeat("汉", 50)
	got := truncateResult(long, 10)
	runes := []rune(got)
	if runes[0] != '汉' || len(runes) <= 10 {
		t.Errorf("截断结果异常: %q (len=%d)", got, len(runes))
	}
	if !strings.Contains(got, "截断") {
		t.Errorf("应带截断标记: %q", got)
	}
}

func TestIsRetryableRateLimit(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("HTTP 429 Too Many Requests"), true},
		{errors.New("rate limit exceeded"), true},
		{errors.New("Throttling.RateQuota: Too many requests"), true},
		{errors.New("quota exceeded for this model"), true},
		{errors.New("requests per minute limit reached"), true},
		{errors.New("429"), true},
		{errors.New("connection refused"), false},
		{errors.New("invalid api key"), false},
		{errors.New("context deadline exceeded"), false},
	}
	for _, c := range cases {
		if got := isRetryableRateLimit(c.err); got != c.want {
			t.Errorf("isRetryableRateLimit(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}
