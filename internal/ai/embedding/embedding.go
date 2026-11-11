package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"podcast/config"
	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	aclopenai "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/youcd/toolkit/log"
)

// embedClient 抽象底层 embedding 提供方，便于支持 dashscope、ollama 等
type embedClient interface {
	embed(ctx context.Context, texts []string) ([][]float32, error)
}

// uaTransport 在请求未显式携带 User-Agent 时注入应用 UA（PodCast），
// 便于在 llama-server / llama_proxy 等服务端日志中识别来源
type uaTransport struct {
	base http.RoundTripper
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", config.UA)
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

// newHTTPClient 创建注入应用 User-Agent 的 HTTP 客户端
func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: &uaTransport{base: http.DefaultTransport},
	}
}

// Embedder Embedding 客户端
type Embedder struct {
	model     string
	dimension int64 // 期望的向量维度（来自配置），0 表示不校验
	client    embedClient
	// Embedder 兼容历史调用：仅 dashscope 模式下非 nil（供依赖 dashscope 组件的场景使用）
	Embedder *dashscope.Embedder
}

// openaiClient 通用 OpenAI 兼容 embedding（POST {baseURL}/embeddings），
// 适用于 llama_proxy、vLLM、LM Studio 等任何实现 OpenAI embeddings 协议的本地/自托管节点
type openaiClient struct {
	*aclopenai.EmbeddingClient
}

func (o *openaiClient) embed(ctx context.Context, texts []string) ([][]float32, error) {
	result, err := o.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, err
	}
	vectors := make([][]float32, 0, len(result))
	for _, vec := range result {
		row := make([]float32, len(vec))
		for i, f := range vec {
			row[i] = float32(f)
		}
		vectors = append(vectors, row)
	}
	return vectors, nil
}

// dashscopeClient 阿里云百炼 dashscope embedding
type dashscopeClient struct {
	*dashscope.Embedder
}

func (d *dashscopeClient) embed(ctx context.Context, texts []string) ([][]float32, error) {
	result, err := d.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, err
	}
	vectors := make([][]float32, 0, len(result))
	for _, vec := range result {
		row := make([]float32, len(vec))
		for i, f := range vec {
			row[i] = float32(f)
		}
		vectors = append(vectors, row)
	}
	return vectors, nil
}

// ollamaClient 本地 Ollama embedding（POST /api/embed）
type ollamaClient struct {
	baseURL    string
	model      string
	client     *http.Client
	dimensions *int // 可选：请求指定输出维度（matryoshka 截断），nil 表示模型默认维度
}

