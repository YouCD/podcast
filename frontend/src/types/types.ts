export interface Rss {
  id: number;
  title: string;
  // content: string;
  llm_content: string;
  llm_html: string;
  llm_result: string;
  date: string;
  categories: string;
  // score: number;
  source: string;
  link: string;
  md5: string;
  created_at: string;
  updated_at: string;
  dgraph: DgraphResponse | null;
}

export interface NextLlmHtml {
  current: Rss;
  next: Rss;
  previous: Rss | null;
}

export interface TimeStay {
  time_stay: number;
  md5: string;
}

export interface QinNiu {
  accessKey: string;
  secretKey: string;
  bucket: string;
  publicCdnPath: string;
}

// 添加Report接口定义
export interface Report {
  id: number;
  question: string;
  created_at: string;
  updated_at: string;
  time_array: string;
  podcast_mp3_url: string;
}

export interface llmResult {
  categories: string;
  subtitle: string;
  contentSummary: string;
  keyPoints: string[];
  opinion: string;
  specifics: Map<string, string>;
  summarize: string;
}

export interface status {
  low_quality_count: number;
  read_count: number;
}

export interface user {
  name: string;
  password: string;
}

export interface tokenInfo {
  message: string;
  token: string;
  success: boolean;
}

export interface TokenPayload {
  id: number;
  name: string;
  exp: number;
}


export interface messageInfo {
  role: string;
  content: string;
}

// 定义ChatRecord类型
export interface sessionItem {
  id?: number;
  session_id: string;
  user_id: number;
  title: string;
  created_at?: string;
  updated_at?: string;
}

export interface changeTitleReq {
  message: messageInfo[];
  session_id: string;
}

export interface changeTitleResp {
  title: string;
}

export interface ResponseData {
  data: any;
  message: string;
  code: number;
}

// 定义边（Edge）结构
export interface DgraphEdge {
  source: string; // 目标节点 UID
  target: string; // 边的名称（可选）
  value: string;
}

export interface DgraphNode {
  uid: string;
  id: string;
  name: string;
  "dgraph.type": string;
  aliases: string[];
  category: number;
}

export interface DgraphResponse {
  nodes: DgraphNode[]; // 目标节点 UID
  edges: DgraphEdge[]; // 边的名称（可选）
}

// Prompt相关类型定义
export interface Prompt {
  id: number;
  key_name: string;
  data: string;
  genre: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

// Prompt创建请求类型
export interface CreatePromptRequest {
  key_name: string;
  data: string;
}

// Prompt更新请求类型
export interface UpdatePromptRequest {
  key_name?: string;
  data?: string;
}

// Template相关类型定义
export interface Template {
  id: number;
  key_name: string;
  data: string;
  genre: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

// Template创建请求类型
export interface CreateTemplateRequest {
  key_name: string;
  data: string;
}

// Template更新请求类型
export interface UpdateTemplateRequest {
  key_name?: string;
  data?: string;
}

export interface msgRequest {
  id?: number;
  created_at?: string;
  session_id: string;
  role: string;
  content?: string;
  uuid:string;
  typing?:boolean|undefined;
  reasoning_content: string;
  showReasoningContent?: boolean|undefined;
  reasoning_typing?: boolean|undefined;
  loading?: boolean|undefined;
  // 新增：计划相关（前端渲染时为对象）
  plan?: PlanData;
  // 新增：步骤列表（前端渲染时为对象数组）
  steps?: StepInfo[];
  // 新增：思考内容（流式）
  think_content?: string;
  // 思考内容是否启用打字机效果（SSE 流式时为 true）
  think_typing?: boolean;
  // 消息类型标识
  message_type?: 'normal' | 'plan' | 'step';
}

// 用于发送给后端的消息类型（plan 和 steps 为 JSON 字符串）
export interface msgRequestForBackend {
  id?: number;
  created_at?: string;
  session_id: string;
  role: string;
  content?: string;
  uuid: string;
  reasoning_content: string;
  // 发送给后端时为 JSON 字符串
  plan?: string;
  steps?: string;
  think_content?: string;
  message_type?: 'normal' | 'plan' | 'step';
}
// 后端返回的原始消息类型（plan 和 steps 为 JSON 字符串）
export interface RawMessage {
  id?: number;
  created_at?: string;
  session_id: string;
  role: string;
  content?: string;
  uuid: string;
  reasoning_content: string;
  // 后端返回时为 JSON 字符串
  plan?: string;
  steps?: string;
  think_content?: string;
  message_type?: 'normal' | 'plan' | 'step';
}

export interface sessionResponse {
  messages: RawMessage[];
  session: sessionItem;
}

// ===== 新增：计划和步骤相关类型定义 =====

// 计划步骤定义
export interface PlanStep {
  id: number;
  description: string;
  tool_name: string;
  tool_args: string;
  reason: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  // result?: string;
}

// 计划数据
export interface PlanData {
  query: string;
  steps: PlanStep[];
  current_step: number;
  is_complete: boolean;
}

// 步骤信息（用于渲染）
export interface StepInfo {
  step_id: number;
  description: string;
  reason: string;
  tool_name: string;
  tool_args?: Record<string, any>;
  status: 'pending' | 'running' | 'completed' | 'failed';
  result?: string;
  expanded?: boolean; // 是否展开结果
}

// SSE 事件数据类型
export interface PlanCreatedEvent {
  plan: PlanData;
}

export interface StepStartEvent {
  step_id: number;
  description: string;
  reason: string;
  tool_name: string;
}

export interface StepResultEvent {
  step_id: number;
  result: string;
  status: 'completed' | 'failed';
  tool_args?: Record<string, any>;
}

export interface ThinkEvent {
  message: string;
}

export interface MessageEvent {
  message: string;
  data?: string;
}
