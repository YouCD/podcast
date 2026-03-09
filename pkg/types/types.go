package types

type Mcp struct {
	Headers map[string]string `yaml:"headers" json:"headers"`
	URL     string            `yaml:"url" json:"url"`
}
