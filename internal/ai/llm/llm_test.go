package llm

import (
	"podcast/config"
	"testing"
)

func init() {
	_, err := config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
	if err != nil {
		panic(err)
	}
}

func TestNewLLMPool(t *testing.T) {
	NewLLMPool()
}
