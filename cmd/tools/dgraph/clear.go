// main.go - Dgraph知识图谱关系方向清洗工具
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/dgraph-io/dgo/v250"
	"github.com/dgraph-io/dgo/v250/protos/api"
	"google.golang.org/grpc"
)

// RelationRule 定义了一个关系（谓词）的验证规则
type RelationRule struct {
	AllowedSourceTypes []string
	AllowedTargetTypes []string
}

// ValidationRules 存储所有谓词的验证规则
// 规则来源：统一知识图谱 Schema（科技AI + 金融商业 + 地缘政治）
var ValidationRules = map[string]RelationRule{
	// ============================================
	// 科技AI领域谓词
	// ============================================
	"<研发>": {
		// 企业/人物 研发 技术/AI模型/计算设备
		AllowedSourceTypes: []string{"企业", "人物"},
		AllowedTargetTypes: []string{"技术", "AI模型", "计算设备"},
	},
	"<基于>": {
		// AI模型/技术 基于 技术/AI模型/计算设备/企业/标准
		AllowedSourceTypes: []string{"AI模型", "技术"},
		AllowedTargetTypes: []string{"技术", "AI模型", "计算设备", "企业", "标准"},
	},
	"<训练于>": {
		// AI模型 训练于 数据集/计算设备
		AllowedSourceTypes: []string{"AI模型"},
		AllowedTargetTypes: []string{"计算设备", "数据集"},
	},
	"<所属企业>": {
		// 人物/AI模型/技术/计算设备 所属企业 企业
		AllowedSourceTypes: []string{"人物", "AI模型", "技术", "计算设备", "管理人员"},
		AllowedTargetTypes: []string{"企业"},
	},

	// ============================================
	// 金融商业领域谓词
	// ============================================
	"<投资>": {
		// 企业/投资机构 投资 企业
		AllowedSourceTypes: []string{"企业", "投资机构"},
		AllowedTargetTypes: []string{"企业"},
	},
	"<收购>": {
		// 企业 收购 企业
		AllowedSourceTypes: []string{"企业"},
		AllowedTargetTypes: []string{"企业"},
	},
	"<竞争>": {
		// 企业/投资机构 竞争 企业/投资机构（双向对称）
		AllowedSourceTypes: []string{"企业", "投资机构", "AI模型"},
		AllowedTargetTypes: []string{"企业", "投资机构", "AI模型"},
	},
	"<合作>": {
		// 企业/投资机构 合作 企业/投资机构（双向对称）
		AllowedSourceTypes: []string{"企业", "投资机构"},
		AllowedTargetTypes: []string{"企业", "投资机构"},
	},
	"<雇佣>": {
		// 企业 雇佣 人物/管理人员
		AllowedSourceTypes: []string{"企业"},
		AllowedTargetTypes: []string{"人物", "管理人员"},
	},
	"<裁员>": {
		// 企业 裁员 行业（表示裁员影响的行业）
		AllowedSourceTypes: []string{"企业"},
		AllowedTargetTypes: []string{"行业", "人物"},
	},
	"<掌管>": {
		// 人物/管理人员/政治领袖 掌管 企业/国家
		AllowedSourceTypes: []string{"人物", "管理人员", "政治领袖"},
		AllowedTargetTypes: []string{"企业", "国家"},
	},
	"<所属行业>": {
		// 企业 所属行业 行业
		AllowedSourceTypes: []string{"企业"},
		AllowedTargetTypes: []string{"行业"},
	},

	// ============================================
	// 地缘政治领域谓词
	// ============================================
	"<制裁>": {
		// 国家/国际组织 制裁 国家/行业/企业
		AllowedSourceTypes: []string{"国家", "国际组织"},
		AllowedTargetTypes: []string{"国家", "行业", "企业"},
	},
	"<结盟>": {
		// 国家 结盟 国家/国际组织（双向对称）
		// 国际组织 结盟 国家（如加入北约）
		AllowedSourceTypes: []string{"国家", "国际组织"},
		AllowedTargetTypes: []string{"国家", "国际组织"},
	},
	"<签署协议>": {
		// 国家/国际组织 签署协议 政策
		AllowedSourceTypes: []string{"国家", "国际组织"},
		AllowedTargetTypes: []string{"政策"},
	},
	"<影响行业>": {
		// 政策/国家/市场事件 影响行业 行业
		AllowedSourceTypes: []string{"政策", "国家", "市场事件", "冲突事件"},
		AllowedTargetTypes: []string{"行业"},
	},
	"<部署于>": {
		// 国家/国际组织 部署于 国家（军事/维和部署）
		AllowedSourceTypes: []string{"国家", "国际组织"},
		AllowedTargetTypes: []string{"国家"},
	},
	"<访问>": {
		// 政治领袖 访问 国家
		AllowedSourceTypes: []string{"政治领袖"},
		AllowedTargetTypes: []string{"国家"},
	},
	"<所属国家>": {
		// 企业/人物/政治领袖 所属国家 国家
		AllowedSourceTypes: []string{"企业", "人物", "政治领袖", "政策"},
		AllowedTargetTypes: []string{"国家"},
	},
	"<所属组织>": {
		// 国家 所属组织 国际组织
		AllowedSourceTypes: []string{"国家"},
		AllowedTargetTypes: []string{"国际组织"},
	},
	"<成员>": {
		// 国际组织 成员 国家
		AllowedSourceTypes: []string{"国际组织"},
		AllowedTargetTypes: []string{"国家"},
	},
	"<参与方>": {
		// 冲突事件 参与方 国家
		AllowedSourceTypes: []string{"冲突事件"},
		AllowedTargetTypes: []string{"国家", "国际组织"},
	},
	"<制定方>": {
		// 政策 制定方 国家/国际组织
		AllowedSourceTypes: []string{"政策"},
		AllowedTargetTypes: []string{"国家", "国际组织"},
	},
}

