package dgraph

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"podcast/config"

	"podcast/pkg/types"

	"github.com/dgraph-io/dgo/v250"
	"github.com/dgraph-io/dgo/v250/protos/api"
	jsoniter "github.com/json-iterator/go"
	"github.com/youcd/toolkit/log"
)

const (
	schema = `# ============================================
# 统一知识图谱 Schema
# 覆盖领域：科技AI + 金融商业 + 地缘政治
# 语法：Dgraph 标准 Schema 格式
# ============================================

# --------------------------------------------
# 基础属性谓词定义
# --------------------------------------------
<name>: string @index(exact) @upsert .
<aliases>: [string] @index(term) .
<数据来源领域>: string @index(exact) .
<数据质量评级>: string @index(exact) .

# --------------------------------------------
# 科技AI领域谓词
# --------------------------------------------
<研发>: [uid] @reverse .
<研发_置信度>: float .
<研发_证据>: string .
<研发_时间>: datetime .

<基于>: [uid] @reverse .
<基于_置信度>: float .
<基于_证据>: string .
<基于_时间>: datetime .

<训练于>: [uid] @reverse .
<训练于_置信度>: float .
<训练于_证据>: string .
<训练于_时间>: datetime .

<所属企业>: [uid] @reverse .

# --------------------------------------------
# 金融商业领域谓词
# --------------------------------------------
<投资>: [uid] @reverse .
<投资_置信度>: float .
<投资_证据>: string .
<投资_时间>: datetime .
<投资_金额>: string .
<投资_轮次>: string .

<收购>: [uid] @reverse .
<收购_置信度>: float .
<收购_证据>: string .
<收购_时间>: datetime .
<收购_比例>: float .

<竞争>: [uid] @reverse .
<竞争_置信度>: float .
<竞争_证据>: string .
<竞争_时间>: datetime .
<竞争_领域>: string .

<合作>: [uid] @reverse .
<合作_置信度>: float .
<合作_证据>: string .
<合作_时间>: datetime .
<合作_类型>: string .

<雇佣>: [uid] @reverse .
<雇佣_置信度>: float .
<雇佣_证据>: string .
<雇佣_时间>: datetime .
<雇佣_职位>: string .

<裁员>: [uid] @reverse .
<裁员_置信度>: float .
<裁员_证据>: string .
<裁员_时间>: datetime .
<裁员_规模>: string .

<掌管>: [uid] @reverse .
<掌管_置信度>: float .
<掌管_证据>: string .
<掌管_时间>: datetime .
<掌管_职位>: string .

<所属行业>: [uid] @reverse .

# --------------------------------------------
# 地缘政治领域谓词
# --------------------------------------------
<制裁>: [uid] @reverse .
<制裁_置信度>: float .
<制裁_证据>: string .
<制裁_时间>: datetime .
<制裁_类型>: string .
<制裁_范围>: string .

<结盟>: [uid] @reverse .
<结盟_置信度>: float .
<结盟_证据>: string .
<结盟_时间>: datetime .
<结盟_类型>: string .

<签署协议>: [uid] @reverse .
<签署协议_置信度>: float .
<签署协议_证据>: string .
<签署协议_时间>: datetime .
<签署协议_类型>: string .
<签署协议_状态>: string .

<影响行业>: [uid] @reverse .
<影响行业_置信度>: float .
<影响行业_证据>: string .
<影响行业_时间>: datetime .
<影响行业_方向>: string .

<部署于>: [uid] @reverse .
<部署于_置信度>: float .
<部署于_证据>: string .
<部署于_时间>: datetime .
<部署于_资产>: string .

<访问>: [uid] @reverse .
<访问_置信度>: float .
<访问_证据>: string .
<访问_时间>: datetime .
<访问_类型>: string .

<所属国家>: [uid] @reverse .
<所属组织>: [uid] @reverse .
<所属实体>: [uid] @reverse .

# --------------------------------------------
# 辅助谓词
# --------------------------------------------
<相关方>: [uid] @reverse .
<参与方>: [uid] @reverse .
<制定方>: [uid] @reverse .
<成员>: [uid] @reverse .

# --------------------------------------------
# 类型定义（Types）
# --------------------------------------------

# === 科技AI领域类型 ===
type <技术> {
	name
	aliases
	<研发>
	<基于>
	<训练于>
	<所属企业>
}

type <AI模型> {
	name
	aliases
	<研发>
	<基于>
	<训练于>
	<竞争>
	<所属企业>
}

type <计算设备> {
	name
	aliases
	<研发>
	<所属企业>
}

# === 金融商业领域类型 ===
type <企业>{
	name
	aliases
	<研发>
	<投资>
	<收购>
	<竞争>
	<合作>
	<雇佣>
	<裁员>
	<掌管>
	<所属行业>
}

type <投资机构> {
	name
	aliases
	<投资>
	<竞争>
	<合作>
}

type <行业> {
	name
	aliases
	<影响行业>
	<裁员>
}

type <市场事件> {
	name
	aliases
	<相关方>
	<影响行业>
}

# === 地缘政治领域类型 ===
type <国家> {
	name
	aliases
	<制裁>
	<结盟>
	<签署协议>
	<影响行业>
	<部署于>
	<访问>
	<所属组织>
}

type <国际组织> {
	name
	aliases
	<制裁>
	<结盟>
	<签署协议>
	<部署于>
	<成员>
}

type <政策> {
	name
	aliases
	<签署协议>
	<影响行业>
	<制定方>
}

type <冲突事件> {
	name
	aliases
	<参与方>
	<影响行业>
}

# === 跨领域通用类型 ===
type <人物> {
	name
	aliases
	<雇佣>
	<掌管>
	<访问>
	<所属实体>
}

type <政治领袖> {
	name
	aliases
	<掌管>
	<访问>
	<所属国家>
}

type <管理人员> {
	name
	aliases
	<雇佣>
	<掌管>
	<所属企业>
}
`
)

