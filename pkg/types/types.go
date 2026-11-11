package types

type Mcp struct {
	Headers map[string]string `yaml:"headers" json:"headers"`
	URL     string            `yaml:"url" json:"url"`
	// Strategy 该代理下工具的使用策略（用于规划器"工具详解"段落），可选；
	// 配置后该代理的工具会进入规划提示词的策略段落。
	// 工具的一句话描述（summary）不在此配置，直接使用 MCP 服务器通过 ListTools 下发的工具描述。
	Strategy string `yaml:"strategy" json:"strategy"`
}
