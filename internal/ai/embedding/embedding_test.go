package embedding

import (
	"bytes"
	"context"
	"math"
	"net/http"
	"strings"
	"testing"

	"podcast/config"
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
	if got := req.Header.Get("User-Agent"); got != config.UA {
		t.Errorf("User-Agent 应为 %s，实际 %s", config.UA, got)
	}
}

func readBody(t *testing.T, req *http.Request) string {
	t.Helper()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(req.Body); err != nil {
		t.Fatalf("读取 body 失败: %v", err)
	}
	return buf.String()
}

func TestOllamaClient_DimensionsParam(t *testing.T) {
	// 配置了 dimension：body 应携带 dimensions 参数（matryoshka 截断）
	d := 1024
	o := &ollamaClient{baseURL: "http://192.168.1.188:11434", model: "qwen3-embedding:4b", dimensions: &d}
	req, err := o.newEmbedRequest(context.Background(), []string{"测试"})
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	body := readBody(t, req)
	if !strings.Contains(body, `"dimensions":1024`) {
		t.Errorf("body 应包含 dimensions=1024: %s", body)
	}

	// 未配置 dimension：body 不应携带 dimensions（omitempty，使用模型默认维度）
	o2 := &ollamaClient{baseURL: "http://192.168.1.188:11434", model: "qwen3-embedding:4b"}
	req2, err := o2.newEmbedRequest(context.Background(), []string{"测试"})
	if err != nil {
		t.Fatalf("构造请求失败: %v", err)
	}
	if strings.Contains(readBody(t, req2), `"dimensions"`) {
		t.Errorf("body 不应包含 dimensions: %s", body)
	}
}

func TestTruncateAndNormalize(t *testing.T) {
	out := truncateAndNormalize([]float32{3, 4, 0.1, 0.2}, 2)
	if len(out) != 2 {
		t.Fatalf("期望 2 维，实际 %d", len(out))
	}
	if math.Abs(float64(out[0])-0.6) > 1e-6 || math.Abs(float64(out[1])-0.8) > 1e-6 {
		t.Errorf("[3,4] 归一化应为 [0.6,0.8]，实际 %v", out)
	}
}

func TestCreateEmbeddings_DimensionFallback(t *testing.T) {
	// 模拟服务端返回超过配置维度的向量（如 qwen3-embedding:4b 默认 2560 维），
	// 应被兜底截断到配置维度 1024，而不是让 pgvector 插入报维度不匹配错误。
	fake := &fakeClient{vec: make([]float32, 2560)}
	for i := range fake.vec {
		fake.vec[i] = 1
	}
	e := &Embedder{model: "qwen3-embedding:4b", dimension: 1024, client: fake}
	vec, err := e.CreateEmbeddings(context.Background(), "测试")
	if err != nil {
		t.Fatalf("CreateEmbeddings 失败: %v", err)
	}
	if len(vec) != 1024 {
		t.Fatalf("期望 1024 维，实际 %d", len(vec))
	}
	// 归一化校验：2560 个 1 截断到 1024 个 1，归一化后每维 = 1/sqrt(1024)
	want := 1.0 / math.Sqrt(1024)
	if math.Abs(float64(vec[0])-want) > 1e-5 {
		t.Errorf("归一化结果错误: got %v want %v", vec[0], want)
	}

	// 返回维度小于配置维度时应报错
	e2 := &Embedder{model: "m", dimension: 1024, client: &fakeClient{vec: make([]float32, 512)}}
	if _, err := e2.CreateEmbeddings(context.Background(), "测试"); err == nil {
		t.Error("维度小于配置维度时应报错")
	}
}

// fakeClient 测试用 embedClient
type fakeClient struct{ vec []float32 }

func (f *fakeClient) embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{f.vec}, nil
}

// TestCreateEmbeddings_LiveOllama 真实环境验证：qwen3-embedding:4b + dimension=1024 应返回 1024 维
// （服务器不可达时跳过，不影响离线测试）
func TestCreateEmbeddings_LiveOllama(t *testing.T) {
	cfg := &types.Embedding{Provider: "ollama", BaseURL: "http://192.168.1.188:11434", Model: "qwen3-embedding:4b", Dimension: 1024}
	e, err := NewEmbedder(context.Background(), cfg)
	if err != nil {
		t.Skipf("创建 embedder 失败: %v", err)
	}
	vec, err := e.CreateEmbeddings(context.Background(), "医疗系统服务公司 Medley 旗下业务平台遭黑客入侵")
	if err != nil {
		t.Skipf("ollama 服务不可达或调用失败: %v", err)
	}
	if len(vec) != 1024 {
		t.Fatalf("期望 1024 维，实际 %d", len(vec))
	}
}