// FixAction 定义修复动作类型
type FixAction string

const (
	FixActionSwap    FixAction = "swap"    // 可通过交换修复
	FixActionDelete  FixAction = "delete"  // 无法修复，需要删除
	FixActionUnknown FixAction = "unknown" // 类型未知，需要人工处理
)

// EdgeToFix 代表一个需要修复的边
type EdgeToFix struct {
	Predicate   string
	SubjectUID  string
	ObjectUID   string
	SubjectType string
	ObjectType  string
	FixAction   FixAction // 修复动作
}

// FixReport 修复报告
type FixReport struct {
	TotalFound       int
	CanFixBySwap     int
	NeedDelete       int
	NeedManualReview int
	SkipEmptyType    int
}

// MergeReport 合并报告
type MergeReport struct {
	SourceUID        string
	TargetUID        string
	SourceName       string
	TargetName       string
	AliasesMigrated  int // 迁移的别名数量
	OutEdgesMigrated int // 迁移的出边数量
	InEdgesMigrated  int // 迁移的入边数量
	Success          bool
	Error            string
}

// NodeInfo 节点信息
type NodeInfo struct {
	UID      string
	Name     string
	Type     string
	Aliases  []string
	OutEdges []EdgeInfo // 出边
	InEdges  []EdgeInfo // 入边
}

// EdgeInfo 边信息
type EdgeInfo struct {
	Predicate string
	TargetUID string
	SourceUID string
}

// DgraphCleaner 直接在 Dgraph 上进行清洗
type DgraphCleaner struct {
	dg     *dgo.Dgraph
	rules  map[string]RelationRule
	dryRun bool // 是否为演练模式（只打印，不执行）
}

// NewDgraphCleaner 创建一个新的 DgraphCleaner 实例
func NewDgraphCleaner(dgraphAddr string, dryRun bool) (*DgraphCleaner, error) {
	// 连接到 Dgraph
	conn, err := grpc.Dial(dgraphAddr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("连接 Dgraph 失败: %w", err)
	}

	dgraphClient := dgo.NewDgraphClient(api.NewDgraphClient(conn))
	return &DgraphCleaner{
		dg:     dgraphClient,
		rules:  ValidationRules,
		dryRun: dryRun,
	}, nil
}

// Close 关闭连接

// Run 执行清洗过程
func (c *DgraphCleaner) Run() error {
	log.Println("开始查找需要修复的边...")
	edgesToFix, report, err := c.findIncorrectEdges()
	if err != nil {
		return fmt.Errorf("查找错误边时失败: %w", err)
	}

	// 打印报告
	c.printReport(report)

	if len(edgesToFix) == 0 {
		log.Println("恭喜！没有发现需要修复的边。")
		return nil
	}

	if c.dryRun {
		log.Println("--- 演练模式：以下边将被处理，但不会实际执行 ---")
		for _, edge := range edgesToFix {
			switch edge.FixAction {
			case FixActionSwap:
				log.Printf(" - [交换] %s (%s) -> %s -> %s (%s) => %s (%s) -> %s -> %s (%s)",
					edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType,
					edge.ObjectUID, edge.ObjectType, edge.Predicate, edge.SubjectUID, edge.SubjectType)
			case FixActionDelete:
				log.Printf(" - [删除] %s (%s) -> %s -> %s (%s) [无法通过交换修复]",
					edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType)
			case FixActionUnknown:
				log.Printf(" - [跳过] %s (%s) -> %s -> %s (%s) [类型未知，需人工处理]",
					edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType)
			}
		}
		return nil
	}

	return c.fixEdgesInBatches(edgesToFix)
}

// printReport 打印修复报告
func (c *DgraphCleaner) printReport(report FixReport) {
	log.Println("========== 修复报告 ==========")
	log.Printf("发现错误边总数: %d", report.TotalFound)
	log.Printf("可通过交换修复: %d", report.CanFixBySwap)
	log.Printf("需要删除的边:   %d", report.NeedDelete)
	log.Printf("需人工处理:     %d", report.NeedManualReview)
	log.Printf("跳过(类型为空): %d", report.SkipEmptyType)
	log.Println("==============================")
}

