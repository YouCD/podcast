package embedding

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"podcast/pkg/types"
)

func TestNewEmbedder_ProviderDetection(t *testing.T) {
	cases := []struct {
		name    string
		cfg     *types.Embedding
		wantErr bool
	}{
		{
			name: "ollama provider 显式指定",
			cfg:  &types.Embedding{Provider: "ollama", BaseURL: "http://192.168.1.188:11434", Model: "qwen3-embedding:0.6b", Dimension: 1024},
		},
		{
			name: "非 dashscope baseURL 自动识别为 ollama",
			cfg:  &types.Embedding{BaseURL: "http://192.168.1.188:11434", Model: "qwen3-embedding:0.6b", Dimension: 1024},
		},
		{
			name: "ollama 缺少 baseURL 报错",
			cfg:  &types.Embedding{Provider: "ollama", Model: "qwen3-embedding:0.6b"},
			wantErr: true,
		},
		{
			name: "dashscope baseURL 识别为 dashscope",
			cfg:  &types.Embedding{APIKey: "sk-test", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", Model: "text-embedding-v4", Dimension: 1024},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e, err := NewEmbedder(context.Background(), tc.cfg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际 nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewEmbedder 失败: %v", err)
			}
			if e == nil {
				t.Fatal("返回 nil embedder")
			}
		})
	}
}

func TestOllamaClient_Endpoint(t *testing.T) {
	o := &ollamaClient{baseURL: "http://192.168.1.188:11434", model: "qwen3-embedding:0.6b"}
	// 构造一个请求报文验证 endpoint 与 body 结构（不发网络请求）
	req, err := o.newEmbedRequest(context.Background(), []string{"测试"})
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	if got := req.URL.String(); got != "http://192.168.1.188:11434/api/embed" {
		t.Errorf("endpoint 不匹配: %s", got)
	}
	var body bytes.Buffer
	if _, err := body.ReadFrom(req.Body); err != nil {
		t.Fatalf("读取 body 失败: %v", err)
	}
	if !strings.Contains(body.String(), `"qwen3-embedding:0.6b"`) {
		t.Errorf("body 缺少 model: %s", body.String())
	}
}
