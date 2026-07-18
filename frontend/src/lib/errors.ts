// 翻译错误前端处理：与后端 errors.go 的 ErrCode* 常量保持同步。
// 提供按错误类别差异化的人类可读消息与重试策略建议。
import type { TranslateErrorCode } from "./types";

// 错误类别常量，与 Go 后端 errors.go 一一对应
export const ErrorCodes: Record<string, TranslateErrorCode> = {
  Unknown: "unknown",
  Credentials: "credentials",
  Network: "network",
  Timeout: "timeout",
  RateLimit: "rate_limit",
  InvalidInput: "invalid_input",
  ServiceUnavailable: "service_unavailable",
};

// 默认用户可读消息（后端 message 也可用，这里作为兜底）
const DEFAULT_MESSAGES: Record<TranslateErrorCode, string> = {
  unknown: "翻译失败",
  credentials: "凭据无效或未配置，请在设置中检查",
  network: "网络连接失败，请检查网络",
  timeout: "请求超时，请稍后重试",
  rate_limit: "服务限流，请稍后再试",
  invalid_input: "输入文本为空",
  service_unavailable: "翻译服务暂时不可用",
};

// StructuredErrorResult 是 normalizeError 的返回类型
export interface StructuredErrorResult {
  code: TranslateErrorCode;
  message: string;
}

// AnyErrorInput 兼容多种错误输入形式：字符串、Error 实例、含 errorCode/error 字段的结果对象
export type AnyErrorInput =
  | string
  | Error
  | { errorCode?: string; error?: string }
  | null
  | undefined;

// retryable 错误类别是否值得用户重试（凭据/输入非法不应直接重试）
export function isRetryable(code: string | undefined): boolean {
  switch (code) {
    case ErrorCodes.Network:
    case ErrorCodes.Timeout:
    case ErrorCodes.RateLimit:
    case ErrorCodes.ServiceUnavailable:
    case ErrorCodes.Unknown:
      return true;
    case ErrorCodes.Credentials:
    case ErrorCodes.InvalidInput:
      return false;
    default:
      return true;
  }
}

// shouldShowSettingsButton 是否应在错误提示中展示"前往设置"按钮
export function shouldShowSettingsButton(code: string | undefined): boolean {
  return code === ErrorCodes.Credentials;
}

// normalizeError 从后端 TranslateResult/EngineTranslateResult 提取结构化错误。
// 兼容旧后端只返回字符串错误的情况（code 兜底为 unknown）。
export function normalizeError(result: AnyErrorInput): StructuredErrorResult {
  const code = (result && typeof result === "object" && "errorCode" in result
    ? (result.errorCode as string) || ErrorCodes.Unknown
    : ErrorCodes.Unknown) as TranslateErrorCode;
  const message =
    (result && typeof result === "object" && "error" in result && result.error) ||
    DEFAULT_MESSAGES[code] ||
    DEFAULT_MESSAGES[ErrorCodes.Unknown];
  return { code, message };
}

// formatErrorToToast 格式化错误用于 toast 展示，自动去除 "Error:" 前缀
export function formatErrorToToast(err: AnyErrorInput): string {
  // 兼容旧形式：err 是字符串
  if (typeof err === "string") {
    return String(err || "翻译失败").replace(/^Error:\s*/, "");
  }
  // 兼容异常对象
  if (err instanceof Error) {
    return String(err.message || "翻译失败").replace(/^Error:\s*/, "");
  }
  // 结构化结果对象
  if (err && typeof err === "object") {
    const { message } = normalizeError(err);
    return message;
  }
  return "翻译失败";
}
