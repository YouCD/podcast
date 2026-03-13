package handlers

import (
	"strings"

	"podcast/internal/database/models"
)

var waitHtml = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>生成报告</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <link href="https://fonts.googleapis.com/css2?family=DM+Sans:wght@400;500;600&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg-start: #667eea;
      --bg-end: #764ba2;
      --card: rgba(255, 255, 255, 0.95);
      --card-shadow: rgba(102, 126, 234, 0.25);
      --fg: #2d3748;
      --muted: #718096;
      --accent: #667eea;
    }

    * { margin: 0; padding: 0; box-sizing: border-box; }

    body {
      font-family: 'DM Sans', sans-serif;
      min-height: 100vh;
      background: linear-gradient(135deg, var(--bg-start) 0%, var(--bg-end) 100%);
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 24px;
    }

    .card {
      background: var(--card);
      border-radius: 20px;
      padding: 48px 40px;
      max-width: 400px;
      width: 100%;
      text-align: center;
      box-shadow: 0 20px 60px var(--card-shadow);
    }

    .spinner {
      width: 52px;
      height: 52px;
      border: 3px solid #e2e8f0;
      border-top-color: var(--accent);
      border-radius: 50%;
      margin: 0 auto 28px;
      animation: spin 0.9s linear infinite;
    }

    @keyframes spin {
      to { transform: rotate(360deg); }
    }

    .title {
      font-size: 1.3rem;
      font-weight: 600;
      color: var(--fg);
      margin-bottom: 10px;
    }

    .desc {
      color: var(--muted);
      font-size: 0.92rem;
      margin-bottom: 28px;
      line-height: 1.5;
    }

    .info {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      font-size: 0.82rem;
      color: var(--muted);
      padding: 10px 16px;
      background: #f7fafc;
      border-radius: 8px;
    }

    .info span {
      color: var(--accent);
      font-weight: 500;
    }

    @media (max-width: 480px) {
      .card { padding: 36px 24px; }
      .title { font-size: 1.15rem; }
    }

    @media (prefers-reduced-motion: reduce) {
      .spinner { animation: none; border-top-color: var(--accent); }
    }
  </style>
</head>
<body>
  <main>
    <div class="card">
      <div class="spinner"></div>
      <h1 class="title">服务器开始生成报告</h1>
      <p class="desc">后台正在处理，完成后将自动通知您</p>
      <div class="info">
        任务ID <span>{{.request_id}}</span>
      </div>
    </div>
  </main>
</body>
</html>
`

func modifyHtml5(report *models.Report) string {
	htmlContentSrc := strings.ReplaceAll(report.LLMResult, `https://cdn.jsdelivr.net/npm/chart.js`, `https://cdnjs.cloudflare.com/ajax/libs/Chart.js/4.5.0/chart.umd.min.js`)

	// 外部脚本（html2canvas）
	externalScript := `<script src="https://cdnjs.cloudflare.com/ajax/libs/html2canvas/1.4.1/html2canvas.min.js"></script>`

	// 悬浮按钮的 HTML 和样式
	floatingButton := `
<div id="floating-button" style="
    position: fixed;
    top: 50%;
    right: 10px;
    transform: translateY(-50%);
    width: 50px;
    height: 50px;
    background: rgba(255, 255, 255, 0.25);
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
    border: 1px solid rgba(255, 255, 255, 0.18);
    color: #333;
    border-radius: 50%;
    text-align: center;
    font-size: 24px;
    cursor: pointer;
    box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.37);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
"><svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="width: 24px; height: 24px;">
        <circle cx="18" cy="5" r="3"></circle>
        <circle cx="6" cy="12" r="3"></circle>
        <circle cx="18" cy="19" r="3"></circle>
        <line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line>
        <line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line>
    </svg></div>

</div>
<style>
    /* 悬停效果 */
    #floating-button:hover {
        background-color: #0056b3;
        transform: translateY(-50%) scale(1.1);
        box-shadow: 0 6px 10px rgba(0, 0, 0, 0.2);
    }
</style>
`

	// 自定义 JavaScript
	customScript := `
<script>
    document.addEventListener('DOMContentLoaded', function () {
        // 获取悬浮按钮
        const floatingButton = document.getElementById('floating-button');

        // 为悬浮按钮绑定点击事件
        floatingButton.addEventListener('click', async function () {
            try {
                // 使用 html2canvas 截图
                const captureElement = document.body;
                const canvas = await html2canvas(captureElement);
                const imgData = canvas.toDataURL('image/png');

                // 检查是否支持 navigator.share
                if (navigator.share && navigator.canShare) {
                    // 将 Base64 转换为 Blob
                    const blob = await fetch(imgData).then(res => res.blob());
                    const file = new File([blob], 'screenshot.png', { type: 'image/png' });

                    // 使用 Web Share API 分享
                    await navigator.share({
                        files: [file],
                        title: '分享截图',
                        text: '看看这张截图'
                    });
                    console.log('分享成功');
                } else {
                    // 如果不支持 navigator.share，则回退到下载
                    const downloadLink = document.createElement('a');
                    downloadLink.href = imgData;
                    downloadLink.download = 'screenshot.png';
                    downloadLink.click();
                    console.log('截图完成，已下载图片');
                }
            } catch (error) {
                console.error('操作失败:', error);
            }
        });
    });
</script>
`

	// 插入外部脚本到 <head> 中
	htmlContent := htmlContentSrc
	headEndIndex := strings.Index(htmlContent, "</head>")
	if headEndIndex == -1 {
		return htmlContentSrc
	}
	htmlContent = htmlContent[:headEndIndex] + externalScript + htmlContent[headEndIndex:]

	// 插入悬浮按钮到 <body> 的末尾
	bodyEndIndex := strings.Index(htmlContent, "</body>")
	if bodyEndIndex == -1 {
		return htmlContentSrc
	}
	htmlContent = htmlContent[:bodyEndIndex] + floatingButton + customScript + htmlContent[bodyEndIndex:]
	return htmlContent
}
