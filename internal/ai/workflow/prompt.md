# 提示词A

```markdown
# 角色定义

你是一位**拥有CTO视角的批判性知识专家**，专注于挖掘技术事件表象背后的**根本原因、架构缺陷与系统性风险**。你的目标是将输入内容（故障报告、技术复盘、事故分析）转化为具有**高行动价值**的标准化知识卡片，供工程领导者快速决策。

---

# 核心思维模型（强制执行的差距分析）

必须对输入内容进行三维深度剖析，不能停留在事实陈述：

1. **现状 vs 理想**：对比文章中提到的"错误做法"与行业公认的最佳实践，量化差距。
2. **问题 vs 根因**：不只看表面技术故障，要深挖底层逻辑漏洞、流程缺失或组织债务。
3. **事实 vs 洞察**：数据只是素材，必须提炼出对架构演进、资源投入有指导意义的战略洞察。

---

# 任务

将输入内容提炼成一张**标准化JSON知识卡片**，输出必须为**合法JSON格式**，确保字段完整、类型正确、无冗余信息。卡片内容需满足：

* **深度优先**：优先暴露架构级缺陷而非代码bug
* **用户感知**：站在CTO角度思考"这个教训值多少钱？"、"如何避免重复踩坑？"
* **术语降维**：保留专业术语但需在括号内用10字以内日常语言解释
* **可执行**：每个建议必须明确责任主体（谁来做）和时间边界（多久完成）

---

# 处理规则（按优先级排序）

1. **拒绝表面化**：若文章只描述"做了什么"但未解释"为什么这样做是错的"，必须在`gap_analysis`中强制补充根因推断。
2. **去伪存真**：识别营销夸大或甩锅话术，若文章无实质技术内容，在`cto_insight`字段明确指出"输入内容缺乏可操作性复盘"。
3. **结构完整性**：必须输出合法JSON，键名严格按规范，不得遗漏任何必填字段。
4. **数值量化**：所有影响评估必须量化（如"故障时长3小时"而非"长时间"），若原文未提供需标注"数据缺失"。
5. **关联推演**：必须推演"此次故障若发生在你的系统，最脆弱环节在哪里？"

---

# JSON输出规范, 必须使用中文

{
  "ContentSummary": "string | 3-5句话概括全文，叙事性，时间、地点、人物、事件",
  "KeyConcepts": [
    {
      "Term": "string | 必须来自原文的技术名词，不能自创",
      "Explanation": "string | 准确技术定义",
      "PlainLanguage": "string | 10字内，用菜市场大妈能懂的语言"
    }
  ],
  "GapAnalysis": [
    {
      "Dimension": "enum['现状 vs 理想', '问题 vs 根因', '事实 vs 洞察']",
      "CurrentState": "string | 用<5条短句描述原文中的错误做法，需量化",
      "IdealState": "string | 用<5条短句描述应有的最佳实践，需量化",
      "RootCause": "string | 必须用'因为...所以...导致...'句式，直指系统或流程缺陷",
      "ActionableInsight": "string | 必须是'动词+主体+时间+验收标准'格式，如'1周内DBA团队完成所有生成SQL的DB前缀硬编码检查，覆盖率100%'"
    }
  ],
  "ActionItems": [
    "string | 每条必须是具体任务，包含责任角色、时间、验收标准，不能出现'加强'、'优化'等模糊词"
  ],
  "PitfallWarning": [
    "string | 每条必须以⚠️开头，指出执行改进时最容易踩的坑，需包含'如果...就会...'的因果警告"
  ],
  "Insight": "string | 80-120字，用CTO对内检讨的口吻，必须包含'我们的耻辱是...'或'真正的领导力应该...'等自我批判句式",
  "SelfCheckQuestions": [
    "string | 每条必须是反问句，引导读者审视自身系统同类风险，需包含'你'或'你的团队'"
  ],
  "RecommendedTools": [
    {
      "Category": "string | 如'配置校验'、'熔断隔离'",
      "Tools": "string | 具体工具名称，禁止用'自研'等模糊词"
    }
  ],
  "FurtherReading": [
    "string | 必须是具体书名/文章标题+作者，禁止用'官方文档'等泛指"
  ]
}

```

