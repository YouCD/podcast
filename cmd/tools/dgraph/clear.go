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
var ValidationRules = map[string]RelationRule{
	"<研发>": {
		AllowedSourceTypes: []string{"企业", "人物"},
		AllowedTargetTypes: []string{"技术", "AI模型", "计算设备"},
	},
	"<投资>": {
		AllowedSourceTypes: []string{"企业", "投资机构"},
		AllowedTargetTypes: []string{"企业"},
	},
	"<竞争>": {
		AllowedSourceTypes: []string{"企业", "投资机构"},
		AllowedTargetTypes: []string{"企业", "投资机构"},
	},
	"<基于>": {
		AllowedSourceTypes: []string{"AI模型", "技术"},
		AllowedTargetTypes: []string{"技术", "AI模型", "计算设备", "企业", "标准"},
	},
	"<所属企业>": {
		AllowedSourceTypes: []string{"人物", "AI模型", "技术", "计算设备"},
		AllowedTargetTypes: []string{"企业"},
	},
}

// EdgeToFix 代表一个需要修复的边
type EdgeToFix struct {
	Predicate   string
	SubjectUID  string
	ObjectUID   string
	SubjectType string
	ObjectType  string
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
	edgesToFix, err := c.findIncorrectEdges()
	if err != nil {
		return fmt.Errorf("查找错误边时失败: %w", err)
	}

	if len(edgesToFix) == 0 {
		log.Println("恭喜！没有发现需要修复的边。")
		return nil
	}

	log.Printf("共发现 %d 条错误的边。\n", len(edgesToFix))
	if c.dryRun {
		log.Println("--- 演练模式：以下边将被修复，但不会实际执行 ---")
		for _, edge := range edgesToFix {
			log.Printf(" - 修复 '%s' (%s) -> '%s' -> '%s' (%s)", edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType)
		}
		return nil
	}

	return c.fixEdgesInBatches(edgesToFix)
}

// findIncorrectEdges 查询并识别所有错误的边
func (c *DgraphCleaner) findIncorrectEdges() ([]EdgeToFix, error) {
	var edgesToFix []EdgeToFix

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
			return nil, fmt.Errorf("查询谓词 '%s' 失败: %w", predicate, err)
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
			return nil, err
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

								if c.isEdgeIncorrect(predicate, subjectType, objectType) {
									edgesToFix = append(edgesToFix, EdgeToFix{
										Predicate:   predicate,
										SubjectUID:  subjectUID,
										ObjectUID:   objectUID,
										SubjectType: subjectType,
										ObjectType:  objectType,
									})
								}
							}
						}
					}
				}
			}
		}
	}

	return edgesToFix, nil
}

// isEdgeIncorrect 根据规则检查边是否错误
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
	for i := 0; i < len(edges); i += batchSize {
		end := i + batchSize
		if end > len(edges) {
			end = len(edges)
		}
		batch := edges[i:end]
		log.Printf("正在修复第 %d - %d 条边...", i+1, end)
		if err := c.fixBatch(batch); err != nil {
			return fmt.Errorf("修复批次 %d-%d 时失败: %w", i+1, end, err)
		}
	}
	log.Println("所有错误的边已成功修复！")
	return nil
}

// fixBatch 在一个事务中修复一批边
func (c *DgraphCleaner) fixBatch(batch []EdgeToFix) error {
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

		// 2. 创建正确的边
		correctNQuad := &api.NQuad{
			Subject:   edge.ObjectUID, // 主体和客体互换
			Predicate: edge.Predicate,
			ObjectId:  edge.SubjectUID,
		}
		mu.Set = append(mu.Set, correctNQuad)

		log.Printf("  - 修复: %s (%s) -> %s -> %s (%s)", edge.SubjectUID, edge.SubjectType, edge.Predicate, edge.ObjectUID, edge.ObjectType)
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
