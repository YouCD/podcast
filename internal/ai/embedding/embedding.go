package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"podcast/pkg/types"

	"github.com/cloudwego/eino-ext/components/embedding/dashscope"
	"github.com/youcd/toolkit/log"
)

// embedClient 抽象底层 embedding 提供方，便于支持 dashscope、ollama 等
type embedClient interface {
	embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Embedder Embedding 客户端
type Embedder struct {
	model  string
	client embedClient
	// Embedder 兼容历史调用：仅 dashscope 模式下非 nil（供依赖 dashscope 组件的场景使用）
	Embedder *dashscope.Embedder
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
	baseURL string
	model   string
	client  *http.Client
}

type ollamaEmbedRequest struct {
	Model string `json:"model"`
	Input any    `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

// newEmbedRequest 构造 ollama /api/embed 请求（供 embed 调用与单元测试复用）
func (o *ollamaClient) newEmbedRequest(ctx context.Context, texts []string) (*http.Request, error) {
	url := strings.TrimRight(o.baseURL, "/") + "/api/embed"
	body, err := json.Marshal(ollamaEmbedRequest{Model: o.model, Input: texts})
	if err != nil {
		return nil, fmt.Errorf("marshal ollama embed request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("new ollama embed request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
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

// NewEmbedder 创建 Embedding 客户端，支持 dashscope（默认）与 ollama
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
		return &Embedder{
			model: cfg.Model,
			client: &ollamaClient{
				baseURL: cfg.BaseURL,
				model:   cfg.Model,
				client:  &http.Client{Timeout: 60 * time.Second},
			},
		}, nil
	default: // dashscope
		d := int(cfg.Dimension)
		embedder, err := dashscope.NewEmbedder(ctx, &dashscope.EmbeddingConfig{
			APIKey:     cfg.APIKey,
			Model:      cfg.Model,
			Dimensions: &d,
		})
		if err != nil {
			log.WithCtx(ctx).Error("NewEmbedder failed, init dashscope embedder err: %v", err)
			return nil, err
		}
		return &Embedder{
			model:    cfg.Model,
			client:   &dashscopeClient{Embedder: embedder},
			Embedder: embedder,
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
	return vectors[0], nil
}
