export type EngineId = "tencent" | "aliyun" | "baidu";

export type BaiduDomain =
  | "general"
  | "it"
  | "finance"
  | "machinery"
  | "senimed"
  | "novel"
  | "academic"
  | "aerospace"
  | "wiki"
  | "news"
  | "law"
  | "contract";

export type TranslateErrorCode =
  | "unknown"
  | "credentials"
  | "network"
  | "timeout"
  | "rate_limit"
  | "invalid_input"
  | "service_unavailable"
  | "cancelled";

export interface ServiceConfig {
  secretId: string;
  secretKey: string;
  region: string;
}

export interface BaiduConfig {
  appId: string;
  secretKey: string;
  domain: BaiduDomain;
}

export interface CloudConfig {
  version: number;
  tencent: ServiceConfig;
  aliyun: ServiceConfig;
  baidu: BaiduConfig;
  defaultEngine: EngineId;
  isDark: boolean;
  sidebarCollapsed: boolean;
  autoTranslate: boolean;
  sourceLanguage: string;
  targetLanguage: string;
  compareMode: boolean;
  compareEngines: EngineId[];
  clipboardWatch: boolean;
}

export interface TranslateRequest {
  requestId: string;
  text: string;
  source: string;
  target: string;
  engine: EngineId;
}

export interface MultiTranslateRequest {
  requestId: string;
  text: string;
  source: string;
  target: string;
  engines: EngineId[];
}

export interface TranslateResult {
  requestId: string;
  source: string;
  autoSrc: string;
  target: string;
  text: string;
  notice?: string;
  error?: string;
  errorCode?: TranslateErrorCode;
}

export interface EngineTranslateResult {
  requestId: string;
  engine: EngineId;
  text: string;
  notice?: string;
  error?: string;
  errorCode?: TranslateErrorCode;
}

export interface MultiTranslateResult {
  requestId: string;
  source: string;
  autoSrc: string;
  target: string;
  results: Partial<Record<EngineId, EngineTranslateResult>>;
}

export interface HistoryEntry {
  id: number;
  input: string;
  output: string;
  source: string;
  target: string;
  time: string;
}

export interface HistoryQuery {
  query: string;
  offset: number;
  limit: number;
}

export interface HistoryPage {
  entries: HistoryEntry[];
  total: number;
  hasMore: boolean;
}

export interface ErrorToast {
  msg: string;
  code: TranslateErrorCode;
  canRetry: boolean;
  showSettings: boolean;
  ts: number;
}