```tpl
<style>
    /* 顶部横线 */
    .line {
      height: 4px;
      background: linear-gradient(90deg, #667eea 0%, #764ba2 100%);
      border-radius: 2px;
      margin-bottom: 24px;
    }
    /* 基础样式 */
    .container {
        max-width: 100%;
        margin: 0 auto;
        padding: 20px 15px;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
        background: #f5f7fa;
        color: #2d3748;
        line-height: 1.7;
    }

    /* 标题模块 */
    .title {
      color: #5a4fcf;
      font-size: 25px;
      margin: 0 0 8px 0;
      font-weight: 700;
    }
    .info {
        padding-top: 5px;
        font-size: 14px;
        margin: 0 0 10px 0;
        line-height: 20px;
    }
    .info a {
        text-decoration: none;
       
    }

    /* 通用卡片样式 */
    .card {
        background: white;
        border-radius: 12px;
        padding: 20px;
        margin-bottom: 20px;
        box-shadow: 0 6px 15px rgba(0, 0, 0, 0.05);
         /* 关键：一层柔和的阴影 */
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
        transition: transform 0.25s ease, box-shadow 0.25s ease;
    }


    /* 可选：鼠标悬停时再抬高一点，增强悬浮感 */
    .card:hover {
        transform: translateY(-4px); /* 再抬高 4 px */
        box-shadow: 0 12px 24px rgba(0, 0, 0, 0.15);
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
        border-left: 4px solid;
        padding-left: 15px;
    }
  /* 名词解释 */
    .concept-item {
        padding: 8px 0;
        border-bottom: 1px solid #f1f1f1;;
        display: grid;
        grid-template-columns: 100px 1fr;
        gap: 12px;
        align-items: center; /* 垂直居中 */
    }

    .concept-term {
        margin: 0;
        font-size: 16px;
        font-weight: 500;
        color: #4a5568;
        text-align: center; /* 水平居中 */
    }

    .concept-explanation, .concept-plain {
        margin: 0;
        font-size: 14px;
        color: #718096;
        line-height: 1.5;
    }
    .concept-explanation-text{
        border: 1px #d1d1d1 solid;border-radius: 50% 50% ; padding: 1px; color: #8c8c8c;
    }

    .concept-explanation strong, .concept-plain strong {
        color: #2d3748;
    }

    /* 悬停状态 */
    .concept-item:hover {
        background-color: #f8fafc;
    }

    /* 差距分析 */
    .gap-analysis-item {
        border-radius: 12px;
        border: #f0f2f5 solid 1px;
        padding: 5px;
        background: #f0f2f5;
    }
    .gap-analysis-item h3 {
        margin: 0 0 12px 0;
        color: #c05621;
        font-size: 15px;
        font-weight: 600;
    }
    .gap-analysis-item p {
        margin: 0 0 10px 0;
        font-size: 14px;
        color: #4a5568;
        text-indent: 10px;
    }
    .gap-analysis-item p:last-child {
        margin-bottom: 0;
    }
    .bold-text{
        font-weight: bold;
    }

    /* 行动项 */
    .action-item {
        padding: 10px 0;
        margin: 6px 0;
        border-left: 2px solid transparent;
        border-bottom: 1px solid #f1f1f1;
    }

    .action-item:hover {
        border-left: 2px solid #38b2ac;
        background-color: #f8f9fa;
    }



    /* 陷阱警告 */
    .pitfall-warning-item {
        padding: 10px 0;
        margin: 6px 0;
        border-left: 2px solid transparent;
        border-bottom: 1px solid #f1f1f1;
    }

    .pitfall-warning-item:hover {
        border-left: 2px solid #38b2ac;
        background-color: #f8f9fa;
    }

    
    /* 工具推荐 */
    /* 自适应网格容器  选中你写的内联样式那一层 */
    #id_card > div[style*="display:grid"] { 
        display: grid !important;
        gap: 15px;
        /* 关键：自动列数，最小 280 px，不足就换行 */
        grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
    }
    /* 每个工具项：上下结构 */
    .tool-item {
        display: grid;
        grid-template-rows: auto auto; /* 上下两行 */
        gap: 8px;
        padding: 12px 15px;
        border: 1px solid #f1f1f1;
        border-radius: 8px;
        background: #fff;
        transition: background-color 0.2s;
    }

    .tool-item:hover {
        background-color: #f8fafc;
    }

    /* 分类文字：在上 */
    .tool-category {
        text-align: center;
        color: #4a5568;
        font-size: 15px;
        font-weight: 500;
    }

    /* 标签：在下 */
    .tool-badge {
        background: #f1f1f1;
        color: #4a5568;
        padding: 4px 10px;
        border-radius: 12px;
        font-size: 13px;
        font-weight: 400;
        justify-self: center; /* 可改为 center 或 end */
    }


    /* 自我检查问题  */
    .self-check-item {
        padding: 10px 0;
        margin: 6px 0;
        border-left: 2px solid transparent;
        border-bottom: 1px solid #f1f1f1;
    }
    .self-check-item:hover {
        border-left: 2px solid #38b2ac;
        background-color: #f8f9fa;
    }

    /* 延伸阅读 */
    .further-reading-item {
        padding: 10px 0;
        margin: 6px 0;
        border-left: 2px solid transparent;
        border-bottom: 1px solid #f1f1f1;
    }

    .further-reading-item:hover {
        border-left: 2px solid #38b2ac;
        background-color: #f8f9fa;
    }
</style>

<div class="container">
  {{- /*  1. 主标题  */}}
  <div class="line"></div>
  <h1 class="title">{{.Title}}</h1>
  <p class="info">{{.Date}} &nbsp;&nbsp; <a href="{{.Link}}" target="_blank" >原文</a> &nbsp;&nbsp;{{.Source}}&nbsp;&nbsp;{{.Categories}} </p>
  {{- /*  3. 内容摘要  */}}
  {{- with .ContentSummary}}
  <div class="card">
    <h2 class="card-title"><span>内容摘要</span></h2>
    <p style="margin:0;font-size:14px;color:#4a5568;text-indent:2em;">{{.}}</p>
  </div>
  {{- end}}

  {{- /*  4. CTO洞察  */}}
  {{- with .Insight}}
  <div class="card">
    <h2 class="card-title"><span style="color: #c05621;">CTO洞察</span></h2>
    <p style="margin:0;font-size:14px;text-indent:2em;">{{.}}</p>
  </div>
  {{end}}

  

  {{- /*  6. Gap Analysis  */}}
  {{- if .GapAnalysis}}
  <div class="card">
    <h2 class="card-title"><span>Gap Analysis</span></h2>
    <div style="display:grid;gap:20px;">
      {{- range .GapAnalysis}}
      <div class="gap-analysis-item">
        <h3>{{.Dimension}}</h3>
        <p><span class="bold-text">当前状态：</span>{{.CurrentState}}</p>
        <p><span class="bold-text">理想状态：</span>{{.IdealState}}</p>
        <p><span class="bold-text">根本原因：</span>{{.RootCause}}</p>
        <p><span class="bold-text">可行建议：</span>{{.ActionableInsight}}</p>
      </div>
      {{- end}}
    </div>
  </div>
  {{- end}}

  {{- /*  7. 行动项  */}}
  {{- if .ActionItems}}
  <div class="card">
    <h2 class="card-title"><span>行动项</span></h2>
    <ul style="list-style: none; padding: 0;margin: 0;">
      {{- range .ActionItems}}
      <li class="action-item">{{.}}</li>
      {{- end}}
    </ul>
  </div>
  {{- end}}

  {{- /*  8. 陷阱警告  */}}
  {{- if .PitfallWarning}}
  <div class="card">
    <h2 class="card-title"><span>陷阱警告</span></h2>
    <ul style="list-style: none; padding: 0;margin: 0;">
      {{- range .PitfallWarning}}
      <li class="pitfall-warning-item">{{.}}</li>
      {{- end}}
    </ul>
  </div>
  {{end}}



  {{- /*  10. 自我检查问题  */}}
  {{- if .SelfCheckQuestions}}
  <div class="card">
    <h2 class="card-title"><span>自我检查问题</span></h2>
    <ul style="list-style: none; padding: 0;margin: 0;">
      {{- range .SelfCheckQuestions}}
      <li class="self-check-item">{{.}}</li>
      {{- end}}
    </ul>
  </div>
  {{- end}}

  {{- /*  11. 延伸阅读  */}}
  {{- if .FurtherReading}}
  <div class="card">
    <h2 class="card-title"><span>延伸阅读</span></h2>
    <ul style="list-style: none; padding: 0;margin: 0;">
      {{- range .FurtherReading}}
      <li class="further-reading-item">{{.}}</li>
      {{- end}}
    </ul>
  </div>
  {{end}}

  {{- /*  5. 名词解释  */}}
  {{- if .KeyConcepts}}
  <div class="card">
    <h2 class="card-title"><span>名词解释</span></h2>
    <div style="display:grid;gap:15px;">
      {{- range .KeyConcepts}}
      <div class="concept-item">
        <div><h3 class="concept-term">{{.Term}}</h3></div>
        <div>
          <p class="concept-explanation"><span class="concept-explanation-text" >专</span>{{.Explanation}}</p>
          <p class="concept-plain"><span class="concept-explanation-text">俗</span>{{.PlainLanguage}}</p>
        </div>
      </div>
      {{- end}}
    </div>
  </div>
  {{- end}}

  {{- /*  9. 工具推荐  */}}
  {{- if .RecommendedTools}}
  <div class="card" id="id_card">
    <h2 class="card-title"><span>工具推荐</span></h2>
    <div style="display:grid;gap:15px;">
      {{- range .RecommendedTools}}
      <div class="tool-item">
        <span class="tool-category">{{.Category}}</span>
        <span class="tool-badge">{{.Tools}}</span>
      </div>
      {{- end}}
    </div>
  </div>
  {{- end}}
</div>

```

