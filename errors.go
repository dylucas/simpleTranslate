package main

import (
	"context"
	"errors"
	"strings"
)

// 错误类别：用于前端按类别差异化提示与重试策略。
// 前端通过 TranslateError.Code 字段读取此值。
const (
	// ErrCodeUnknown 未知错误（兜底分类）
	ErrCodeUnknown = "unknown"
	// ErrCodeCredentials 凭据缺失或无效（API Key 未配置 / 鉴权失败）
	ErrCodeCredentials = "credentials"
	// ErrCodeNetwork 网络错误（连接失败、DNS、超时）
	ErrCodeNetwork = "network"
	// ErrCodeTimeout 请求超时（外层 engineTimeout 触发）
	ErrCodeTimeout = "timeout"
	// ErrCodeRateLimit 服务端限流（429）或配额耗尽
	ErrCodeRateLimit = "rate_limit"
	// ErrCodeInvalidInput 输入非法（空文本、目标语言不支持等）
	ErrCodeInvalidInput = "invalid_input"
	// ErrCodeServiceUnavailable 服务端 5xx 不可用
	ErrCodeServiceUnavailable = "service_unavailable"
	// ErrCodeCancelled 请求被用户或应用主动取消
	ErrCodeCancelled = "cancelled"
)

// TranslateError 结构化翻译错误，序列化为 JSON 后供前端按类别处理。
// 兼容旧前端：Error() 返回与原格式一致的字符串。
type TranslateError struct {
	Code    string `json:"code"`             // 错误类别，供前端判断重试策略
	Engine  string `json:"engine,omitempty"` // 出错的引擎（tencent/aliyun）
	Message string `json:"message"`          // 用户可读的错误信息（已去除 Error: 前缀）
	Cause   string `json:"cause,omitempty"`  // 底层错误摘要，便于调试
}

func (e *TranslateError) Error() string {
	if e.Engine != "" {
		return e.Engine + ": " + e.Message
	}
	return e.Message
}

// newTranslateError 构造 TranslateError，自动从 cause 提取原始信息
func newTranslateError(code, engine, message string, cause error) *TranslateError {
	te := &TranslateError{
		Code:    code,
		Engine:  engine,
		Message: message,
	}
	if cause != nil {
		te.Cause = cause.Error()
	}
	return te
}

// classifyError 根据底层错误信息推断错误类别。
// 识别策略：依次匹配关键子串，未命中则返回 ErrCodeUnknown。
func classifyError(engine string, err error) *TranslateError {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return newTranslateError(ErrCodeCancelled, engine, "请求已取消", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newTranslateError(ErrCodeTimeout, engine, "请求超时，请稍后重试", err)
	}
	msg := err.Error()
	lower := strings.ToLower(msg)

	// 凭据相关：缺 AccessKey、鉴权失败、403
	if strings.Contains(lower, "accesskey") ||
		strings.Contains(lower, "api key") ||
		strings.Contains(lower, "secretid") ||
		strings.Contains(lower, "secretkey") ||
		strings.Contains(lower, "http 401") ||
		strings.Contains(lower, "http 403") ||
		strings.Contains(lower, "unauthorized") ||
		strings.Contains(lower, "forbidden") ||
		strings.Contains(lower, "未配置") ||
		strings.Contains(lower, "鉴权") ||
		strings.Contains(lower, "无权限") ||
		strings.Contains(lower, "signaturedoesnotmatch") ||
		strings.Contains(lower, "invalidaccesskey") {
		return newTranslateError(ErrCodeCredentials, engine, "凭据无效或未配置，请在设置中检查", err)
	}

	// 超时
	if strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "超时") ||
		strings.Contains(lower, "deadline exceeded") {
		return newTranslateError(ErrCodeTimeout, engine, "请求超时，请稍后重试", err)
	}

	// 限流：429 / rate limit / 配额
	if strings.Contains(lower, "429") ||
		strings.Contains(lower, "rate limit") ||
		strings.Contains(lower, "ratelimit") ||
		strings.Contains(lower, "配额") ||
		strings.Contains(lower, "throttl") {
		return newTranslateError(ErrCodeRateLimit, engine, "服务限流，请稍后再试", err)
	}

	// 服务不可用：5xx
	if strings.Contains(lower, "500") ||
		strings.Contains(lower, "502") ||
		strings.Contains(lower, "503") ||
		strings.Contains(lower, "504") ||
		strings.Contains(lower, "internal server error") ||
		strings.Contains(lower, "service unavailable") ||
		strings.Contains(lower, "bad gateway") {
		return newTranslateError(ErrCodeServiceUnavailable, engine, "翻译服务暂时不可用", err)
	}

	// 网络：连接拒绝、DNS、EOF
	if strings.Contains(lower, "connection") ||
		strings.Contains(lower, "dial") ||
		strings.Contains(lower, "dns") ||
		strings.Contains(lower, "eof") ||
		strings.Contains(lower, "no such host") ||
		strings.Contains(lower, "network") ||
		strings.Contains(lower, "broken pipe") {
		return newTranslateError(ErrCodeNetwork, engine, "网络连接失败，请检查网络", err)
	}

	// 输入非法：空文本
	if strings.Contains(lower, "空文本") || strings.Contains(lower, "empty text") {
		return newTranslateError(ErrCodeInvalidInput, engine, "输入文本为空", err)
	}

	// 兜底
	return newTranslateError(ErrCodeUnknown, engine, "翻译失败", err)
}

// wrapTranslateError 将底层翻译错误包装为 TranslateError。
// 若 err 已经是 TranslateError 则原样返回（避免重复包装）。
func wrapTranslateError(engine string, err error) *TranslateError {
	if err == nil {
		return nil
	}
	var te *TranslateError
	if errors.As(err, &te) {
		return te
	}
	return classifyError(engine, err)
}
