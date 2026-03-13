package byte_dance

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/youcd/toolkit/log"
)

// PodcastRound 对话轮次结构
type PodcastRound struct {
	Speaker string `json:"speaker"`
	Text    string `json:"text"`
}
type UsageResponse struct {
	Usage struct {
		InputTextTokens   int `json:"input_text_tokens"`
		OutputAudioTokens int `json:"output_audio_tokens"`
	} `json:"usage"`
}

// createPodcastFromDialogue 核心封装函数：传入对话列表，返回音频下载地址
func createPodcastFromDialogue(ctx context.Context, appid, accessToken string, dialogues []PodcastRound) (string, error) {
	var inputTextTokens, outputAudioTokens int
	var useageResp UsageResponse
	// 1. 构建 WebSocket 连接 Header
	header := newHeader(appid, accessToken)

	// 2. 建立 WebSocket 连接
	conn, r, err := websocket.DefaultDialer.DialContext(
		context.Background(),
		"wss://openspeech.bytedance.com/api/v3/sami/podcasttts",
		header,
	)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v, response: %v", err, r)
	}
	log.WithCtx(ctx).Info("Connection established, Logid: ", r.Header.Get("x-tt-logid"))

	// 确保连接关闭
	defer func() {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = conn.Close()
	}()

	// 3. 开始连接 (event=1)
	if err := StartConnection(conn); err != nil {
		return "", fmt.Errorf("StartConnection 失败: %v", err)
	}

	// 等待连接建立 (event=50)
	if _, err = WaitForEvent(conn, MsgTypeFullServerResponse, EventType_ConnectionStarted); err != nil {
		return "", fmt.Errorf("等待 ConnectionStarted 失败: %v", err)
	}

	// 4. 准备请求参数
	sessionID, payload := newPayload(dialogues)

	// 5. 开始会话 (event=100)
	if err := StartSession(conn, payload, sessionID); err != nil {
		return "", fmt.Errorf("StartSession 失败: %v", err)
	}

	// 等待会话建立 (event=150)
	if _, err = WaitForEvent(conn, MsgTypeFullServerResponse, EventType_SessionStarted); err != nil {
		return "", fmt.Errorf("等待 SessionStarted 失败: %v", err)
	}

	// 6. 结束会话 (event=102) - 告诉服务端参数传完了
	if err := FinishSession(conn, sessionID); err != nil {
		return "", fmt.Errorf("FinishSession 失败: %v", err)
	}

	// 7. 循环等待最终结果
	var audioURL string
	for {
		msg, err := ReceiveMessage(conn)
		if err != nil {
			return "", fmt.Errorf("接收消息失败: %v", err)
		}

		switch msg.MsgType {
		case MsgTypeError:
			// 收到错误直接返回
			return "", fmt.Errorf("收到服务端错误: %s", string(msg.Payload))

		case MsgTypeFullServerResponse:
			switch msg.EventType {
			case EventType_PodcastEnd: // 事件码 363 【turn0fetch0】
				var data struct {
					MetaInfo struct {
						AudioURL string `json:"audio_url"`
					} `json:"meta_info"`
				}
				if err := json.Unmarshal(msg.Payload, &data); err != nil {
					return "", fmt.Errorf("解析 PodcastEnd 失败: %v", err)
				}
				audioURL = data.MetaInfo.AudioURL
				log.WithCtx(ctx).Info("获取到音频下载地址", audioURL)
				break
			case EventType_UsageResponse:
				if err = json.Unmarshal(msg.Payload, &useageResp); err == nil {
					inputTextTokens += useageResp.Usage.InputTextTokens
					outputAudioTokens += useageResp.Usage.OutputAudioTokens
				}

			case EventType_SessionFinished:
				// 会话结束，退出循环
				goto END_LOOP
			}
		default:
			// 忽略音频分片消息 (MsgTypeAudioOnlyServer) 和其他中间状态
			// log.WithCtx(ctx).Infof("忽略中间消息 Type: %d, Event: %d", msg.MsgType, msg.EventType)
		}
	}

