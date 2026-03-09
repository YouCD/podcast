package template

import (
	"context"
	"encoding/json"
	"os"
	"podcast/config"
	"podcast/pkg/types"
	"testing"
	"text/template"

	"github.com/youcd/toolkit/log"
)

func mainA() {
	// 把用户给的 JSON 读成 Report 结构体（字段缺失也能跑）
	type llmResult struct {
		Title      string `json:"Title"`
		Date       string `json:"Date"`
		Link       string `json:"Link"`
		Source     string `json:"Source"`
		Categories string `json:"Categories"`
		types.LLMResult
	}

	//var rr map[string]interface{}
	var rr llmResult
	err := json.Unmarshal([]byte(`{"qa":[{"question":"苹果新宣传片《我并不出众》的核心理念是什么？","answer":"强调残障学生与同龄人一样普通，科技助其独立完成学业。"},{"question":"视频中重点展示的Mac辅助功能有哪些？","answer":"放大器、旁白（VoiceOver）、布莱叶盲文接入等。"},{"question":"该宣传片背景音乐是否为公开曲目？","answer":"未在公开渠道找到信息，疑似苹果定制作品。"}],"contentSummary":"2025年12月2日，苹果发布名为《我并不出众》的宣传视频，聚焦Mac、iPhone和iPad的辅助功能如何支持不同能力的学生完成大学学业。视频记录了学生在校园生活、课堂学习、体育运动及实验中的真实场景，展现其借助科技克服挑战的独立姿态。核心功能包括屏幕放大、语音朗读、盲文接入与实时字幕等。影片通过非煽情化叙事，传递残障学生作为普通学生的平等价值，引发社会对无障碍设计的关注。","subtitle":"苹果｜辅助功能宣传视频","affect":["视力差也得靠放大器，看不清就别想上课","听不清课只能靠实时字幕，错一个字就掉队","用盲文输入慢得像蜗牛，交作业总卡在最后一秒"]}`), &rr)
	if err != nil {
		log.WithCtx(context.Background()).Panicf("json decode: %v", err)
	}

	tmpl, err := template.New("a.html").Funcs(funcMap).Parse(defaultTmpl)
	if err != nil {
		log.WithCtx(context.Background()).Error("parse tmpl: %v", err)
		return
	}
	if err := tmpl.Execute(os.Stdout, rr); err != nil {
		log.WithCtx(context.Background()).Panicf("execute: %v", err)
	}
}
func init() {
	config.LoadAppConfig("/home/ycd/self_data/source_code/podcast/config/config.yaml")
}
func Test_mainA(t *testing.T) {
	mainA()
}
