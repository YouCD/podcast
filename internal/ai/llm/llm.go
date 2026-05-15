package llm

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"podcast/pkg/types"

	"github.com/youcd/toolkit/log"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
)

// 全局 LLM 池实例（用于兼容旧代码）
var (
	globalPool     *LLMPool
	globalPoolOnce sync.Once
	globalPoolMu   sync.RWMutex
)

// LLMPool LLM 连接池，管理多个 LLM 提供商
type LLMPool struct {
	entries  []*llmEntry              // 所有可用配置
	limiters map[string]*rate.Limiter // key -> 独立限速器

	// 信号量控制并发度，防止把 provider 打满
	sem    *semaphore.Weighted
	badTTL time.Duration

	cooling   *sync.Map     // key -> time.Time (frozen until)
	baseDelay time.Duration // 基础退避时间
}

type llmEntry struct {
	info *types.LLMInfo
	key  string
}

// NewLLMPool 创建新的 LLM 连接池（推荐使用）
func NewLLMPool(llmConfigs []*types.LLMInfo) *LLMPool {
	entries := make([]*llmEntry, 0, len(llmConfigs))
	limiters := make(map[string]*rate.Limiter)
	seen := make(map[string]struct{})
	for _, cfg := range llmConfigs {
		key := fmt.Sprintf("%s|%s|%s", cfg.ApiKey, cfg.Model, cfg.GetBaseURL())
		// 去重：相同 key 的配置只保留第一个
		if _, ok := seen[key]; ok {
			log.WithCtx(context.Background()).Warnf("duplicate LLM config skipped: %s|%s", cfg.Model, cfg.GetBaseURL())
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, &llmEntry{
			info: cfg,
			key:  key,
		})
		// 每个提供商独立限速：10 QPS，突发 5
		limiters[key] = rate.NewLimiter(rate.Every(100*time.Millisecond), 10)
	}

	// 打乱顺序实现负载均衡
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})

	pool := &LLMPool{
		entries:  entries,
		limiters: limiters,
		// 并发度设为 2*len(entries)，既充分利用又不打满
		sem:       semaphore.NewWeighted(int64(len(entries) * 2)),
		cooling:   &sync.Map{},
		baseDelay: 4 * time.Hour, // 基础冷却 1h
	}

	// 设置全局池
	globalPoolMu.Lock()
	globalPool = pool
	globalPoolMu.Unlock()

	return pool
}

// GetLLMPool 获取全局 LLM 池实例（兼容旧代码）
// Deprecated: 推荐通过依赖注入使用 LLMPool
func GetLLMPool() *LLMPool {
	globalPoolMu.RLock()
	defer globalPoolMu.RUnlock()
	return globalPool
}

// Get 从池中获取一个可用的 LLM，支持重试和上下文取消
func (p *LLMPool) Get(ctx context.Context) (*types.LLMInfo, error) {
	// 获取执行许可（防止并发过高）
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return nil, fmt.Errorf("acquire semaphore: %w", err)
	}

	// 最多尝试所有 entry 一次，避免无限循环
	for i := 0; i < len(p.entries); i++ {
		select {
		case <-ctx.Done():
			p.sem.Release(1) // 记得释放
			return nil, ctx.Err()
		default:
		}

		idx := (i + rand.Intn(len(p.entries))) % len(p.entries) // 随机起始位置

		entry := p.entries[idx]

		// 等待该提供商的限速令牌
		limiter := p.limiters[entry.key]
		if err := limiter.Wait(ctx); err != nil {
			continue // 超时或取消，尝试下一个
		}

		log.WithCtx(ctx).Debugw("llmPool", "Get", p.getKey(entry.info))
		return entry.info, nil
	}

	p.sem.Release(1)
	return nil, fmt.Errorf("no available LLM provider (all marked bad or rate limited)")
}

// Put 释放资源（实际上是释放信号量，让其他等待者可以执行）
func (p *LLMPool) Put(ctx context.Context, info *types.LLMInfo) {
	// 不需要归还到 channel，直接释放信号量
	p.sem.Release(1)
	log.WithCtx(ctx).Debugw("llmPool", "Put", p.getKey(info))
}

// MarkRateLimited 专门处理 429，指数退避
func (p *LLMPool) MarkRateLimited(ctx context.Context, info *types.LLMInfo) {
	key := p.getKey(info)

	// 计算退避时间：基础时间 * (1 + 随机扰动)
	backoff := p.baseDelay + time.Duration(rand.Intn(3000))*time.Millisecond

	// 如果有已存在的冷却时间，延长它（简单指数退避）
	if existing, ok := p.cooling.Load(key); ok {
		if t := existing.(time.Time); t.After(time.Now()) {
			backoff = t.Sub(time.Now()) * 2 // 指数倍增
			if backoff > 2*time.Minute {    // 最大 2 分钟
				backoff = 2 * time.Minute
			}
		}
	}

	frozenUntil := time.Now().Add(backoff)
	p.cooling.Store(key, frozenUntil)

	log.WithCtx(ctx).Warnf("RateLimited (429) for %s, cooling until %v (backoff: %v)",
		key, frozenUntil.Format("15:04:05"), backoff)
}

func (p *LLMPool) getKey(info *types.LLMInfo) string {
	return fmt.Sprintf("%s|%s|%s", info.ApiKey, info.GetModelName(), info.GetBaseURL())
}