END_LOOP:
	// 8. 结束连接
	if err := FinishConnection(conn); err != nil {
		log.WithCtx(ctx).Warn("FinishConnection error:", err)
	}
	log.WithCtx(ctx).Infow("获取到使用情况", "inputTextTokens", inputTextTokens, "outputAudioTokens", outputAudioTokens)
	return audioURL, nil
}

func newPayload(dialogues []PodcastRound) (string, []byte) {
	reqParams := map[string]interface{}{
		"input_id":  uuid.New().String(),
		"action":    3, // 核心参数：使用对话文本模式
		"nlp_texts": dialogues,
		"audio_config": map[string]interface{}{
			"format":      "mp3",
			"sample_rate": 24000,
			"speech_rate": 0,
		},
		"input_info": map[string]interface{}{
			"return_audio_url": true, // 核心参数：要求返回下载链接
			"only_nlp_text":    false,
		},
		"speaker_info": map[string]interface{}{
			"random_order": false,
		},
		"use_head_music": true,
		"use_tail_music": false,
	}

	sessionID := uuid.New().String()
	payload, _ := json.Marshal(&reqParams)
	return sessionID, payload
}

func newHeader(appid string, accessToken string) http.Header {
	header := http.Header{}
	header.Set("X-Api-App-Id", appid)
	header.Set("X-Api-App-Key", "aGjiRDfUWi")
	header.Set("X-Api-Access-Key", accessToken)
	header.Set("X-Api-Resource-Id", "volc.service_type.10050")
	header.Set("X-Api-Connect-Id", uuid.New().String())
	return header
}

// downloadFile 简单的文件下载函数
func downloadFile(ctx context.Context, url, filePath string) error {
	log.WithCtx(ctx).Infof("开始下载: %s, 保存到: %s", url, filePath)

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// parseScript 解析剧本文本
// rawText: 原始文本，如 "S1:你好\nS2:你好呀"
// speakerMap: 标签到具体发音人ID的映射，如 map[string]string{"S1": "male_id", "S2": "female_id"}
func parseScript(rawText string, speakerMap map[string]string) []PodcastRound {
	var dialogues []PodcastRound

	// 正则解释：
	// ^\s*      : 匹配行首可能有空格
	// \[        : 匹配左方括号 [ (需要转义)
	// (\w+)     : 捕获组1，匹配标签文字 (如 S1, S2)
	// \]        : 匹配右方括号 ] (需要转义)
	// \s*       : 匹配标签后的空格
	// (.*)      : 捕获组2，匹配该行剩余内容 (台词)
	re := regexp.MustCompile(`^\s*\[(\w+)\]\s*(.*)$`)

	lines := strings.Split(rawText, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			label := matches[1] // 提取到的标签，如 "S1"
			text := matches[2]  // 提取到的台词

			// 从映射表中查找对应的发音人ID
			speakerID, ok := speakerMap[label]
			if !ok {
				// 如果没有配置映射，跳过或使用默认值
				continue
			}

			dialogues = append(dialogues, PodcastRound{
				Speaker: speakerID,
				Text:    text,
			})
		}
	}

	return dialogues
}

func PodCast(ctx context.Context, appid, accessToken, output, data string) error {
	speakerMapping := map[string]string{
		"S2": "zh_male_dayixiansheng_v2_saturn_bigtts",  // 男声
		"S1": "zh_female_mizaitongxue_v2_saturn_bigtts", // 女声
	}
	// 示例对话文本 - 这里可以替换为你的实际对话数据
	dialogues := parseScript(data, speakerMapping)

	// 1. 创建播客任务，获取下载地址

	log.WithCtx(ctx).Info("正在生成播客...")
	audioURL, err := createPodcastFromDialogue(ctx, appid, accessToken, dialogues)
	if err != nil {
		return fmt.Errorf("创建播客任务失败: %w", err)
	}

	if audioURL == "" {
		return errors.New("未获取到音频下载地址")
	}
	log.WithCtx(ctx).Infof("成功获取下载地址: %s", audioURL)
	// 2. 下载文件
	log.WithCtx(ctx).Infof("正在下载音频文件 -> %s", output)
	if err := downloadFile(ctx, audioURL, output); err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}

	log.WithCtx(ctx).Infof("下载完成，文件已保存至: %s", output)
	return nil
}