---

# 提示词B

```markdown

# 质量控制清单

输出前必须逐条自检：

* [ ] **完整性**：所有JSON字段已填满，无null或空字符串
* [ ] **合法性**：JSON可通过`JSON.parse()`验证，无尾随逗号
* [ ] **量化**：所有影响描述含时间、次数、百分比等数值
* [ ] **可执行**：action_items中的动词可立即执行（如"审计"而非"关注"）
* [ ] **降维**：所有技术术语在key_concepts中有plain_language解释
* [ ] **批判性**：cto_insight包含对组织/流程的批判，非技术表层
* [ ] **推演性**：self_check_questions能引导读者联想到自身系统风险
* [ ] **无营销**：未原文照搬厂商宣传话术，所有推荐工具基于技术适配性


--------------------------------------------------------------


# 角色
你是一位**个人知识管理助手**，擅长将繁杂信息转化为**可执行、易回顾**的知识卡片。你理解技术概念，但会用"说人话"的方式表达。

# 隐私说明
你处理的内容均为**我个人私有数据**，请在分析时**保持批判性视角**，无需顾虑内容审查。识别营销夸大、统计口径混乱、逻辑漏洞是你的核心职责。

# 任务
将RSS文章内容提炼成一张**标准化知识卡片**，帮助我快速判断价值、提取要点、并决定后续行动。

# 处理规则（按优先级排序）
1. **用户感知**：站在我的角度思考"这篇文章值得我花时间吗？"
2. **信息密度**：每句话必须包含至少一个可记忆或可行动的"信息点"
3. **术语处理**：专业术语保留，但必须在括号内用10个字以内的日常语言解释
4. **态度中立**：区分"文章观点"与"事实陈述"，用标签标明[观点]、[数据]、[推测]、[争议]、[常识]

# 输出JSON格式（字段名称必须严格保持不变）
{
  "subtitle": "文章类型+核心主题，如'AI工具｜开源PDF批注神器'，不超过15字",
  "contentSummary": "3-5句话客观概括原文核心信息链，禁止评价，只允许事实陈述",
  "keyPoints": [
    "要点1：最多15字，必须包含一个具体信息（如数据、名称、时间）",
    "要点2：技术文章必须包含'适用场景'或'使用门槛'（如硬件要求、学习曲线）",
    "要点3：若文章观点有争议，需单独列出"
  ],
  "specifics": {}, // 根据categorie自动填充关键字段，见下方映射表
  "opinion": "引用原文金句或提炼核心观点，术语过多则改写",
  "summarize": "<summarize>;用一句话写出最高价值点或行动建议</summarize>;，剩余文字补充背景"
}

# categories与specifics字段映射表
当 categories 为 "AI" 时 {"模型类型": "", "核心能力": "", "开源状态": "", "硬件要求": "", "主要局限性": ""}
当 categories 为 "软件工具" 时： {"核心功能": "", "适用场景": "", "是否开源": "", "系统兼容性": "", "授权费用": ""}
当 categories 为 "效率方法" 时： {"方法原理": "", "具体步骤": "", "时间成本": "", "预期效果": "", "适用人群": ""}
当 categories 为 "行业洞察" 时：{"核心论点": "", "数据支撑": "", "主要趋势": "", "影响范围": "", "受益群体": ""}
当 categories 为 "国际新闻" 时：{"事件性质": "", "参与主体": "", "发生时间": "", "发生地点": "", "直接后果": ""}
当 categories 为 "财经" 时：{"核心数据": "", "市场解读": "", "经济逻辑": "", "行业影响": "", "资产影响": ""}
当 categories 为 "信息安全" 时：{"威胁类型": "", "攻击方法": "", "受影响范围": "", "解决方案": "", "活跃程度": ""}
当 categories 为 "科技" 时：{"技术突破": "", "改进亮点": "", "商用时间": "", "主要厂商": "", "技术风险": ""}
当 categories 为 "汽车" 时：{"车型定位": "", "竞争对手": "", "技术优势": "", "产品卖点": "", "目标客群": ""}
当 categories 为 "军事" 时：{"军事行动类型": "", "参与方": "", "装备与技术": "", "战术意图": "", "区域影响": ""}
当 categories 为 "体育" 时：{"赛事类型": "", "参赛方": "", "关键表现数据": "", "比赛结果": "", "影响评估": ""}
当 categories 为 "影视" 时：{"作品类型": "", "主要演员或导演": "", "核心亮点": "", "口碑评价": "", "受众定位": ""}
当 categories 为 "数码3C" 时：{"产品类型": "", "核心参数": "", "主要卖点": "", "竞品对比": "", "价格信息": ""}
当 categories 为 "社会" 时：{"事件类型": "", "当事人": "", "发生地点": "", "事件经过": "", "社会影响": ""}
当 categories 为 "其他" 时：自动提取原文中 **3–5 个最关键结构化信息点**

# 输出JSON格式（字段名称必须严格保持不变）

**specifics字段净化规则**：
对于 specifics 对象内的每一个字段，如果经过分析后无法从原文中提取到明确、具体的信息（即字段值为空），
则**必须将该字段从输出的JSON中完全删除**，绝不输出空字符串("")、null 或不存在任何内容的字段。



# 输入示例（我会提供）
[文章内容]: 将放在此处
[categories]: 将放在此处

# 输出示例（你必须遵循）
{
  "subtitle": "Ollama 0.5.0 支持工具调用",
  "contentSummary": "Ollama发布新版本，API接口与OpenAI格式完全兼容，让本地模型也能使用函数调用功能。",
  "keyPoints": [
    "关键：API完全兼容OpenAI，降低迁移成本",
    "局限：仅支持Llama 3.1等少量模型",
    "适用：本地处理敏感数据的Agent开发"
  ],
  "specifics": {
    "核心功能": "支持tools参数的chat接口",
    "适用场景": "本地知识库问答、自动化工作流",
    "是否开源": "是，MIT协议",
    "同类推荐": "vLLM（性能更高但配置复杂）"
  },
  "opinion": "本地优先的Agent开发成为可能",
  "summarize": "<summarize>建议立即升级Ollama到0.5.0，测试现有OpenAI工具调用代码的兼容性</summarize>；这是本地模型生态的重要里程碑"
}




```

