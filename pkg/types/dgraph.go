package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// DgraphPayload 是与 JSON 顶层结构对应的 Go 结构体
type DgraphPayload struct {
	Set []Node `json:"set"`
}

// Node 代表 Dgraph 中的一个节点
// 我们为其实现自定义的 UnmarshalJSON 来处理动态的谓词键
type Node struct {
	Uid        string   `json:"uid"`
	Name       string   `json:"name"`
	DgraphType string   `json:"dgraph.type"`
	Aliases    []string `json:"aliases"`
	// Predicates 用于存储所有动态的谓词，如 "投资", "研发", "隶属"
	// 键是谓词名，值是对应的边信息切片
	Predicates map[string][]Edge `json:"predicates,omitempty"`
}

// Edge 代表一个带属性的边（谓词）
// 我们为其实现自定义的 UnmarshalJSON 来处理动态的属性键
type Edge struct {
	Uid        string  `json:"uid"`
	Confidence float64 `json:"confidence"`
	Evidence   string  `json:"evidence"`
	Time       string  `json:"time"`
}

// UnmarshalJSON 为 Node 实现了自定义的 JSON 解析逻辑
// 这使得我们可以将未知的顶级键（如 "投资"）解析到 Predicates map 中
func (n *Node) UnmarshalJSON(data []byte) error {
	// 1. 将所有 JSON 字段解析到一个 map 中
	var allFields map[string]json.RawMessage
	if err := json.Unmarshal(data, &allFields); err != nil {
		return err
	}

	// 2. 创建一个辅助结构体，只包含我们已知的静态字段
	aux := &struct {
		Uid        string                     `json:"uid"`
		Name       string                     `json:"name"`
		DgraphType string                     `json:"dgraph.type"`
		Aliases    []string                   `json:"aliases,omitempty"`
		Rest       map[string]json.RawMessage `json:"-"`
	}{
		Rest: make(map[string]json.RawMessage),
	}

	// 3. 将已知的字段解析到辅助结构体中
	for key, value := range allFields {
		switch key {
		case "uid":
			if err := json.Unmarshal(value, &aux.Uid); err != nil {
				return err
			}
		case "name":
			if err := json.Unmarshal(value, &aux.Name); err != nil {
				return err
			}
		case "dgraph.type":
			if err := json.Unmarshal(value, &aux.DgraphType); err != nil {
				return err
			}
		case "aliases":
			if err := json.Unmarshal(value, &aux.Aliases); err != nil {
				return err
			}
		default:
			// 将所有未知字段存入 Rest map
			aux.Rest[key] = value
		}
	}

	// 4. 将辅助结构体的值赋给当前 Node
	n.Uid = aux.Uid
	n.Name = aux.Name
	n.DgraphType = aux.DgraphType
	n.Aliases = aux.Aliases

	// 确保 Predicates map 已初始化
	if n.Predicates == nil {
		n.Predicates = make(map[string][]Edge)
	} else {
		// 清空现有数据，避免重复添加
		for k := range n.Predicates {
			delete(n.Predicates, k)
		}
	}

	// 5. 解析剩余的动态字段（谓词）
	for predicateName, rawEdges := range aux.Rest {
		var edges []Edge
		if err := json.Unmarshal(rawEdges, &edges); err != nil {
			return fmt.Errorf("failed to unmarshal edges for predicate %s: %w", predicateName, err)
		}
		if len(edges) == 0 {
			continue // 跳过空谓词数组
		}
		n.Predicates[predicateName] = edges
	}

	return nil
}

