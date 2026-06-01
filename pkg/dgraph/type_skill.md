# Dgraph Type 数据清理通用 Skill

## 环境

- Dgraph HTTP: `http://10.10.0.1:8080`
- Dgraph 版本: v25.2.0
- DQL 查询 Content-Type: `application/dql`
- 突变 Content-Type: `application/json`（需 `commitNow=true` 或手动 startTs + commit）

## 常用查询

### 查看所有 type

```dql
{ types(func: has(dgraph.type)) { dgraph.type } }
```

### 按 type 查实体

```dql
{ all(func: type("某Type")) { uid name dgraph.type aliases } }
```

### 查实体所有 predicates

```dql
{ q(func: uid(0x1234)) { uid expand(_all_) } }
```

### 批量查特定 UID

```dql
{ q(func: uid(0x1234, 0x5678)) { uid name dgraph.type } }
```

### 按条件搜索

```dql
{ q(func: regexp(name, /关键词/i)) { uid name dgraph.type } }
{ q(func: eq(name, ["A", "B"])) { uid name dgraph.type } }
```

## Type 操作

### 1. 实体移出 type（删除）

```python
requests.post(f"{URL}/mutate?commitNow=true", json={
    "delete": [{"uid": uid, "dgraph.type": "要删除的Type"}]
})
```

> 注意：仅移除 type 标签，UID 保留为空节点。实体不再出现在 `type("某Type")` 查询中。

### 2. 实体迁移至其他 type

```python
# 删除旧 type
requests.post(f"{URL}/mutate?commitNow=true", json={
    "delete": [{"uid": uid, "dgraph.type": "旧Type"}]
})
# 设置新 type
requests.post(f"{URL}/mutate?commitNow=true", json={
    "set": [{"uid": uid, "dgraph.type": "新Type"}]
})
```

> 可批量操作，一次 mutate 支持多个 uid。

### 3. 删除实体所有数据

```python
# 查出现有 predicates
r = requests.post(f"{URL}/query", json={
    "query": '{ q(func: uid(' + uid + ')) { uid expand(_all_) } }'
})
m = r.json()['data']['q'][0]
preds = [k for k in m.keys() if k != 'uid']

# 逐个置 null 删除
delete_obj = {"uid": uid}
for p in preds:
    delete_obj[p] = None
requests.post(f"{URL}/mutate?commitNow=true", json={"delete": [delete_obj]})

# 最终删除 type
requests.post(f"{URL}/mutate?commitNow=true", json={
    "delete": [{"uid": uid, "dgraph.type": "旧Type"}]
})
```

### 4. 新增自定义 type

```bash
curl -X POST "http://10.10.0.1:8080/alter" \
  --data-binary $'type <Type名称> {\n  name\n  aliases\n}'
```

> 中文 type 名需要使用 unicode 编码或 `--data-binary` 传输。

## 别名（aliases）清理

### 批量替换别名

> Dgraph 的 `set` 对 list 类型是追加而非替换，必须**先 delete 再 set**：

```python
# 删除所有 alias
requests.post(f"{URL}/mutate?commitNow=true", json={
    "delete": [{"uid": uid, "aliases": None}]
})
# 设置新列表
requests.post(f"{URL}/mutate?commitNow=true", json={
    "set": [{"uid": uid, "aliases": ["别名1", "别名2"]}]
})
```

### 别名脏数据检测规则

| 类型 | 检测方式 | 示例 |
|---|---|---|
| 论文标题 | 长度 > 50 或含冒号 `:` | `CineTrans: Learning to Generate...` |
| name 冗余 | 别名与 name 大小写一致 | name=`Grok`, alias=`Grok` |
| 内部重复 | 同一实体去重后仍有重复 | `grok` / `Grok` |
| 百分比/分数 | 含 `\d+%` | `Qwen-Image 60.6%` |
| 营销文案 | 含"智能"、"领航"、"革命性"、"业界领先" | `智能体基础模型MagicAgent` |
| 描述性文本 | 含"的"+"模型/平台/系统"且 > 20 字 | `口型、表情、眼神的多模态协同表达模型` |
| 无意义短名 | 1-2 字母无上下文 | `cc`, `DT` |

## 识别不属于当前 type 的实体

### 核心思路

> 基于 **实体 name 的关键词** 判断其属于哪类事物，对不属于当前 type 的实体做迁移或删除。

### 常见实体分类规则

| 类别 | 特征关键词 | 建议 type |
|---|---|---|
| **公司/组织** | 公司、集团、企业、实验室、研究院、研究所、内部、自研、品牌 | 删除 |
| **硬件/终端设备** | 手机、芯片、耳机、眼镜、手表、机器人、硬件、设备、AR | `终端设备` |
| **技术/架构** | Transformer、自回归、LSTM、LeNet、编码器、扩散语言 | `技术` |
| **人物** | 人名、skill 后缀 | `人物` |
| **OS/系统** | OS、系统、HarmonyOS、澎湃、ColorOS | 删除 |
| **游戏/APP** | 游戏、App、短剧、王者荣耀 | 删除 |
| **平台/引擎** | 平台、引擎、框架、框架 | `AI智能体` 或 `技术` |
| **泛化类别词** | XX 模型、XX 助手、XX 系统（仅泛化名词） | 删除 |
| **AI 智能体** | 智能体、Agent、助手、AI 系统、AI 平台 | `AI智能体` |

### 扫描脚本模板

```python
issues = []
for m in models:
    name = m['name']
    # 按关键词匹配
    if any(kw in name for kw in ['关键词1', '关键词2']):
        issues.append((m['uid'], name, "类别名"))
```

## type 统计

```python
r = requests.post(f"{URL}/query", json={
    "query": '{ types(func: has(dgraph.type)) { dgraph.type } }'
})
for m in r.json()['data']['types']:
    for t in m.get('dgraph.type', []):
        counts[t] = counts.get(t, 0) + 1
```

## 本次清理记录

**目标 type**: `AI模型`（805 → **691**）
**新增 type**: `AI智能体`（9 条）

| 操作 | 数量 | 明细 |
|---|---|---|
| 删除难识别短名实体 | 7 | M6, rCM, ACT, KAI, 氢离子, VDR, 喵记多 |
| 清理别名脏数据 | 249 实体 | 论文标题 + name冗余 + 内部重复 |
| 删除公司/组织 | 9 | 华为AI模型、Meta下一代AI模型等 |
| 删除泛化类别词 | 30 | AI 助手、视觉语言模型等 |
| 重设为 `技术` | 8 | Transformer、自回归模型、LeNet 等 |
| 重设为 `人物` | 3 | 张雪峰.skill 等 |
| 重设为 `终端设备` | 32 | 麒麟芯片、nova手机、MIX Fold 5 等 |
| OS/系统 移出 | 4 | ColorOS 16、澎湃OS、HarmonyOS NEXT、小牛灵犀AIOS |
| 重设为 `AI智能体` | 9 | OpenClaw、Anthropic AI 系统、智己汽车超级智能体等 |