type Dgraph struct {
	client *dgo.Dgraph
}

func New() (*Dgraph, error) {
	client, err := dgo.Open(fmt.Sprintf("dgraph://%s", config.Cfg.Database.Dgraph))
	if err != nil {
		return nil, err
	}

	d := &Dgraph{client: client}

	//// 在初始化时设置一次 Schema
	//ctx := ctx
	//if err := d.client.SetSchema(ctx, schema); err != nil {
	//	return nil, fmt.Errorf("failed to set initial schema: %w", err)
	//}
	return d, nil
}
func (d *Dgraph) Init(ctx context.Context) error {
	if err := d.client.SetSchema(ctx, schema); err != nil {
		return fmt.Errorf("failed to set initial schema: %w", err)
	}
	return nil
}
func (d *Dgraph) Close() {
	d.client.Close()
}

func (d *Dgraph) GetClient() *dgo.Dgraph {
	return d.client
}

func (d *Dgraph) Insert(ctx context.Context, payload *types.DgraphPayload) error {
	var errs []error
	for _, node := range payload.Set {
		if node.DgraphType == "" {
			continue
		}
		//	创建节点
		_, err := d.UpsertEntity(ctx, node.Name, node.DgraphType, node.Aliases)
		if err != nil {
			log.WithCtx(ctx).Errorw("Insert", "node", node, "error", err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to insert nodes: %w", errors.Join(errs...))
	}
	err := d.UpsertRelation(ctx, payload)
	if err != nil {
		return fmt.Errorf("failed to insert relations: %w", err)
	}
	return nil
}

func (d *Dgraph) UpsertByName(ctx context.Context, name string, data map[string]any) error {
	query := `
	query {
	  q(func: eq(name, "` + name + `")) {
	    v as uid
	  }
	}
`

	// 把 data 注入 uid(v)
	data["uid"] = "uid(v)"

	setJSON, _ := json.Marshal(data)

	mu := &api.Mutation{
		SetJson: setJSON,
	}

	req := &api.Request{
		Query:     query,
		Mutations: []*api.Mutation{mu},
		CommitNow: true,
	}

	_, err := d.client.NewTxn().Do(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

// QueryOne 查询单个实体
func (d *Dgraph) QueryOne(ctx context.Context, name string) (*types.Node, error) {
	/*
		{
		  company(func: eq(name, "OpenAI")) {
		    uid
		    name
		    aliases
		  }
		}
	*/
	// 定义查询字符串
	dql := `
    query Company($name: string) {
        company(func: eq(name, $name)) {
            uid
            name
            aliases
        }
    }`

	// 定义变量映射
	vars := map[string]string{"$name": name}

	// 执行查询
	resp, err := d.client.NewTxn().QueryWithVars(ctx, dql, vars)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	var n types.Node
	toString := jsoniter.Get(resp.Json, "company").Get(0).ToString()
	err = jsoniter.Unmarshal([]byte(toString), &n)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON,str: %s ,err: %w", toString, err)
	}

	return &n, nil
}

func (d *Dgraph) UpsertEntity(ctx context.Context, name string, dtype string, newAliases []string) (string, error) {
	var alias []string
	for _, n := range newAliases {
		alias = append(alias, strings.ToLower(n))
	}

	// 1. 先查询现有实体和它的 aliases
	query := `
	query {
	  q(func: eq(name, "` + name + `")) {
	    uid
	    aliases
	  }
	}`

	resp, err := d.client.NewTxn().Query(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to query entity: %w", err)
	}

	var existing struct {
		Q []struct {
			UID     string   `json:"uid"`
			Aliases []string `json:"aliases"`
		} `json:"q"`
	}

	var uid string
	var finalAliases []string

	if err := json.Unmarshal(resp.Json, &existing); err != nil {
		return "", fmt.Errorf("failed to unmarshal: %w", err)
	}

	if len(existing.Q) > 0 {
		// 实体已存在，合并 aliases
		uid = existing.Q[0].UID
		existingAliases := existing.Q[0].Aliases

		// 使用 map 去重合并
		aliasSet := make(map[string]bool)
		for _, a := range existingAliases {
			aliasSet[a] = true
		}
		for _, a := range alias {
			aliasSet[a] = true
		}

		finalAliases = make([]string, 0, len(aliasSet))
		for a := range aliasSet {
			finalAliases = append(finalAliases, a)
		}

		// 使用具体的 uid 更新
		data := map[string]any{
			"uid":     uid,
			"aliases": finalAliases,
		}

		setJSON, _ := json.Marshal(data)
		mu := &api.Mutation{SetJson: setJSON}
		req := &api.Request{
			Mutations: []*api.Mutation{mu},
			CommitNow: true,
		}

		_, err = d.client.NewTxn().Do(ctx, req)
		if err != nil {
			return "", fmt.Errorf("failed to update aliases: %w", err)
		}

		return uid, nil
	} else {
		// 实体不存在，新建
		data := map[string]any{
			"uid":         "_:new" + name,
			"name":        name,
			"dgraph.type": dtype,
		}

		if len(newAliases) > 0 {
			data["aliases"] = newAliases
		}

		setJSON, _ := json.Marshal(data)
		mu := &api.Mutation{SetJson: setJSON}
		req := &api.Request{
			Mutations: []*api.Mutation{mu},
			CommitNow: true,
		}

		resp, err := d.client.NewTxn().Do(ctx, req)
		if err != nil {
			return "", fmt.Errorf("failed to create entity: %w", err)
		}

		for _, uid := range resp.Uids {
			return uid, nil
		}
		return "", fmt.Errorf("uid not found for new entity %s", name)
	}
}

func (d *Dgraph) UpsertRelation(ctx context.Context, payload *types.DgraphPayload) error {
	var errs []error
	for _, node := range payload.Set {
		if node.Name == "" {
			log.WithCtx(ctx).Warnw("UpsertRelation", "node", node, "err", "name为空")
			continue
		}
		// 查询节点信息
		fromNode, err := d.QueryOne(ctx, node.Name)
		if err != nil {
			log.WithCtx(ctx).Errorw("UpsertRelation", "name", node.Name, "error", err)
			errs = append(errs, err)
			continue
		}

		for s, edges := range node.Predicates {
			for _, edge := range edges {
				name := payload.GetNameByID(edge.Uid)
				if name == "" {
					log.WithCtx(ctx).Warnw("UpsertRelation", "Uid", edge.Uid, "predicates", s, "edge", edge, "err", "uid未找到")
					continue
				}
				toNode, err := d.QueryOne(ctx, name)
				if err != nil {
					log.WithCtx(ctx).Errorw("UpsertRelation", "Uid", edge.Uid, "name", name, "predicates", s, "edge", edge, "err", err)
					errs = append(errs, err)
					continue
				}
				if err = d.upsertRelation(ctx, fromNode.Uid, toNode.Uid, s, &edge); err != nil {
					log.WithCtx(ctx).Errorw("UpsertRelation", "error", err)
					errs = append(errs, err)
				}
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to upsert relations: %w", errors.Join(errs...))
	}
	return nil
}

func (d *Dgraph) upsertRelation(ctx context.Context, fromUID, toUID, predicate string, edge *types.Edge) error {
	// 构造 NQuad
	var setNquads strings.Builder

	// 主关系
	setNquads.WriteString(fmt.Sprintf(
		`<%s> <%s> <%s> .`+"\n",
		fromUID,
		predicate,
		toUID,
	))

	// 置信度
	if edge.Confidence != 0 {
		setNquads.WriteString(fmt.Sprintf(
			`<%s> <%s_置信度> "%f"^^<xs:float> .`+"\n",
			fromUID,
			predicate,
			edge.Confidence,
		))
	}

	// 证据
	if edge.Evidence != "" {
		setNquads.WriteString(fmt.Sprintf(
			`<%s> <%s_证据> "%s" .`+"\n",
			fromUID,
			predicate,
			edge.Evidence,
		))
	}

	// 时间
	if edge.Time != "" {
		layout := "2006-01-02 15:04:05 -0700 MST"
		// 解析时间
		parsedTime, err := time.Parse(layout, edge.Time)
		if err != nil {
			log.WithCtx(ctx).Errorw("upsertRelation", "Time", edge.Time, "err", err)
			parsedTime = time.Now()
		} // 转换为本地时区
		localTime := parsedTime.In(time.Local)
		setNquads.WriteString(fmt.Sprintf(
			`<%s> <%s_时间> "%s"^^<xs:dateTime> .`+"\n",
			fromUID,
			predicate,
			localTime.Format(time.RFC3339),
		))
	}
	req := &api.Request{
		Mutations: []*api.Mutation{
			{
				SetNquads: []byte(setNquads.String()),
			},
		},
		CommitNow: true,
	}

	_, err := d.client.NewTxn().Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to upsert relation: %w", err)
	}

	return nil
}

/*
func (d *Dgraph) QueryType(ctx context.Context, t string) error {
	// 动态构建查询语句
	dql := fmt.Sprintf(`
{
  data(func: type(%#v)) {
    uid
    name
    aliases
  }
}
	`, t)

	// 执行查询
	resp, err := d.client.NewTxn().Query(ctx, dql)
	if err != nil {
		return fmt.Errorf("failed to query: %w", err)
	}

	// 打印查询结果

	return nil
}

*/

func (d *Dgraph) QueryAliases(ctx context.Context, name string) (string, error) {
	// 动态构建查询语句
	dql := fmt.Sprintf(`
{
  q(func: eq(aliases, %#v)) {
    uid
    name
    aliases
	dgraph.type
  }
}
`, name)

	// 执行查询
	resp, err := d.client.NewTxn().Query(ctx, dql)
	if err != nil {
		return "", fmt.Errorf("failed to query: %w", err)
	}

	for _, uid := range resp.Uids {
		return uid, nil
	}
	//
	//var n types.Node
	//toString := jsoniter.Get(resp.Json, "company").Get(0).ToString()
	//err = jsoniter.Unmarshal([]byte(toString), &n)
	//if err != nil {
	//	return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	//}
	//
	//// 打印查询结果

	return "", nil
}

func (d *Dgraph) Query(ctx context.Context, dql string) (*types.DgraphResp, error) {
	// 执行查询
	resp, err := d.client.NewTxn().Query(ctx, dql)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}

	payload := struct {
		Q []types.DgraphNode `json:"q"`
	}{}
	err = json.Unmarshal(resp.Json, &payload)
	if err != nil {
		log.WithCtx(ctx).Errorw("Query", "error", err)
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	nodes := make(map[string]*types.DgraphNode)
	edges := []*types.DgraphEdge{}

	for _, node := range payload.Q {
		node.Category = 1
		node.ID = node.Name
		nodes[node.Name] = &node
		for p, predicate := range node.Predicates {
			for _, edge := range predicate {
				edge.Category = 4
				edge.ID = edge.Name
				nodes[edge.Name] = &edge
				edges = append(edges, &types.DgraphEdge{
					Source: node.Name,
					Target: edge.Name,
					Value:  p,
				})
				// fmt.Printf("%s -- [%s] -->  %s \n", node.Name, p, edge.Name)
			}
		}
	}

	// 打印查询结果
	var n []*types.DgraphNode
	for _, node := range nodes {
		n = append(n, node)
	}
	var r types.DgraphResp
	var hasData bool
	if len(n) > 0 {
		r.Nodes = n
	}
	if len(edges) > 0 {
		r.Edges = edges
		hasData = true
	}
	if !hasData {
		return nil, fmt.Errorf("no data found")
	}
	return &r, nil
}

func (d *Dgraph) Predicates(ctx context.Context, dql string) (string, error) {
	// 执行查询
	resp, err := d.client.NewTxn().Query(ctx, dql)
	if err != nil {
		return "", fmt.Errorf("failed to query: %w", err)
	}

	payload := struct {
		Q []types.DgraphNode `json:"q"`
	}{}
	err = json.Unmarshal(resp.Json, &payload)
	if err != nil {
		log.WithCtx(ctx).Errorw("Query", "error", err)
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	var predicates []string
	for _, node := range payload.Q {
		for p, predicate := range node.Predicates {
			for _, edge := range predicate {
				predicates = append(predicates, fmt.Sprintf("[%s - %s - %s]", node.Name, p, edge.Name))
			}
		}
	}

	return strings.Join(predicates, "\n"), nil
}