type ollamaEmbedRequest struct {
	Model      string `json:"model"`
	Input      any    `json:"input"`
	Dimensions *int   `json:"dimensions,omitempty"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// newEmbedRequest 构造 ollama /api/embed 请求（供 embed 调用与单元测试复用）
func (o *ollamaClient) newEmbedRequest(ctx context.Context, texts []string) (*http.Request, error) {
	url := strings.TrimRight(o.baseURL, "/") + "/api/embed"
	body, err := json.Marshal(ollamaEmbedRequest{Model: o.model, Input: texts, Dimensions: o.dimensions})
	if err != nil {
		return nil, fmt.Errorf("marshal ollama embed request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new ollama embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", config.UA)
	return req, nil
}

func (o *ollamaClient) embed(ctx context.Context, texts []string) ([][]float32, error) {
	req, err := o.newEmbedRequest(ctx, texts)
	if err != nil {
		return nil, err
	}
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama embed http %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed ollamaEmbedResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal ollama embed response: %w", err)
	}
	if len(parsed.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama embed returned empty embeddings")
	}
	return parsed.Embeddings, nil
}

// NewEmbedder 创建 Embedding 客户端，支持 dashscope（默认）、ollama 与 openai（通用 OpenAI 兼容协议）
func NewEmbedder(ctx context.Context, cfg *types.Embedding) (*Embedder, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if provider == "" {
		// 兼容旧配置：非 dashscope 的 baseURL 视为 ollama
		if cfg.BaseURL != "" && !strings.Contains(cfg.BaseURL, "dashscope.aliyuncs.com") {
			provider = "ollama"
		} else {
			provider = "dashscope"
		}
	}

	switch provider {
	case "ollama":
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("ollama embedding 需要配置 baseURL（如 http://192.168.1.188:11434）")
		}
		var dimensions *int
		if cfg.Dimension > 0 {
			d := int(cfg.Dimension)
			dimensions = &d
		}
		return &Embedder{
			model:     cfg.Model,
			dimension: cfg.Dimension,
			client: &ollamaClient{
				baseURL:    cfg.BaseURL,
				model:      cfg.Model,
				client:     newHTTPClient(120 * time.Second),
				dimensions: dimensions,
			},
		}, nil
	case "openai":
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("openai embedding 需要配置 baseURL（如 http://192.168.111.126:8083/v1）")
		}
		var dimensions *int
		if cfg.Dimension > 0 {
			d := int(cfg.Dimension)
			dimensions = &d
		}
		encodingFmt := aclopenai.EmbeddingEncodingFormatFloat
		cli, err := aclopenai.NewEmbeddingClient(ctx, &aclopenai.EmbeddingConfig{
			APIKey:         cfg.APIKey,
			BaseURL:        cfg.BaseURL,
			Model:          cfg.Model,
			Dimensions:     dimensions,
			EncodingFormat: &encodingFmt,
			HTTPClient:     newHTTPClient(120 * time.Second),
		})
		if err != nil {
			log.WithCtx(ctx).Error("NewEmbedder failed, init openai embed client err: %v", err)
			return nil, err
		}
		return &Embedder{
			model:     cfg.Model,
			dimension: cfg.Dimension,
			client:    &openaiClient{EmbeddingClient: cli},
		}, nil
	default: // dashscope
		var dimensions *int
		if cfg.Dimension > 0 {
			d := int(cfg.Dimension)
			dimensions = &d
		}
		embedder, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
			APIKey:     cfg.APIKey,
			Model:      cfg.Model,
			Dimensions: dimensions,
			HTTPClient: newHTTPClient(120 * time.Second),
		})
		if err != nil {
			log.WithCtx(ctx).Error("NewEmbedder failed, init dashscope embedder err: %v", err)
			return nil, err
		}
		return &Embedder{
			model:     cfg.Model,
			dimension: cfg.Dimension,
			client:    &dashscopeClient{Embedder: embedder},
			Embedder:  embedder,
		}, nil
	}
}

// CreateEmbeddings 创建单个文本的嵌入向量
func (e *Embedder) CreateEmbeddings(ctx context.Context, content string) ([]float32, error) {
	vectors, err := e.client.embed(ctx, []string{content})
	if err != nil {
		return nil, fmt.Errorf("embedding strings failed: %w", err)
	}
	if len(vectors) == 0 || len(vectors[0]) == 0 {
		return nil, fmt.Errorf("embedding returned empty vector")
	}
	vec := vectors[0]
	// 兜底校验：实际维度必须与配置（即 pgvector 表的 vector(N) 维度）一致，
	// 否则插入时会报 "expected N dimensions, not M"。
	if e.dimension > 0 && int64(len(vec)) != e.dimension {
		if len(vec) > int(e.dimension) {
			// 服务端未按要求截断时，手动取前 N 维并重新归一化（matryoshka 截断方式）。
			// 部分自托管节点（如 llama.cpp llama-server）不支持 dimensions 参数、按原生维度输出，
			// 截断后的向量与服务端 MRL 截断数学等价，属预期行为，故用 DEBUG 级别。
			log.WithCtx(ctx).Debugw("embedding dimension mismatch, truncating",
				"model", e.model, "expected", e.dimension, "actual", len(vec))
			vec = truncateAndNormalize(vec, int(e.dimension))
		} else {
			return nil, fmt.Errorf("embedding 维度 %d 小于配置维度 %d（model=%s），请检查配置", len(vec), e.dimension, e.model)
		}
	}
	return vec, nil
}

// truncateAndNormalize 截取前 n 维并重新做 L2 归一化（matryoshka 截断方式）
func truncateAndNormalize(vec []float32, n int) []float32 {
	out := make([]float32, n)
	copy(out, vec[:n])
	var norm float64
	for _, v := range out {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		s := float32(math.Sqrt(norm))
		for i := range out {
			out[i] /= s
		}
	}
	return out
}
