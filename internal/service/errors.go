package service

import "errors"

var (
	// ErrNotPrompt 表示记录不是 Prompt 类型（Genre!=1）
	ErrNotPrompt = errors.New("不是有效的Prompt记录")
	// ErrNotTemplate 表示记录不是 Template 类型（Genre!=2）
	ErrNotTemplate = errors.New("不是有效的Template记录")
	// ErrRssNotFound 表示未找到对应 RSS 内容
	ErrRssNotFound = errors.New("RSS content not found")
)

// 供 prompt/template 服务使用，便于 handlers 做 errors.Is 判断
var (
	errNotPrompt   = ErrNotPrompt
	errNotTemplate = ErrNotTemplate
	errRssNotFound = ErrRssNotFound
)
