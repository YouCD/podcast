package daily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"podcast/config"
	"podcast/pkg/byte_dance"

	jsoniter "github.com/json-iterator/go"
	"github.com/youcd/toolkit/log"
)

const (
	msSpace = `https://youchangding-soulx-podcast-1-7b.ms.show`
	apiURL  = `/gradio_api/call/fast_synthesize`
)

func generatePodcastVideo(ctx context.Context, state *graphState) (*graphState, error) {
	if state.report.PodcastContent == "" {
		log.WithCtx(ctx).Warnf("播客文本内容为空")
		return state, nil
	}
	log.WithCtx(ctx).Info("播客音频生成开始")
	file, err := genPodcastVideo(ctx, state.report.PodcastContent)
	if err != nil {
		log.WithCtx(ctx).Errorf("播客文件生成失败: %s", err)
		return state, nil
	}

	//file, err := save2File(ctx, podcastUrl)
	//if err != nil {
	//	log.WithCtx(ctx).Errorf("保存播客文件失败: %s", err)
	//	return state, nil
	//}

	state.report.PodcastMP3URL = file
	log.WithCtx(ctx).Info("播客音频生成完成")
	return state, nil
}

func genPodcastVideoA(ctx context.Context, data string) (string, error) {
	// 创建请求数据
	//nolint:modernize
	jsonData := []byte(fmt.Sprintf(`{
    "data": [
        %#v
    ]
}`, data))

	log.WithCtx(ctx).Debug(string(jsonData))

	// 发送POST请求获取EVENT_ID
	eventID, err := getEventID(ctx, jsonData)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return "", fmt.Errorf("failed to get event ID: %w", err)
	}

	log.WithCtx(ctx).Debugf("Event ID: %s", eventID)
	// 使用EVENT_ID发送GET请求获取结果
	audioURL, err := streamResult(ctx, eventID)
	if err != nil {
		log.WithCtx(ctx).Errorf("Event ID: %s", eventID)
		return "", err
	}

	if audioURL != "" {
		log.WithCtx(ctx).Infof("Audio URL extracted: %s", audioURL)
		return audioURL, nil
	}
	return "", ErrNoAudioURL
}

func getEventID(ctx context.Context, jsonData []byte) (string, error) {
	// 创建HTTP请求
	url := msSpace + apiURL
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", ErrStatusNotOK
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}

	// 解析响应获取EVENT_ID

	bodyStr := string(body)

	parts := strings.Split(bodyStr, "\"")
	if len(parts) < 4 {
		return "", ErrBodyError
	}

	eventID := parts[3]
	return strings.TrimSpace(eventID), nil
}

func streamResult(ctx context.Context, eventID string) (string, error) {
	// 创建GET请求URL
	url := fmt.Sprintf("%s/%s", msSpace+apiURL, eventID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	// 发送GET请求
	client := &http.Client{
		Timeout: 0, // 不设置超时，因为流式响应可能需要很长时间
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending GET request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", ErrStatusNotOK
	}

	all, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %w", err)
	}
	log.WithCtx(ctx).Debug(string(all))
	//nolint:all
	var dataArray []interface{}
	//nolint:all
	for _, s := range strings.Split(string(all), "\n") {
		if strings.Contains(s, "audio.wav") {
			// 尝试解析JSON数据
			dataPart := strings.TrimPrefix(s, "data:")
			dataPart = strings.TrimSpace(dataPart)
			// 解析数组中的第二个元素（索引1）
			if err := json.Unmarshal([]byte(dataPart), &dataArray); err != nil {
				log.WithCtx(ctx).Error(err)
				continue
			}
		}
	}
	log.WithCtx(ctx).Debugw("ms space", "result", dataArray)
	for _, jsonObj := range dataArray {
		marshal, err := jsoniter.Marshal(jsonObj)
		if err != nil {
			continue
		}
		log.WithCtx(ctx).Debug(string(marshal))
		urlStr := jsoniter.Get(marshal, "url").ToString()
		if urlStr != "" {
			return urlStr, nil
		}
	}
	return "", ErrNoAudioURL
}

func save2File(ctx context.Context, podcastUrl string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, podcastUrl, nil)
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download podcast: %w", err)
	}
	defer resp.Body.Close()
	fileName := fmt.Sprintf("%s.wav", time.Now().Format("2006-01-02_15_04_05"))
	f := path.Join(config.Cfg.Global.PodcastDir, fileName)
	err = os.MkdirAll(config.Cfg.Global.PodcastDir, 0o755)
	if err != nil {
		log.WithCtx(ctx).Error(err)
	}
	file, err := os.OpenFile(f, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.WithCtx(ctx).Error(err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.WithCtx(ctx).Error(err)
	}
	return fileName, nil
}
func genPodcastVideo(ctx context.Context, data string) (string, error) {
	fileName := fmt.Sprintf("%s.mp3", time.Now().Format("2006-01-02_15_04_05"))
	f := path.Join(config.Cfg.Global.PodcastDir, fileName)
	log.WithCtx(ctx).Debugf("fileName: %s", f)

	_ = os.MkdirAll(config.Cfg.Global.PodcastDir, 0o755)
	err := byte_dance.PodCast(ctx, config.Cfg.ByteDance.AppID, config.Cfg.ByteDance.AccessToken, f, data)
	if err != nil {
		log.WithCtx(ctx).Error(err)
		return "", err
	}
	return fileName, nil
}
