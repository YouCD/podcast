package template

var defaultTmpl = `
<style>
    summarize {
        background: linear-gradient(90deg,
                rgba(102, 126, 234, 0.2) 0%,
                rgba(118, 75, 162, 0.2) 100%);
        color: #5a4fcf;
        padding: 2px 6px;
        border-radius: 4px;
        font-weight: 500;
    }

    .container {
        max-width: 800px;
        margin: 0 auto;
        padding: 24px;
        background: rgba(255, 255, 255, 0.85);
        border-radius: 16px;
        box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        backdrop-filter: blur(4px);
    }

    .line {
        height: 4px;
        background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
        border-radius: 2px;
        margin-bottom: 24px;
    }

    .title {
        color: #5a4fcf;
        font-size: 25px;
        margin: 0 0 8px 0;
        font-weight: 700;
    }

    .subtitle {
        color: #666;
        font-size: 14px;
        margin: 0 0 10px 0;
    }

    .subtitle a {
        text-decoration: none;
    }

    .contentSummary_container {
        margin: 24px 0;
        padding: 20px;
        background: rgba(255, 255, 255, 0.6);
        border-radius: 12px;
        border: 1px solid rgba(102, 126, 234, 0.2);
    }

    .contentSummary_container p {
        margin: 0;
        color: #333;
        line-height: 1.7;
		text-indent: 2em; /* 相对于当前字体大小 */
    }

    .shadow {
        /* 关键：一层柔和的阴影 */
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
        transition: transform 0.25s ease, box-shadow 0.25s ease;
    }

    /* 可选：鼠标悬停时再抬高一点，增强悬浮感 */
    .shadow:hover {
        transform: translateY(-4px);
        /* 再抬高 4 px */
        box-shadow: 0 12px 24px rgba(0, 0, 0, 0.15);
    }

    .container-box {
        margin: 24px 0;
        padding: 16px;
        background: rgba(255, 255, 255, 0.6);
        border-radius: 12px;
        border-right: 1px solid rgba(102, 126, 234, 0.3);
        border-top: 1px solid rgba(102, 126, 234, 0.3);
        border-bottom: 1px solid rgba(102, 126, 234, 0.3);
    }

    .container-box p {
        margin: 0 0 8px 0;
        color: #667eea;
        font-weight: 600;
        font-size: 14px;
    }

    .container-box span {
        margin: 0;
        color: #333;
        line-height: 1.6;
    }

 	.card-title {
        color: #4a5568;
        margin: 0 0 16px;
        font-size: 16px;
        font-weight: 600;
        display: flex;
        align-items: center;
    }
    .card-title span {
        border-left: 4px solid #21a640;
        padding-left: 15px;
    }

	/* 问答列表容器 */
	.qa-list {
		display: flex;
		flex-direction: column;
	}
	
	/* 单个问答项 */
	.qa-item {
		padding: 8px;
		background: rgba(255, 255, 255, 0.7);
		border-radius: 8px;
		border: 1px solid rgba(102, 126, 234, 0.15);
		transition: all 0.25s ease;
		margin-bottom: 5px;
	}
	
	.qa-item:hover {
		background: rgba(255, 255, 255, 0.9);
		transform: translateY(-2px);
		box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
	}
	
	/* 问题行样式 */
	.question-row {
		font-size: 16px;
		font-weight: 500;
		color: #5a4fcf;
		margin-bottom: 2px;
	}
	
	.question-row strong {
		color: #764ba2;
		font-weight: 600;
	}
	
	/* 回答行样式 */
	.answer-row {
		font-size: 14px;
		color: #4a5568;
		line-height: 1.6;
	}
</style>
<div class="container">
    <div class="line"></div>
    <h1 class="title">{{.Title}}</h1>
    <p class="subtitle">{{.Subtitle}}</p>
    <p class="subtitle">{{ .Date }}&nbsp;&nbsp;{{ .Source }}&nbsp;&nbsp;{{ .Categories }}&nbsp;&nbsp; <a href="{{.Link}}" target="_blank">原文</a> </p>
	{{- if gt (len .ContentSummary) 0 }}    
	<div class="contentSummary_container shadow">
        <p>{{.ContentSummary}}</p>
    </div>
	{{- end}}
	{{- if gt (len .QA) 0 }}
	<div class="container-box shadow">
	<h2 class="card-title"><span>核心问题</span></h2>
		<div class="qa-list">
			{{- range  .QA}}
			<div class="qa-item">
				<div class="question-row"><strong>{{.Question}}</div>
				<div class="answer-row">{{.Answer}}</div>
			</div>
			{{- end}}
		</div>
	</div>
	{{- end }}
</div>
`
