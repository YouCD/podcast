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
}
export interface sessionResponse {
  messages: msgRequest[];
  session: sessionItem;
}
