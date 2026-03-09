package types

import (
	"encoding/json"
	"testing"
)

func TestNodeUnmarshalJSON_DataLoss(t *testing.T) {
	// 测试用例1：正常数据
	testCases := []struct {
		name     string
		jsonData string
		expected int // 期望的谓词数量
	}{
		{
			name: "正常投资关系数据",
			jsonData: `{
				"uid": "_:company1",
				"name": "测试公司",
				"dgraph.type": "Company",
				"aliases": ["测试"],
				"投资": [
					{
						"投资_uid": "_:target1",
						"投资_置信度": 0.95,
						"投资_证据": "投资证据1",
						"投资_时间": "2024-01-01"
					},
					{
						"投资_uid": "_:target2",
						"投资_置信度": 0.85,
						"投资_证据": "投资证据2",
						"投资_时间": "2024-01-02"
					}
				]
			}`,
			expected: 1,
		},
		{
			name: "多个谓词关系",
			jsonData: `{
				"uid": "_:entity1",
				"name": "实体1",
				"dgraph.type": "Entity",
				"投资": [
					{
						"投资_uid": "_:invest1",
						"投资_置信度": 0.9,
						"投资_证据": "投资证据",
						"投资_时间": "2024-01-01"
					}
				],
				"合作": [
					{
						"合作_uid": "_:partner1",
						"合作_置信度": 0.8,
						"合作_证据": "合作证据",
						"合作_时间": "2024-01-02"
					}
				]
			}`,
			expected: 2,
		},
		{
			name: "空谓词数据",
			jsonData: `{
				"uid": "_:empty1",
				"name": "空实体",
				"dgraph.type": "Entity",
				"投资": []
			}`,
			expected: 0, // 空数组应该被跳过
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var node Node
			err := json.Unmarshal([]byte(tc.jsonData), &node)
			if err != nil {
				t.Fatalf("解析失败: %v", err)
			}

			// 检查谓词数量
			actualPredicates := len(node.Predicates)
			if actualPredicates != tc.expected {
				t.Errorf("谓词数量不匹配: 期望 %d, 实际 %d", tc.expected, actualPredicates)
				t.Logf("实际谓词: %v", node.Predicates)
			}

			// 验证数据完整性
			for predicateName, edges := range node.Predicates {
				if len(edges) == 0 {
					t.Errorf("谓词 '%s' 的边数组为空", predicateName)
				}
				for i, edge := range edges {
					if edge.Uid == "" {
						t.Errorf("谓词 '%s' 的第 %d 个边 UID 为空", predicateName, i+1)
					}
					if edge.Confidence <= 0 {
						t.Errorf("谓词 '%s' 的第 %d 个边置信度无效: %f", predicateName, i+1, edge.Confidence)
					}
				}
			}
		})
	}
}

func TestNodeUnmarshalJSON_EdgeParsing(t *testing.T) {
	// 测试边解析的详细情况
	jsonData := `{
		"uid": "_:test1",
		"name": "测试实体",
		"dgraph.type": "Company",
		"研发": [
			{
				"研发_uid": "_:product1",
				"研发_置信度": 0.92,
				"研发_证据": "该公司正在研发新产品",
				"研发_时间": "2024-03-15"
			}
		]
	}`

	var node Node
	err := json.Unmarshal([]byte(jsonData), &node)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 检查特定谓词
	edges, exists := node.Predicates["研发"]
	if !exists {
		t.Fatal("未找到'研发'谓词")
	}

	if len(edges) != 1 {
		t.Fatalf("期望1个边，实际%d个", len(edges))
	}

	edge := edges[0]
	if edge.Uid != "_:product1" {
		t.Errorf("UID不匹配: 期望 '_:product1', 实际 '%s'", edge.Uid)
	}
	if edge.Confidence != 0.92 {
		t.Errorf("置信度不匹配: 期望 0.92, 实际 %f", edge.Confidence)
	}
	if edge.Evidence != "该公司正在研发新产品" {
		t.Errorf("证据不匹配: 期望 '该公司正在研发新产品', 实际 '%s'", edge.Evidence)
	}
	if edge.Time != "2024-03-15" {
		t.Errorf("时间不匹配: 期望 '2024-03-15', 实际 '%s'", edge.Time)
	}
}
