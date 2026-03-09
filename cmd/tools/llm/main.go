package main

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/schema"
)

func main() {
	cfg := &qwen.ChatModelConfig{
		APIKey:  os.Getenv("APIKey"),
		BaseURL: os.Getenv("BaseURL"),
		Model:   os.Getenv("Model"),
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeText,
		},
		// Temperature: &t,
	}

	chatModel, err := qwen.NewChatModel(context.Background(), cfg)
	if err != nil {
		fmt.Println(err)
		return
	}
	msg, err := chatModel.Generate(context.Background(), []*schema.Message{{Role: "user", Content: "你好"}})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(msg.Content)
}
