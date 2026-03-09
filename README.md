# Podcast - AI 驱动的智能 RSS 内容分析平台

<img src="frontend/public/assets/favicon.png" alt="Podcast Logo" width="120" height="120">

一个基于 Go 语言开发的智能化 RSS 内容聚合与分析平台。通过 AI 技术对 RSS 内容进行智能分类、深度分析、知识图谱构建，提供个性化推荐与智能问答服务。

---

## 目录

- [核心特性](#核心特性)
- [技术架构](#技术架构)
- [工作流程](#工作流程)
- [项目结构](#项目结构)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [API 文档](#api-文档)
- [MCP 协议](#mcp-协议)
- [数据库设计](#数据库设计)
- [部署指南](#部署指南)
- [开发指南](#开发指南)

---

## 核心特性

### 智能内容处理

- **AI 内容分类** - 基于大语言模型自动识别内容类别，支持自定义分类规则
- **深度内容分析** - 自动提取关键信息、生成摘要、实体识别
- **智能去重** - 基于 MD5 内容指纹与向量相似度的双重去重机制
- **内容质量评估** - 评估可读性、信息价值与时效性

### 知识图谱

- **自动关系抽取** - 识别内容中的实体及其关系
- **图谱可视化** - 直观展示知识网络结构
- **关联推荐** - 基于图谱的智能内容关联

### 智能问答

- **RAG 检索增强** - 基于向量数据库的语义搜索
- **多轮对话** - 支持上下文理解的连续问答
- **答案溯源** - 提供答案来源引用

### 报告生成

- **自动报告** - 定时生成日报、周报、月报
- **播客转换** - 自动将报告转换为音频播客
- **自定义主题** - 支持配置个性化分析主题

---

## 技术架构

### 后端技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 核心框架 | Go 1.25+ | 高性能编程语言 |
| Web 框架 | Gin | 高性能 HTTP 路由框架 |
| ORM | GORM | 全功能对象关系映射 |
| AI 框架 | Eino | CloudWeGo AI 应用开发框架 |
| 向量数据库 | Milvus | 高性能向量检索引擎 |
| 图数据库 | Dgraph | 分布式图数据库 |
| 认证 | JWT | 无状态身份认证 |

### 前端技术栈

| 组件 | 技术选型 | 说明 |
|------|----------|------|
| 框架 | Vue 3 | 渐进式前端框架 |
| 语言 | TypeScript | 类型安全 |
| 构建工具 | Vite | 下一代前端构建工具 |
| 状态管理 | Pinia | Vue 3 官方状态管理 |
| 路由 | Vue Router | 单页面应用路由 |

### AI 服务

- **大语言模型**: 支持 OpenAI 兼容 API (GPT、GLM、Qwen 等)
- **向量嵌入**: DashScope / OpenAI Embedding
- **语音合成**: 字节跳动 TTS API

---

## 工作流程

RSS 内容处理采用基于 Eino Framework 的流式 AI 工作流：

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   RSS 源    │───▶│  内容抓取   │───▶│  日期筛选   │───▶│  智能去重   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                               │
┌─────────────┐    ┌─────────────┐    ┌─────────────┐           │
│  结构化存储 │◀───│  向量存储   │◀───│  知识图谱   │◀──────────┘
└─────────────┘    └─────────────┘    └─────────────┘
      │                  │                  │
      ▼                  ▼                  ▼
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  前端展示   │    │  语义检索   │    │  关联分析   │
└─────────────┘    └─────────────┘    └─────────────┘
```

### 工作流节点说明

| 节点 | 功能 | 说明 |
|------|------|------|
| fetch_rss | 内容抓取 | 从配置的 RSS 源拉取最新内容 |
| filter_today | 日期筛选 | 过滤出指定时间范围内的内容 |
| deduplicate | 智能去重 | 基于 MD5 和向量相似度去重 |
| categorization | 内容分类 | AI 自动识别内容类别 |
| analyze_rss | 深度分析 | 提取关键信息、生成摘要 |
| dgraph | 图谱构建 | 提取实体关系、构建知识图谱 |
| save | 持久化存储 | 保存到 MySQL、Milvus、Dgraph |

---

## 项目结构

```
podcast/
├── cmd/                          # 命令行入口
│   ├── root.go                   # 根命令 (服务启动)
│   ├── version.go                # 版本信息
│   ├── set.go                    # 配置设置
│   ├── initialization.go         # 初始化命令
│   └── tools/                    # 辅助工具
│       ├── Indexer/              # 向量索引工具
│       ├── dgraph/               # 图数据库工具
│       ├── query/                # 查询工具
│       └── llm/                  # LLM 测试工具
│
├── config/                       # 配置管理
│   ├── config.go                 # 配置结构与加载
│   └── config_test.go            # 配置测试
│
├── frontend/                     # Vue 3 前端应用
│   ├── src/
│   │   ├── api/                  # API 客户端
│   │   ├── components/           # Vue 组件
│   │   ├── views/                # 页面视图
│   │   ├── stores/               # Pinia 状态管理
│   │   ├── router/               # 路由配置
│   │   └── types/                # TypeScript 类型
│   ├── package.json
│   └── vite.config.ts
│
├── internal/                     # 核心业务逻辑
│   ├── ai/                       # AI 服务模块
│   │   ├── common/               # 通用组件
│   │   ├── embedding/            # 向量嵌入
│   │   ├── llm/                  # 大语言模型封装
│   │   ├── milvus/               # Milvus 操作
│   │   ├── mcp/                  # MCP 协议实现
│   │   ├── rag/                  # RAG 检索增强
│   │   ├── agent/                # AI Agent
│   │   ├── workflow/             # AI 处理工作流
│   │   └── report/               # 报告生成
│   │       ├── daily/            # 日报工作流
│   │       └── weekday_month/    # 周报月报
│   │
│   ├── database/                 # 数据持久层
│   │   ├── dao/                  # 数据访问对象
│   │   └── models/               # 数据模型
│   │
│   ├── service/                  # 业务服务层
│   └── app/                      # 应用容器 (依赖注入)
│
├── pkg/                          # 公共可复用模块
│   ├── cron/                     # 定时任务调度
│   ├── dgraph/                   # Dgraph 封装
│   ├── template/                 # 模板引擎
│   ├── types/                    # 公共类型定义
│   └── byte_dance/               # 字节跳动 TTS
│
├── web/                          # Web 服务层
│   ├── handlers/                 # HTTP 处理器
│   ├── routes/                   # 路由配置
│   └── dist/                     # 前端构建产物
│
├── main.go                       # 应用入口
├── Makefile                      # 构建脚本
├── go.mod                        # Go 模块定义
└── .golangci.yaml                # 代码检查配置
```

---

## 快速开始

### 环境要求

- **Go**: 1.25+
- **Node.js**: 20.19.0+ 或 22.12.0+
- **MySQL**: 8.0+
- **Milvus**: 2.4+
- **Dgraph**: 23.0+ (可选)

### 安装步骤

**1. 克隆项目**

```bash
git clone <repository-url>
cd podcast
```

**2. 安装依赖**

```bash
# Go 依赖
go mod tidy

# 前端依赖
cd frontend && npm install && cd ..
```

**3. 数据库准备**

```sql
CREATE DATABASE IF NOT EXISTS rss CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

**4. 配置文件**

创建 `config/config.yaml`，参考 [配置说明](#配置说明) 章节。

**5. 运行服务**

```bash
# 开发模式
make run

# 或手动构建运行
make build
./bin/podcast -f config/config.yaml
```

### 访问服务

- **Web 界面**: http://localhost:8080
- **前端开发**: http://localhost:5173 (需要 `cd frontend && npm run dev`)

---

## 配置说明

配置文件路径：`config/config.yaml`

### 完整配置示例

```
# 数据库配置
database:
  mysql:
    host: "localhost"
    port: 3306
    user: "root"
    password: "password"
    dbName: "rss"
  dgraph: "localhost:9080"
  milvus:
    endpoint: "localhost:19530"
    apiKey: ""
    dbName: "podcast"
    rssCollection: "rss_content"
    dedupCollection: "content_dedup"
    dimension: 1024
    score: 0.85

# 大语言模型配置 (支持多个，自动负载均衡)
LLM:
  - apiKey: "your-api-key"
    baseURL: "https://api.openai.com/v1"
    model: "gpt-4"
    prompt: "你是一个专业的技术分析师"

# 向量嵌入配置
vector:
  embedding:
    apiKey: "your-api-key"
    baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1"
    model: "text-embedding-v3"

# RSS 源配置
rss:
  - name: "IT 之家"
    url: "https://www.ithome.com/rss"
    disable: false
  - name: "InfoQ 推荐"
    url: "https://www.infoq.cn/feed"
    disable: false

# 全局配置
global:
  logLevel: "info"              # debug, info, warn, error
  logFile: "/var/log/podcast.log"
  hostPort: "0.0.0.0:8080"
  contentLen: 1400              # 内容处理长度阈值
  token: "your-mcp-token"       # MCP 协议令牌
  podcastDir: "/data/podcasts"  # 播客存储目录

# 定时报告配置
report:
  - schedule: "0 0 * * 1"       # 每周一
    topic: "本周 AI 发展趋势"
  - schedule: "0 0 1 * *"       # 每月 1 号
    topic: "本月技术热点回顾"

# 认证配置
authentication:
  jwtSecret: "your-jwt-secret-key"

# MCP 代理配置
mcpProxy:
  filesystem:
    command: "mcp-filesystem"
    args: ["/data"]
  browser:
    command: "mcp-browser"
    args: []

# 字节跳动 TTS 配置 (播客功能)
byteDance:
  appID: "your-app-id"
  accessToken: "your-access-token"
```

### 配置项详解

| 配置项 | 必填 | 说明 |
|--------|------|------|
| database.mysql | 是 | MySQL 连接配置 |
| database.milvus | 是 | Milvus 向量库配置 |
| LLM | 是 | 大语言模型配置，支持多实例 |
| vector.embedding | 是 | 向量嵌入模型配置 |
| rss | 是 | RSS 订阅源列表 |
| global.hostPort | 是 | 服务监听地址 |
| global.logLevel | 否 | 日志级别，默认 info |

---

## API 文档

### 基础路径

所有 API 以 `/api` 为前缀，需要认证的接口需在请求头携带：

```
Authorization: Bearer <jwt_token>
```

### 内容管理

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /api/feed | 获取内容列表 | 否 |
| GET | /api/feed/:id | 获取单条内容 | 否 |
| GET | /api/feed/:id/llm_html | 获取 AI 分析的 HTML | 否 |
| GET | /api/feed/categories | 获取所有分类 | 否 |
| GET | /api/feed/categories/:category/24h | 获取分类 24h 内容 | 否 |
| GET | /api/feed/read24h | 获取 24h 已读内容 | 否 |
| GET | /api/feed/not_read | 获取未读内容 | 否 |
| PUT | /api/feed/:id | 更新内容状态 | 是 |
| POST | /api/feed/time_stay | 记录阅读时长 | 是 |

### 报告管理

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /api/reports | 获取报告列表 | 否 |
| GET | /api/reports/:id/llm_result | 获取报告内容 | 否 |
| GET | /api/reports/:id/detail | 获取报告详情 | 否 |
| GET | /api/reports/:id/play | 获取播客音频 | 否 |
| GET | /api/reports/:id/daily_report | 手动生成日报 | 否 |

### 智能对话

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | /api/chat/stream | 流式 AI 对话 | 是 |
| GET | /api/chat/user | 获取用户会话列表 | 是 |
| POST | /api/chat/session/:session_id | 创建会话 | 是 |
| GET | /api/chat/session/:session_id | 获取会话历史 | 是 |
| DELETE | /api/chat/session/:session_id | 删除会话 | 是 |

### 用户认证

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| POST | /api/user/login | 用户登录 | 否 |

### 配置管理

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /api/keyinfos | 获取配置列表 | 是 |
| POST | /api/keyinfos | 创建配置 | 是 |
| PUT | /api/keyinfos/:id | 更新配置 | 是 |
| DELETE | /api/keyinfos/:id | 删除配置 | 是 |

### 提示词与模板

| 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|
| GET | /api/prompt | 获取提示词列表 | 是 |
| POST | /api/prompt | 创建提示词 | 是 |
| GET | /api/template | 获取模板列表 | 是 |
| POST | /api/template | 创建模板 | 是 |

---

## MCP 协议

项目集成 [MCP (Model Context Protocol)](https://modelcontextprotocol.io/) 协议，提供标准化的 AI 工具接口。

### 可用工具

#### news_search

搜索 24 小时内的新闻内容。

**参数：**
- `categories` (string, 可选): 内容分类，默认 "科技"

**示例：**
```json
{
  "name": "news_search",
  "arguments": {
    "categories": "AI"
  }
}
```

#### news_categories

获取所有可用的新闻分类。

#### get_current_time

获取当前时间。

#### rag_search

基于向量数据库的语义检索。

**参数：**
- `query` (string): 搜索查询

### 使用场景

- **智能助手集成** - 接入 Claude、ChatGPT 等平台
- **自动化工作流** - 与 n8n、Zapier 等平台集成
- **API 扩展** - 为其他应用提供内容检索能力

---



## 部署指南

## 向量数据库申请
 
1. 访问 [Milvus](https://cloud.zilliz.com/ ) 官网。  免费5G空间

## 模型申请

1. 访问 [ModelScope](https://modelscope.cn/my/access/token) 官网。   `2000/每日`免费

## 豆包语音播客大模型


1. 访问 [火山](https://www.volcengine.com/docs/6561/1668014?lang=zh) 官网。   `1000000 tokens`免费

## 网络搜索


1. 访问 [百炼](https://bailian.console.aliyun.com/cn-beijing/?tab=app#/mcp-market/detail/WebSearch) 官网。   `2000次/每月`免费
2. 访问 [百炼](https://bailian.console.aliyun.com/cn-beijing/?tab=model#/model-market/detail/text-embedding-v4) 官网。   `embedding` 模型

### 修改配置文件
[config.yaml](config/config.yaml)
```shell
vim config/config.yaml
```
### Docker 部署



使用 `deploy` 目录下的 Docker Compose 配置一键部署所有服务。

```bash

docker-compose up -d
```



---

## 开发指南

### 本地开发

```bash
# 后端开发
go run main.go -f config/config.yaml

# 前端开发
cd frontend && npm run dev

# 代码检查
make check

# 运行测试
go test -v ./...
```

### 添加新的 RSS 源

在 `config.yaml` 中添加：

```yaml
rss:
  - name: "新源名称"
    url: "https://example.com/rss"
    disable: false
```

### 自定义内容分类

分类提示词存储在 `key_info` 表中，`genre = 1`。

### 自定义报告模板

报告提示词存储在 `key_info` 表中，`genre = 4`，`keyname` 为报告主题。


## 常见问题

**Q: 如何修改默认端口？**

A: 修改配置文件中 `global.hostPort` 的值。

**Q: 如何添加新的 LLM 提供商？**

A: 在配置文件的 `LLM` 数组中添加新配置，系统会自动负载均衡。

**Q: 播客功能如何启用？**

A: 配置 `byteDance` 节点的 `appID` 和 `accessToken`。

**Q: 如何查看详细日志？**

A: 设置 `global.logLevel: "debug"`。

---

## 贡献指南

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

### 代码规范

- 使用 `make check` 进行代码检查
- 遵循 Go 代码规范
- 添加必要的单元测试

---

## 许可证

本项目采用 MIT 许可证。

---

<p align="center">
  Made with ❤️ by Podcast Team
</p>