// UnmarshalJSON 为 Edge 实现了自定义的 JSON 解析逻辑
// 支持多种字段命名格式
func (e *Edge) UnmarshalJSON(data []byte) error {
	// 将 JSON 对象解析到一个临时的 map 中
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// 遍历所有键值对，灵活匹配字段
	for key, value := range temp {
		// 转换为字符串进行比较
		strValue := fmt.Sprintf("%v", value)

		// 多种方式匹配UID字段
		if strings.HasSuffix(key, "_uid") || key == "uid" || key == "Uid" {
			e.Uid = strValue
			continue
		}

		// 多种方式匹配置信度字段
		if strings.HasSuffix(key, "_置信度") || key == "confidence" || key == "Confidence" {
			if f, ok := value.(float64); ok {
				e.Confidence = f
			} else if str, ok := value.(string); ok {
				if f, err := strconv.ParseFloat(str, 64); err == nil {
					e.Confidence = f
				}
			}
			continue
		}

		// 多种方式匹配证据字段
		if strings.HasSuffix(key, "_证据") || key == "evidence" || key == "Evidence" {
			e.Evidence = strValue
			continue
		}

		// 多种方式匹配时间字段
		if strings.HasSuffix(key, "_时间") || key == "time" || key == "Time" {
			e.Time = strValue
			continue
		}
	}
	return nil
}

func (d *DgraphPayload) GetNameByID(id string) string {
	for _, node := range d.Set {
		if node.Uid == id {
			return node.Name
		}
	}
	return ""
}

type DgraphNode struct {
	Uid        string   `json:"uid"`
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	DgraphType string   `json:"dgraph.type"`
	Aliases    []string `json:"aliases"`
	Category   int      `json:"category"`
	// Predicates 用于存储所有动态的谓词，如 "投资", "研发", "隶属"
	// 键是谓词名，值是对应的边信息切片
	Predicates map[string][]DgraphNode `json:"-"`
}

func (n *DgraphNode) UnmarshalJSON(data []byte) error {
	// 1. 将所有 JSON 字段解析到一个 map 中
	var allFields map[string]json.RawMessage
	if err := json.Unmarshal(data, &allFields); err != nil {
		return err
	}

	// 2. 创建一个辅助结构体，只包含我们已知的静态字段
	aux := &struct {
		Uid        string                     `json:"uid"`
		Name       string                     `json:"name"`
		DgraphType []string                   `json:"dgraph.type"`
		Aliases    []string                   `json:"aliases,omitempty"`
		Rest       map[string]json.RawMessage `json:"-"`
	}{
		Rest: make(map[string]json.RawMessage),
	}

	// 3. 将已知的字段解析到辅助结构体中
	for key, value := range allFields {
		switch key {
		case "uid":
			if err := json.Unmarshal(value, &aux.Uid); err != nil {
				return err
			}
		case "name":
			if err := json.Unmarshal(value, &aux.Name); err != nil {
				return err
			}
		case "dgraph.type":
			// dgraph 返回通常为数组，但也可能缺失、为 null 或单个字符串
			var single string
			if err := json.Unmarshal(value, &single); err == nil {
				aux.DgraphType = []string{single}
			} else if err := json.Unmarshal(value, &aux.DgraphType); err != nil {
				return err
			}
		case "aliases":
			if err := json.Unmarshal(value, &aux.Aliases); err != nil {
				return err
			}
		default:
			// 将所有未知字段存入 Rest map
			aux.Rest[key] = value
		}
	}

	// 4. 将辅助结构体的值赋给当前 Node
	n.Uid = aux.Uid
	n.Name = aux.Name
	if len(aux.DgraphType) > 0 {
		n.DgraphType = aux.DgraphType[0]
	}
	n.Aliases = aux.Aliases

	// 确保 Predicates map 已初始化
	if n.Predicates == nil {
		n.Predicates = make(map[string][]DgraphNode)
	} else {
		// 清空现有数据，避免重复添加
		for k := range n.Predicates {
			delete(n.Predicates, k)
		}
	}

	// 5. 解析剩余的动态字段（谓词）
	for predicateName, rawEdges := range aux.Rest {
		var edges []DgraphNode
		if err := json.Unmarshal(rawEdges, &edges); err != nil {
			return fmt.Errorf("failed to unmarshal edges for predicate %s: %w", predicateName, err)
		}
		if len(edges) == 0 {
			continue // 跳过空谓词数组
		}
		n.Predicates[predicateName] = edges
	}

	return nil
}

type DgraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Value  string `json:"value"`
}
type DgraphCategories struct {
	Name string `json:"name"`
}
type DgraphResp struct {
	Nodes []*DgraphNode `json:"nodes,omitempty"`
	Edges []*DgraphEdge `json:"edges,omitempty"`
}
