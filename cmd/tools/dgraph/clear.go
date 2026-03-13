// main.go - Dgraph知识图谱关系方向清洗工具
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"

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

func main() {
	// 从命令行参数获取 Dgraph 地址
	dgraphAddr := flag.String("addr", "10.10.0.1:9080", "Dgraph server address (e.g., localhost:9080)")
	dryRun := flag.Bool("dry-run", true, "If true, only prints the edges to be fixed without making changes.")
	flag.Parse()

	log.Printf("正在连接到 Dgraph: %s (演练模式: %v)", *dgraphAddr, *dryRun)

	cleaner, err := NewDgraphCleaner(*dgraphAddr, *dryRun)
	if err != nil {
		log.Fatalf("创建清洗器失败: %v", err)
	}

	if err := cleaner.Run(); err != nil {
		log.Fatalf("清洗过程失败: %v", err)
	}
}