// findIncorrectEdges 查询并识别所有错误的边
func (c *DgraphCleaner) findIncorrectEdges() ([]EdgeToFix, FixReport, error) {
	var edgesToFix []EdgeToFix
	var report FixReport

	// 为每个需要检查的谓词执行查询
	for predicate := range c.rules {
		query := fmt.Sprintf(`
        {
            q(func: has(%s)) {
                uid
                dgraph.type
                %s {
                    uid
                    dgraph.type
                }
            }
        }`, predicate, predicate)

		txn := c.dg.NewReadOnlyTxn()
		defer txn.Discard(context.Background())

		resp, err := txn.Query(context.Background(), query)
		if err != nil {
			return nil, report, fmt.Errorf("查询谓词 '%s' 失败: %w", predicate, err)
		}

		type Node struct {
			UID        string   `json:"uid"`
			DgraphType []string `json:"dgraph.type"`
		}

		type QueryResult struct {
			Q []struct {
				UID        string   `json:"uid"`
				DgraphType []string `json:"dgraph.type"`
				Children   []Node   `json:"%s"` // 使用占位符
			} `json:"q"`
		}

		// 动态解析 JSON
		var result map[string]interface{}
		if err := json.Unmarshal(resp.Json, &result); err != nil {
			return nil, report, err
		}

		// 手动解析，因为字段名是动态的
		if qList, ok := result["q"].([]interface{}); ok {
			for _, item := range qList {
				if subjectMap, ok := item.(map[string]interface{}); ok {
					subjectUID, _ := subjectMap["uid"].(string)
					subjectTypes, _ := subjectMap["dgraph.type"].([]interface{})
					subjectType := ""
					if len(subjectTypes) > 0 {
						subjectType, _ = subjectTypes[0].(string)
					}

					if children, ok := subjectMap[predicate].([]interface{}); ok {
						for _, child := range children {
							if objectMap, ok := child.(map[string]interface{}); ok {
								objectUID, _ := objectMap["uid"].(string)
								objectTypes, _ := objectMap["dgraph.type"].([]interface{})
								objectType := ""
								if len(objectTypes) > 0 {
									objectType, _ = objectTypes[0].(string)
								}

								// 检查边是否错误，并确定修复动作
								if action, isIncorrect := c.analyzeEdge(predicate, subjectType, objectType); isIncorrect {
									report.TotalFound++

									// 如果主体或客体类型为空，跳过并记录
									if subjectType == "" || objectType == "" {
										report.SkipEmptyType++
										report.NeedManualReview++
										edgesToFix = append(edgesToFix, EdgeToFix{
											Predicate:   predicate,
											SubjectUID:  subjectUID,
											ObjectUID:   objectUID,
											SubjectType: subjectType,
											ObjectType:  objectType,
											FixAction:   FixActionUnknown,
										})
										continue
									}

									edgesToFix = append(edgesToFix, EdgeToFix{
										Predicate:   predicate,
										SubjectUID:  subjectUID,
										ObjectUID:   objectUID,
										SubjectType: subjectType,
										ObjectType:  objectType,
										FixAction:   action,
									})

									switch action {
									case FixActionSwap:
										report.CanFixBySwap++
									case FixActionDelete:
										report.NeedDelete++
									case FixActionUnknown:
										report.NeedManualReview++
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return edgesToFix, report, nil
}

// analyzeEdge 分析边是否错误，并返回修复动作
func (c *DgraphCleaner) analyzeEdge(predicate, subjectType, objectType string) (FixAction, bool) {
	rule, exists := c.rules[predicate]
	if !exists {
		return FixActionUnknown, false
	}

	// 检查当前边是否正确
	subjectValid := c.isTypeInList(subjectType, rule.AllowedSourceTypes)
	objectValid := c.isTypeInList(objectType, rule.AllowedTargetTypes)

	// 如果边是正确的，不需要修复
	if subjectValid && objectValid {
		return FixActionUnknown, false
	}

	// 如果类型为空，无法判断
	if subjectType == "" || objectType == "" {
		return FixActionUnknown, true
	}

	// 检查交换后是否正确
	swappedSubjectValid := c.isTypeInList(objectType, rule.AllowedSourceTypes)
	swappedObjectValid := c.isTypeInList(subjectType, rule.AllowedTargetTypes)

	if swappedSubjectValid && swappedObjectValid {
		return FixActionSwap, true
	}

	// 交换后仍然不正确，需要删除
	return FixActionDelete, true
}

// isTypeInList 检查类型是否在允许列表中
func (c *DgraphCleaner) isTypeInList(typeName string, allowedTypes []string) bool {
	for _, t := range allowedTypes {
		if typeName == t {
			return true
		}
	}
	return false
}

// isEdgeIncorrect 根据规则检查边是否错误（保留用于向后兼容）
func (c *DgraphCleaner) isEdgeIncorrect(predicate, subjectType, objectType string) bool {
	rule, exists := c.rules[predicate]
	if !exists {
		return false
	}

	subjectValid := false
	for _, t := range rule.AllowedSourceTypes {
		if subjectType == t {
			subjectValid = true
			break
		}
	}

	objectValid := false
	for _, t := range rule.AllowedTargetTypes {
		if objectType == t {
			objectValid = true
			break
		}
	}

	// 如果主体或客体类型不合法，说明方向错了
	return !subjectValid || !objectValid
}

// fixEdgesInBatches 分批修复边
func (c *DgraphCleaner) fixEdgesInBatches(edges []EdgeToFix) error {
	const batchSize = 100 // 每批处理 100 条边

	// 分离不同类型的边
	var swapEdges, deleteEdges []EdgeToFix
	for _, edge := range edges {
		switch edge.FixAction {
		case FixActionSwap:
			swapEdges = append(swapEdges, edge)
		case FixActionDelete:
			deleteEdges = append(deleteEdges, edge)
		}
	}

	// 处理需要交换的边
	if len(swapEdges) > 0 {
		log.Printf("开始处理 %d 条需要交换的边...", len(swapEdges))
		for i := 0; i < len(swapEdges); i += batchSize {
			end := i + batchSize
			if end > len(swapEdges) {
				end = len(swapEdges)
			}
			batch := swapEdges[i:end]
			log.Printf("正在交换第 %d - %d 条边...", i+1, end)
			if err := c.swapBatch(batch); err != nil {
				return fmt.Errorf("交换批次 %d-%d 时失败: %w", i+1, end, err)
			}
		}
		log.Println("所有需要交换的边已成功处理！")
	}

	// 处理需要删除的边
	if len(deleteEdges) > 0 {
		log.Printf("开始处理 %d 条需要删除的边...", len(deleteEdges))
		for i := 0; i < len(deleteEdges); i += batchSize {
			end := i + batchSize
			if end > len(deleteEdges) {
				end = len(deleteEdges)
			}
			batch := deleteEdges[i:end]
			log.Printf("正在删除第 %d - %d 条边...", i+1, end)
			if err := c.deleteBatch(batch); err != nil {
				return fmt.Errorf("删除批次 %d-%d 时失败: %w", i+1, end, err)
			}
		}
		log.Println("所有需要删除的边已成功处理！")
	}

	return nil
}

// swapBatch 在一个事务中交换一批边
func (c *DgraphCleaner) swapBatch(batch []EdgeToFix) error {
	txn := c.dg.NewTxn()
	defer txn.Discard(context.Background())

	mu := &api.Mutation{}
	for _, edge := range batch {
		// 1. 删除错误的边
		incorrectNQuad := &api.NQuad{
			Subject:   edge.SubjectUID,
			Predicate: edge.Predicate,
			ObjectId:  edge.ObjectUID,
		}
		mu.Del = append(mu.Del, incorrectNQuad)

		// 2. 创建正确的边（交换主体和客体）
		correctNQuad := &api.NQuad{
			Subject:   edge.ObjectUID,
			Predicate: edge.Predicate,
			ObjectId:  edge.SubjectUID,
		}
		mu.Set = append(mu.Set, correctNQuad)

		log.Printf("  - 交换: %s (%s) -> %s -> %s (%s) => %s (%s) -> %s -> %s (%s)",
			edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType,
			edge.ObjectUID, edge.ObjectType, edge.Predicate, edge.SubjectUID, edge.SubjectType)
	}

	_, err := txn.Mutate(context.Background(), mu)
	if err != nil {
		return fmt.Errorf("事务执行失败: %w", err)
	}

	return txn.Commit(context.Background())
}

// deleteBatch 在一个事务中删除一批边
func (c *DgraphCleaner) deleteBatch(batch []EdgeToFix) error {
	txn := c.dg.NewTxn()
	defer txn.Discard(context.Background())

	mu := &api.Mutation{}
	for _, edge := range batch {
		// 删除错误的边
		incorrectNQuad := &api.NQuad{
			Subject:   edge.SubjectUID,
			Predicate: edge.Predicate,
			ObjectId:  edge.ObjectUID,
		}
		mu.Del = append(mu.Del, incorrectNQuad)

		log.Printf("  - 删除: %s (%s) -> %s -> %s (%s)",
			edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType)
	}

	_, err := txn.Mutate(context.Background(), mu)
	if err != nil {
		return fmt.Errorf("事务执行失败: %w", err)
	}

	return txn.Commit(context.Background())
}

// ============================================
// 节点合并功能
// ============================================

// getNodeInfo 获取节点详细信息（包括入边和出边）
func (c *DgraphCleaner) getNodeInfo(uid string) (*NodeInfo, error) {
	// 查询节点的详细信息，包括所有出边
	// 使用两步查询：先获取基本信息，再获取所有边
	query := fmt.Sprintf(`
	{
		q(func: uid(%s)) {
			uid
			name
			dgraph.type
			aliases
		}
	}`, uid)

	txn := c.dg.NewReadOnlyTxn()
	defer txn.Discard(context.Background())

	resp, err := txn.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("查询节点 %s 失败: %w", uid, err)
	}

	var result struct {
		Q []struct {
			UID        string   `json:"uid"`
			Name       string   `json:"name"`
			DgraphType []string `json:"dgraph.type"`
			Aliases    []string `json:"aliases"`
		} `json:"q"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("解析节点 %s 数据失败: %w", uid, err)
	}

	if len(result.Q) == 0 {
		return nil, fmt.Errorf("节点 %s 不存在", uid)
	}

	nodeData := result.Q[0]
	nodeInfo := &NodeInfo{
		UID:     nodeData.UID,
		Name:    nodeData.Name,
		Aliases: nodeData.Aliases,
	}

	if len(nodeData.DgraphType) > 0 {
		nodeInfo.Type = nodeData.DgraphType[0]
	}

	// 获取 schema 中的所有谓词
	predicates, err := c.getAllPredicates()
	if err != nil {
		log.Printf("警告: 获取谓词列表失败: %v", err)
	} else {
		// 为每个谓词查询出边
		for _, predicate := range predicates {
			edges := c.getOutEdgesForPredicate(uid, predicate)
			nodeInfo.OutEdges = append(nodeInfo.OutEdges, edges...)
		}
	}

	// 查询入边
	inEdges, err := c.getInEdges(uid)
	if err != nil {
		log.Printf("警告: 查询节点 %s 入边失败: %v", uid, err)
	} else {
		nodeInfo.InEdges = inEdges
	}

	return nodeInfo, nil
}

// getAllPredicates 获取 schema 中的所有谓词
func (c *DgraphCleaner) getAllPredicates() ([]string, error) {
	schemaQuery := `schema { predicate }`
	txn := c.dg.NewReadOnlyTxn()
	defer txn.Discard(context.Background())

	resp, err := txn.Query(context.Background(), schemaQuery)
	if err != nil {
		return nil, fmt.Errorf("获取 schema 失败: %w", err)
	}

	var schemaResult struct {
		Schema []struct {
			Predicate string `json:"predicate"`
		} `json:"schema"`
	}

	if err := json.Unmarshal(resp.Json, &schemaResult); err != nil {
		return nil, fmt.Errorf("解析 schema 失败: %w", err)
	}

	// 收集所有谓词
	var predicates []string
	for _, s := range schemaResult.Schema {
		pred := s.Predicate
		// 跳过内置谓词
		if pred == "dgraph.type" || pred == "dgraph.graph.xid" {
			continue
		}
		// 包含所有非内置谓词
		predicates = append(predicates, pred)
	}

	log.Printf("  - 找到 %d 个谓词", len(predicates))
	return predicates, nil
}

// getOutEdgesForPredicate 获取节点在指定谓词上的出边
func (c *DgraphCleaner) getOutEdgesForPredicate(uid, predicate string) []EdgeInfo {
	var edges []EdgeInfo

	// 确保谓词格式正确（用尖括号包裹）
	predFormatted := predicate
	if !strings.HasPrefix(predicate, "<") {
		predFormatted = "<" + predicate + ">"
	}

	query := fmt.Sprintf(`
	{
		q(func: uid(%s)) {
			%s {
				uid
			}
		}
	}`, uid, predFormatted)

	txn := c.dg.NewReadOnlyTxn()
	defer txn.Discard(context.Background())

	resp, err := txn.Query(context.Background(), query)
	if err != nil {
		return edges
	}

	// 使用动态解析
	var rawResult map[string]interface{}
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return edges
	}

	if qList, ok := rawResult["q"].([]interface{}); ok && len(qList) > 0 {
		if nodeMap, ok := qList[0].(map[string]interface{}); ok {
			// 尝试两种格式的键：带尖括号和不带尖括号
			var targets []interface{}
			var found bool
			if t, ok := nodeMap[predFormatted].([]interface{}); ok {
				targets = t
				found = true
			} else if t, ok := nodeMap[predicate].([]interface{}); ok {
				targets = t
				found = true
			}

			if found {
				for _, target := range targets {
					if targetMap, ok := target.(map[string]interface{}); ok {
						targetUID, _ := targetMap["uid"].(string)
						if targetUID != "" {
							edges = append(edges, EdgeInfo{
								Predicate: predFormatted,
								TargetUID: targetUID,
								SourceUID: uid,
							})
						}
					}
				}
			}
		}
	}

	if len(edges) > 0 {
		log.Printf("    - 谓词 %s: 找到 %d 条出边", predicate, len(edges))
	}

	return edges
}

// getInEdges 获取指向该节点的所有入边
func (c *DgraphCleaner) getInEdges(uid string) ([]EdgeInfo, error) {
	var inEdges []EdgeInfo

	// 首先获取 schema 中的所有谓词
	schemaQuery := `schema { predicate }`
	txn := c.dg.NewReadOnlyTxn()
	resp, err := txn.Query(context.Background(), schemaQuery)
	txn.Discard(context.Background())
	if err != nil {
		return nil, fmt.Errorf("获取 schema 失败: %w", err)
	}

	var schemaResult struct {
		Schema []struct {
			Predicate string `json:"predicate"`
		} `json:"schema"`
	}

	if err := json.Unmarshal(resp.Json, &schemaResult); err != nil {
		return nil, fmt.Errorf("解析 schema 失败: %w", err)
	}

	// 收集所有谓词
	var predicates []string
	for _, s := range schemaResult.Schema {
		pred := s.Predicate
		// 跳过内置谓词
		if pred == "dgraph.type" || pred == "dgraph.graph.xid" {
			continue
		}
		predicates = append(predicates, pred)
	}

	// 为每个谓词查询入边
	for _, predicate := range predicates {
		// 确保谓词格式正确（用尖括号包裹）
		predFormatted := predicate
		if !strings.HasPrefix(predicate, "<") {
			predFormatted = "<" + predicate + ">"
		}

		query := fmt.Sprintf(`
		{
			sources(func: has(%s)) {
				uid
				name
				%s @filter(eq(uid, %s)) {
					uid
				}
			}
		}`, predFormatted, predFormatted, uid)

		txn2 := c.dg.NewReadOnlyTxn()
		resp2, err := txn2.Query(context.Background(), query)
		txn2.Discard(context.Background())
		if err != nil {
			continue // 忽略查询错误，继续检查其他谓词
		}

		// 使用动态解析
		var rawResult map[string]interface{}
		if err := json.Unmarshal(resp2.Json, &rawResult); err != nil {
			continue
		}

		if sources, ok := rawResult["sources"].([]interface{}); ok {
			for _, source := range sources {
				if sourceMap, ok := source.(map[string]interface{}); ok {
					sourceUID, _ := sourceMap["uid"].(string)
					if sourceUID == "" || sourceUID == uid {
						continue
					}

					// 检查该源节点是否有指向目标节点的边
					// 尝试两种格式的键：带尖括号和不带尖括号
					var targets []interface{}
					var found bool
					if t, ok := sourceMap[predFormatted].([]interface{}); ok {
						targets = t
						found = true
					} else if t, ok := sourceMap[predicate].([]interface{}); ok {
						targets = t
						found = true
					}

					if found {
						for _, target := range targets {
							if targetMap, ok := target.(map[string]interface{}); ok {
								targetUID, _ := targetMap["uid"].(string)
								if targetUID == uid {
									inEdges = append(inEdges, EdgeInfo{
										Predicate: predFormatted,
										SourceUID: sourceUID,
										TargetUID: uid,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	if len(inEdges) > 0 {
		log.Printf("  - 找到 %d 条入边", len(inEdges))
	}

	return inEdges, nil
}

// MergeNodes 将源节点合并到目标节点
func (c *DgraphCleaner) MergeNodes(sourceUID, targetUID string) (*MergeReport, error) {
	report := &MergeReport{
		SourceUID: sourceUID,
		TargetUID: targetUID,
		Success:   false,
	}

	// 验证参数
	if sourceUID == "" || targetUID == "" {
		report.Error = "源节点和目标节点 UID 不能为空"
		return report, fmt.Errorf(report.Error)
	}

	if sourceUID == targetUID {
		report.Error = "源节点和目标节点不能相同"
		return report, fmt.Errorf(report.Error)
	}

	// 获取源节点信息
	sourceInfo, err := c.getNodeInfo(sourceUID)
	if err != nil {
		report.Error = fmt.Sprintf("获取源节点信息失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}
	report.SourceName = sourceInfo.Name

	// 获取目标节点信息
	targetInfo, err := c.getNodeInfo(targetUID)
	if err != nil {
		report.Error = fmt.Sprintf("获取目标节点信息失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}
	report.TargetName = targetInfo.Name

	// 打印合并预览
	log.Println("========== 合并预览 ==========")
	log.Printf("源节点: %s (%s) [%s]", sourceInfo.Name, sourceInfo.UID, sourceInfo.Type)
	log.Printf("  - 别名数量: %d", len(sourceInfo.Aliases))
	log.Printf("  - 出边数量: %d", len(sourceInfo.OutEdges))
	log.Printf("  - 入边数量: %d", len(sourceInfo.InEdges))
	log.Printf("目标节点: %s (%s) [%s]", targetInfo.Name, targetInfo.UID, targetInfo.Type)
	log.Printf("  - 别名数量: %d", len(targetInfo.Aliases))
	log.Printf("  - 出边数量: %d", len(targetInfo.OutEdges))
	log.Printf("  - 入边数量: %d", len(targetInfo.InEdges))
	log.Println("==============================")

	// 演练模式：只打印预览，不执行
	if c.dryRun {
		log.Println("--- 演练模式：以下操作将被执行 ---")
		log.Printf("1. 迁移 %d 个别名到目标节点", len(sourceInfo.Aliases))
		log.Printf("2. 迁移 %d 条出边到目标节点", len(sourceInfo.OutEdges))
		log.Printf("3. 迁移 %d 条入边到目标节点", len(sourceInfo.InEdges))
		log.Printf("4. 删除源节点 %s", sourceUID)
		report.Success = true
		return report, nil
	}

	// 执行合并操作
	txn := c.dg.NewTxn()
	defer txn.Discard(context.Background())

	// 1. 迁移别名
	aliasesMigrated, err := c.migrateAliasesInTxn(txn, sourceInfo, targetInfo)
	if err != nil {
		report.Error = fmt.Sprintf("迁移别名失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}
	report.AliasesMigrated = aliasesMigrated

	// 2. 迁移出边
	outEdgesMigrated, err := c.migrateOutEdgesInTxn(txn, sourceInfo, targetInfo)
	if err != nil {
		report.Error = fmt.Sprintf("迁移出边失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}
	report.OutEdgesMigrated = outEdgesMigrated

	// 3. 迁移入边
	inEdgesMigrated, err := c.migrateInEdgesInTxn(txn, sourceInfo, targetInfo)
	if err != nil {
		report.Error = fmt.Sprintf("迁移入边失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}
	report.InEdgesMigrated = inEdgesMigrated

	// 4. 删除源节点
	if err := c.deleteNodeInTxn(txn, sourceUID); err != nil {
		report.Error = fmt.Sprintf("删除源节点失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}

	// 提交事务
	if err := txn.Commit(context.Background()); err != nil {
		report.Error = fmt.Sprintf("提交事务失败: %v", err)
		return report, fmt.Errorf(report.Error)
	}

	report.Success = true
	log.Println("========== 合并完成 ==========")
	log.Printf("迁移别名: %d", report.AliasesMigrated)
	log.Printf("迁移出边: %d", report.OutEdgesMigrated)
	log.Printf("迁移入边: %d", report.InEdgesMigrated)
	log.Println("==============================")

	return report, nil
}

// migrateAliasesInTxn 在事务中迁移别名
func (c *DgraphCleaner) migrateAliasesInTxn(txn *dgo.Txn, source, target *NodeInfo) (int, error) {
	if len(source.Aliases) == 0 {
		return 0, nil
	}

	// 合并别名，去重
	aliasSet := make(map[string]bool)
	for _, a := range target.Aliases {
		aliasSet[a] = true
	}
	for _, a := range source.Aliases {
		aliasSet[a] = true
	}

	// 构建更新数据
	mergedAliases := make([]string, 0, len(aliasSet))
	for a := range aliasSet {
		mergedAliases = append(mergedAliases, a)
	}

	data := map[string]interface{}{
		"uid":     target.UID,
		"aliases": mergedAliases,
	}

	setJSON, err := json.Marshal(data)
	if err != nil {
		return 0, fmt.Errorf("序列化别名数据失败: %w", err)
	}

	mu := &api.Mutation{
		SetJson:   setJSON,
		CommitNow: false,
	}

	_, err = txn.Mutate(context.Background(), mu)
	if err != nil {
		return 0, fmt.Errorf("更新别名失败: %w", err)
	}

	log.Printf("  - 迁移别名: %v", source.Aliases)
	return len(source.Aliases), nil
}

// migrateOutEdgesInTxn 在事务中迁移出边
func (c *DgraphCleaner) migrateOutEdgesInTxn(txn *dgo.Txn, source, target *NodeInfo) (int, error) {
	if len(source.OutEdges) == 0 {
		return 0, nil
	}

	migratedCount := 0

	for _, edge := range source.OutEdges {
		// 检查目标节点是否已有相同的边
		hasSameEdge := false
		for _, targetEdge := range target.OutEdges {
			if targetEdge.Predicate == edge.Predicate && targetEdge.TargetUID == edge.TargetUID {
				hasSameEdge = true
				break
			}
		}

		// 去掉谓词中的尖括号（JSON 格式不需要尖括号）
		predName := strings.Trim(edge.Predicate, "<>")

		if !hasSameEdge {
			// 使用 JSON 格式添加新边
			addData := map[string]interface{}{
				"uid": target.UID,
				predName: []map[string]string{
					{"uid": edge.TargetUID},
				},
			}
			setJSON, err := json.Marshal(addData)
			if err != nil {
				return 0, fmt.Errorf("序列化边数据失败: %w", err)
			}

			mu := &api.Mutation{
				SetJson:   setJSON,
				CommitNow: false,
			}

			_, err = txn.Mutate(context.Background(), mu)
			if err != nil {
				return 0, fmt.Errorf("添加新边失败: %w", err)
			}

			migratedCount++
			log.Printf("  - 迁移出边: %s -> %s -> %s", target.UID, edge.Predicate, edge.TargetUID)
		}

		// 使用 NQuad 格式删除源节点的特定边（不带尖括号）
		oldEdge := &api.NQuad{
			Subject:   source.UID,
			Predicate: predName,
			ObjectId:  edge.TargetUID,
		}
		muDel := &api.Mutation{
			Del:       []*api.NQuad{oldEdge},
			CommitNow: false,
		}

		_, err := txn.Mutate(context.Background(), muDel)
		if err != nil {
			return 0, fmt.Errorf("删除源边失败: %w", err)
		}
	}

	return migratedCount, nil
}

// migrateInEdgesInTxn 在事务中迁移入边
func (c *DgraphCleaner) migrateInEdgesInTxn(txn *dgo.Txn, source, target *NodeInfo) (int, error) {
	if len(source.InEdges) == 0 {
		return 0, nil
	}

	migratedCount := 0

	for _, edge := range source.InEdges {
		// 检查目标节点是否已有相同的入边
		hasSameEdge := false
		for _, targetEdge := range target.InEdges {
			if targetEdge.Predicate == edge.Predicate && targetEdge.SourceUID == edge.SourceUID {
				hasSameEdge = true
				break
			}
		}

		// 去掉谓词中的尖括号（JSON 格式不需要尖括号）
		predName := strings.Trim(edge.Predicate, "<>")

		if !hasSameEdge {
			// 使用 JSON 格式添加新边（从源节点指向目标节点）
			addData := map[string]interface{}{
				"uid": edge.SourceUID,
				predName: []map[string]string{
					{"uid": target.UID},
				},
			}
			setJSON, err := json.Marshal(addData)
			if err != nil {
				return 0, fmt.Errorf("序列化边数据失败: %w", err)
			}

			mu := &api.Mutation{
				SetJson:   setJSON,
				CommitNow: false,
			}

			_, err = txn.Mutate(context.Background(), mu)
			if err != nil {
				return 0, fmt.Errorf("添加新入边失败: %w", err)
			}

			migratedCount++
			log.Printf("  - 迁移入边: %s -> %s -> %s", edge.SourceUID, edge.Predicate, target.UID)
		}

		// 使用 NQuad 格式删除指向源节点的特定边（不带尖括号）
		oldEdge := &api.NQuad{
			Subject:   edge.SourceUID,
			Predicate: predName,
			ObjectId:  source.UID,
		}
		muDel := &api.Mutation{
			Del:       []*api.NQuad{oldEdge},
			CommitNow: false,
		}

		_, err := txn.Mutate(context.Background(), muDel)
		if err != nil {
			return 0, fmt.Errorf("删除源入边失败: %w", err)
		}
	}

	return migratedCount, nil
}

// deleteNodeInTxn 在事务中删除节点
func (c *DgraphCleaner) deleteNodeInTxn(txn *dgo.Txn, uid string) error {
	// 使用 JSON 格式删除节点
	// Dgraph 会删除该节点的所有属性和出边
	deleteJSON := map[string]string{
		"uid": uid,
	}

	deleteData, err := json.Marshal(deleteJSON)
	if err != nil {
		return fmt.Errorf("序列化删除数据失败: %w", err)
	}

	mu := &api.Mutation{
		DeleteJson: deleteData,
	}

	_, err = txn.Mutate(context.Background(), mu)
	if err != nil {
		return fmt.Errorf("删除节点失败: %w", err)
	}

	log.Printf("  - 删除节点: %s", uid)
	return nil
}

// MergeMultipleNodes 将多个源节点合并到一个目标节点
func (c *DgraphCleaner) MergeMultipleNodes(sourceUIDs []string, targetUID string) ([]*MergeReport, error) {
	if len(sourceUIDs) == 0 {
		return nil, fmt.Errorf("源节点列表不能为空")
	}

	if targetUID == "" {
		return nil, fmt.Errorf("目标节点 UID 不能为空")
	}

	// 检查目标节点是否在源节点列表中
	for _, sourceUID := range sourceUIDs {
		if sourceUID == targetUID {
			return nil, fmt.Errorf("目标节点 %s 不能在源节点列表中", targetUID)
		}
	}

	reports := make([]*MergeReport, 0, len(sourceUIDs))

	// 获取目标节点信息（用于累加）
	targetInfo, err := c.getNodeInfo(targetUID)
	if err != nil {
		return nil, fmt.Errorf("获取目标节点信息失败: %w", err)
	}

	log.Println("========== 批量合并预览 ==========")
	log.Printf("目标节点: %s (%s) [%s]", targetInfo.Name, targetInfo.UID, targetInfo.Type)
	log.Printf("源节点数量: %d", len(sourceUIDs))
	log.Println("==================================")

	// 逐个处理源节点
	for i, sourceUID := range sourceUIDs {
		log.Printf("\n--- 处理第 %d/%d 个源节点 ---", i+1, len(sourceUIDs))

		report, err := c.MergeNodes(sourceUID, targetUID)
		if err != nil {
			log.Printf("合并源节点 %s 失败: %v", sourceUID, err)
			report = &MergeReport{
				SourceUID: sourceUID,
				TargetUID: targetUID,
				Success:   false,
				Error:     err.Error(),
			}
		}
		reports = append(reports, report)

		// 如果不是演练模式且合并成功，需要更新目标节点信息以便下一次合并
		if !c.dryRun && report.Success {
			// 重新获取目标节点信息（包含已迁移的数据）
			targetInfo, err = c.getNodeInfo(targetUID)
			if err != nil {
				log.Printf("警告: 更新目标节点信息失败: %v", err)
			}
		}
	}

	// 打印汇总报告
	c.printBatchMergeReport(reports)

	return reports, nil
}

// printBatchMergeReport 打印批量合并报告
func (c *DgraphCleaner) printBatchMergeReport(reports []*MergeReport) {
	log.Println("\n========== 批量合并报告 ==========")

	successCount := 0
	failCount := 0
	totalAliases := 0
	totalOutEdges := 0
	totalInEdges := 0

	for _, report := range reports {
		if report.Success {
			successCount++
			totalAliases += report.AliasesMigrated
			totalOutEdges += report.OutEdgesMigrated
			totalInEdges += report.InEdgesMigrated
			log.Printf("✓ %s -> %s: 别名=%d, 出边=%d, 入边=%d",
				report.SourceName, report.TargetName,
				report.AliasesMigrated, report.OutEdgesMigrated, report.InEdgesMigrated)
		} else {
			failCount++
			log.Printf("✗ %s: 失败 - %s", report.SourceUID, report.Error)
		}
	}

	log.Println("----------------------------------")
	log.Printf("成功: %d, 失败: %d", successCount, failCount)
	log.Printf("总计: 别名=%d, 出边=%d, 入边=%d", totalAliases, totalOutEdges, totalInEdges)
	log.Println("==================================")
}

func main() {
	// 从命令行参数获取 Dgraph 地址
	dgraphAddr := flag.String("addr", "10.10.0.1:9080", "Dgraph server address (e.g., localhost:9080)")
	dryRun := flag.Bool("dry-run", true, "If true, only prints the operations without making changes.")

	// 合并功能参数
	mergeMode := flag.Bool("merge", false, "Enable merge mode")
	sourceUID := flag.String("source", "", "Source node UID to merge (will be deleted). For multiple sources, use comma-separated UIDs.")
	sourcesUID := flag.String("sources", "", "Multiple source node UIDs to merge (comma-separated, e.g., 0x1,0x2,0x3)")
	targetUID := flag.String("target", "", "Target node UID to merge into (will be kept)")

	flag.Parse()

	log.Printf("正在连接到 Dgraph: %s (演练模式: %v)", *dgraphAddr, *dryRun)

	cleaner, err := NewDgraphCleaner(*dgraphAddr, *dryRun)
	if err != nil {
		log.Fatalf("创建清洗器失败: %v", err)
	}

	// 根据模式选择执行的功能
	if *mergeMode {
		// 合并模式
		if *targetUID == "" {
			log.Fatal("合并模式需要指定 -target 参数")
		}

		// 确定源节点列表
		var sourceUIDs []string
		if *sourcesUID != "" {
			// 使用多源节点参数
			sourceUIDs = strings.Split(*sourcesUID, ",")
			for i, uid := range sourceUIDs {
				sourceUIDs[i] = strings.TrimSpace(uid)
			}
		} else if *sourceUID != "" {
			// 使用单源节点参数
			sourceUIDs = []string{*sourceUID}
		} else {
			log.Fatal("合并模式需要指定 -source 或 -sources 参数")
		}

		if len(sourceUIDs) == 1 {
			// 单个源节点合并
			log.Printf("开始合并节点: %s -> %s", sourceUIDs[0], *targetUID)
			report, err := cleaner.MergeNodes(sourceUIDs[0], *targetUID)
			if err != nil {
				log.Fatalf("合并失败: %v", err)
			}
			if report.Success {
				log.Println("节点合并成功！")
			}
		} else {
			// 多个源节点合并
			log.Printf("开始批量合并 %d 个节点到 %s", len(sourceUIDs), *targetUID)
			reports, err := cleaner.MergeMultipleNodes(sourceUIDs, *targetUID)
			if err != nil {
				log.Fatalf("批量合并失败: %v", err)
			}
			// 检查是否所有合并都成功
			allSuccess := true
			for _, report := range reports {
				if !report.Success {
					allSuccess = false
					break
				}
			}
			if allSuccess {
				log.Println("所有节点合并成功！")
			} else {
				log.Println("部分节点合并失败，请查看报告")
			}
		}
	} else {
		// 清洗模式
		if err := cleaner.Run(); err != nil {
			log.Fatalf("清洗过程失败: %v", err)
		}
	}
}
