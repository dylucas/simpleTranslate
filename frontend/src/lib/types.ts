// 前端核心类型定义：与后端 Go 结构体对应（wailsjs/go/models.ts 自动生成版本以外补充）。
// 这里提供业务相关的派生类型，便于组件与 lib 模块复用。

// 错误类别：与后端 errors.go 的 ErrCode* 常量保持一致
export type TranslateErrorCode =
  | "unknown"
  | "credentials"
  | "network"
  | "timeout"
  | "rate_limit"
  | "invalid_input"
  | "service_unavailable";

// 翻译结果（与 main.TranslateResult 对应）
export interface TranslateResult {
  source: string;
  autoSrc: string;
  target: string;
  text: string;
  error?: string;
  errorCode?: TranslateErrorCode;
}

// 单引擎翻译结果（对照模式）
export interface EngineTranslateResult {
  engine: string;
  text: string;
  error?: string;
  errorCode?: TranslateErrorCode;
}

// 多引擎翻译结果
export interface MultiTranslateResult {
  source: string;
  autoSrc: string;
  target: string;
  results: Record<string, EngineTranslateResult>;
}

// 历史记录条目
export interface HistoryEntry {
  id: number;
  input: string;
  output: string;
  source: string;
  target: string;
  time: string;
}

// 错误提示状态
export interface ErrorToast {
  msg: string;
  code: TranslateErrorCode;
  canRetry: boolean;
  showSettings: boolean;
  ts: number;
}
