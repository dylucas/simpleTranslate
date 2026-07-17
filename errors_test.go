package main

import (
	"errors"
	"testing"
)

// TestClassifyError_Credentials 各种凭据相关错误应归为 credentials
func TestClassifyError_Credentials(t *testing.T) {
	cases := []string{
		"未配置阿里云 AccessKey",
		"InvalidAccessKey.NotFound",
		"SignatureDoesNotMatch",
		"missing api key",
		"SecretId not set",
	}
	for _, c := range cases {
		te := classifyError("aliyun", errors.New(c))
		if te.Code != ErrCodeCredentials {
			t.Errorf("classify(%q) = %q, want %q", c, te.Code, ErrCodeCredentials)
		}
	}
}

// TestClassifyError_Timeout 超时关键字识别
func TestClassifyError_Timeout(t *testing.T) {
	cases := []string{
		"context deadline exceeded",
		"read timeout",
		"操作超时",
	}
	for _, c := range cases {
		te := classifyError("tencent", errors.New(c))
		if te.Code != ErrCodeTimeout {
			t.Errorf("classify(%q) = %q, want %q", c, te.Code, ErrCodeTimeout)
		}
	}
}

// TestClassifyError_Network 网络错误识别
func TestClassifyError_Network(t *testing.T) {
	cases := []string{
		"connection refused",
		"dial tcp: no such host",
		"EOF",
		"broken pipe",
	}
	for _, c := range cases {
		te := classifyError("tencent", errors.New(c))
		if te.Code != ErrCodeNetwork {
			t.Errorf("classify(%q) = %q, want %q", c, te.Code, ErrCodeNetwork)
		}
	}
}

// TestClassifyError_RateLimit 限流识别
func TestClassifyError_RateLimit(t *testing.T) {
	cases := []string{
		"HTTP 429: Too Many Requests",
		"rate limit exceeded",
		"throttling",
		"配额已用尽",
	}
	for _, c := range cases {
		te := classifyError("aliyun", errors.New(c))
		if te.Code != ErrCodeRateLimit {
			t.Errorf("classify(%q) = %q, want %q", c, te.Code, ErrCodeRateLimit)
		}
	}
}

// TestClassifyError_ServiceUnavailable 5xx 识别
func TestClassifyError_ServiceUnavailable(t *testing.T) {
	cases := []string{
		"HTTP 500: Internal Server Error",
		"HTTP 502: Bad Gateway",
		"HTTP 503: Service Unavailable",
	}
	for _, c := range cases {
		te := classifyError("aliyun", errors.New(c))
		if te.Code != ErrCodeServiceUnavailable {
			t.Errorf("classify(%q) = %q, want %q", c, te.Code, ErrCodeServiceUnavailable)
		}
	}
}

// TestClassifyError_InvalidInput 输入非法识别
func TestClassifyError_InvalidInput(t *testing.T) {
	te := classifyError("tencent", errors.New("空文本"))
	if te.Code != ErrCodeInvalidInput {
		t.Errorf("classify(空文本) = %q, want %q", te.Code, ErrCodeInvalidInput)
	}
}

// TestClassifyError_Unknown 未匹配的错误兜底为 unknown
func TestClassifyError_Unknown(t *testing.T) {
	te := classifyError("tencent", errors.New("some weird error"))
	if te.Code != ErrCodeUnknown {
		t.Errorf("classify(unknown) = %q, want %q", te.Code, ErrCodeUnknown)
	}
}

// TestClassifyError_NilErr nil error 应返回 nil
func TestClassifyError_NilErr(t *testing.T) {
	if te := classifyError("aliyun", nil); te != nil {
		t.Errorf("classify(nil) 期望 nil，得到 %+v", te)
	}
}

// TestWrapTranslateError_AlreadyTyped 已是 TranslateError 时不重复包装
func TestWrapTranslateError_AlreadyTyped(t *testing.T) {
	orig := newTranslateError(ErrCodeTimeout, "aliyun", "翻译超时", nil)
	got := wrapTranslateError("aliyun", orig)
	if got != orig {
		t.Error("已类型化的错误应原样返回，避免重复包装")
	}
}

// TestWrapTranslateError_NewError 普通错误会被分类包装
func TestWrapTranslateError_NewError(t *testing.T) {
	got := wrapTranslateError("aliyun", errors.New("connection refused"))
	if got.Code != ErrCodeNetwork {
		t.Errorf("期望 network，得到 %q", got.Code)
	}
	if got.Engine != "aliyun" {
		t.Errorf("期望 engine=aliyun，得到 %q", got.Engine)
	}
}

// TestTranslateError_ErrorString Error() 输出格式
func TestTranslateError_ErrorString(t *testing.T) {
	withEngine := &TranslateError{Engine: "aliyun", Message: "凭据无效"}
	if got := withEngine.Error(); got != "aliyun: 凭据无效" {
		t.Errorf("期望 'aliyun: 凭据无效'，得到 %q", got)
	}
	withoutEngine := &TranslateError{Message: "翻译失败"}
	if got := withoutEngine.Error(); got != "翻译失败" {
		t.Errorf("期望 '翻译失败'，得到 %q", got)
	}
}