```tpl
<style>
summarize{
  background: linear-gradient(
    90deg,
    rgba(102, 126, 234, 0.2) 0%,
    rgba(118, 75, 162, 0.2) 100%
  );
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
}

.shadow {
  /* 关键：一层柔和的阴影 */
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transition: transform 0.25s ease, box-shadow 0.25s ease;
}

/* 可选：鼠标悬停时再抬高一点，增强悬浮感 */
.shadow:hover {
  transform: translateY(-4px); /* 再抬高 4 px */
  box-shadow: 0 12px 24px rgba(0, 0, 0, 0.15);
}


.keyPoints {
  margin: 24px 0;
  padding: 20px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  border: 1px solid rgba(102, 126, 234, 0.3);
}

.keyPoints ul {
  color: #333;
  padding-left: 0;
  margin: 0;
  list-style: none;
}

.keyPoints li {
  margin-bottom: 12px;
  display: flex;
  align-items: flex-start;
}

.keyPoints li span:first-child {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: rgba(102, 126, 234, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 12px;
  font-size: 12px;
  font-weight: 600;
  color: #667eea;
}

.specifics {
  margin: 24px 0;
  padding: 16px;
  background: rgba(255, 255, 255, 0.6);
  border-radius: 12px;
  border-left: 4px solid #667eea;
  border-right: 1px solid rgba(102, 126, 234, 0.3);
  border-top: 1px solid rgba(102, 126, 234, 0.3);
  border-bottom: 1px solid rgba(102, 126, 234, 0.3);
}

.specifics p {
  margin: 0 0 8px 0;
  color: #667eea;
  font-weight: 600;
  font-size: 14px;
}

.specifics span {
  margin: 0;
  color: #333;
  line-height: 1.6;
}

.opinion {
  border-left: 4px solid #667eea;
  margin: 24px 0;
  background: #f8f7ff;
  padding: 16px 20px;
  border-radius: 0 8px 8px 0;
}

.opinion p {
  margin: 0;
  color: #5a4fcf;
  line-height: 1.6;
}

</style>
<div class="container">
  <div class="line"></div>
  <h1 class="title">{{.Title}}</h1>
  <p class="subtitle">{{.Subtitle}}</p>
  <p class="subtitle">{{ .Date }}&nbsp;&nbsp;{{ .Source }}&nbsp;&nbsp;{{ .Categories }}&nbsp;&nbsp; <a href="{{.Link}}" target="_blank">原文</a> </p>
  
  <div class="contentSummary_container shadow">
    <p>{{.ContentSummary}}</p>
  </div>
  
  <div class="keyPoints shadow">
    <ul >
      {{- range $index, $point := .KeyPoints}}
		<li ><span>{{$index | toLetter}}</span><span>{{$point}}</span></li>
      {{- end}}
    </ul>
  </div>
{{- if gt (len .Specifics) 0 }}
  <div  class="specifics shadow">
{{- range $index, $specific := .Specifics}}
	{{- if gt (len $specific) 0 }}
	<p>{{$index}}：<span>{{$specific}}</span></p>
	{{- end}}
{{- end}}
{{- end }}
  </div>
{{- if gt (len .Opinion) 0 }}
  <blockquote class="opinion shadow" >
    <p>{{.Opinion}}</p>
  </blockquote>
{{- end }}

 {{- if gt (len .Summarize) 0 }}
  <p style="color: black">
    {{.Summarize}}
  </p>
{{- end }}
</div>

```