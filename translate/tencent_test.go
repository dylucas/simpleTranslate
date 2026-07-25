package translate

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestChatCompletionContextCancellation(t *testing.T) {
	previousClient := httpClient
	defer func() { httpClient = previousClient }()

	started := make(chan struct{})
	httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		close(started)
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		_, err := chatCompletionWithAPIKeyContext(ctx, "hello", "test-key")
		errCh <- err
	}()
	<-started
	cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("chatCompletion cancellation error = %v, want context.Canceled", err)
	}
}

// TestNormalizeLangCode 验证模型返回的各种语种代码/名称归一化
func TestNormalizeLangCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ja", "jp"},
		{"JA", "jp"},
		{"ko", "kr"},
		{"zh-cn", "zh"},
		{"en-us", "en"},
		{"english", "en"},
		{"中文", "zh"},
		{"汉语", "zh"},
		{"英语", "en"},
		{"`zh`", "zh"},
		{"\"en\"", "en"},
		{"fr", "fr"},
		// 未知代码原样返回（小写、去标点后）
		{"xx", "xx"},
	}
	for _, c := range cases {
		got := normalizeLangCode(c.in)
		if got != c.want {
			t.Errorf("normalizeLangCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestLangCodeToName 验证内部代码到中文名的映射覆盖
func TestLangCodeToName(t *testing.T) {
	wantKeys := []string{"zh", "en", "jp", "ja", "kr", "ko", "fr", "de", "ru", "es"}
	for _, k := range wantKeys {
		if _, ok := langCodeToName[k]; !ok {
			t.Errorf("langCodeToName 缺少键 %q", k)
		}
	}
	if langCodeToName["zh"] != "中文" {
		t.Errorf("langCodeToName[zh] 期望 中文，得到 %q", langCodeToName["zh"])
	}
}

// TestNameToLangCode_Reverse 验证中文名到代码映射与正向一致
func TestNameToLangCode_Reverse(t *testing.T) {
	cases := map[string]string{
		"中文":   "zh",
		"英语":   "en",
		"日语":   "jp",
		"韩语":   "kr",
		"法语":   "fr",
		"德语":   "de",
		"俄语":   "ru",
		"西班牙语": "es",
	}
	for name, code := range cases {
		got, ok := nameToLangCode[name]
		if !ok {
			t.Errorf("nameToLangCode 缺少 %q", name)
			continue
		}
		if got != code {
			t.Errorf("nameToLangCode[%q] = %q, want %q", name, got, code)
		}
	}
}

// TestDetectLanguage_EmptyText 空文本应返回错误
func TestDetectLanguage_EmptyText(t *testing.T) {
	_, err := DetectLanguage("")
	if err == nil {
		t.Error("空文本应返回错误，得到 nil")
	}
}

// TestDetectLanguage_Whitespace 仅空白应返回错误
func TestDetectLanguage_Whitespace(t *testing.T) {
	_, err := DetectLanguage("   \n\t  ")
	if err == nil {
		t.Error("纯空白文本应返回错误，得到 nil")
	}
}

func TestDetectLanguageRejectsUnsupportedResponse(t *testing.T) {
	_, err := detectLanguage("hello", func(string) (string, error) {
		return "xx", nil
	})
	if err == nil {
		t.Fatal("不支持的识别结果应返回错误")
	}
}

func TestDetectLanguageIgnoresURLs(t *testing.T) {
	const (
		text = "To develop in the browser and call your bound Go methods from Javascript, navigate to: http://localhost:34115"
		url  = "http://localhost:34115"
	)

	var prompt string
	got, err := detectLanguage(text, func(value string) (string, error) {
		prompt = value
		return "en", nil
	})
	if err != nil {
		t.Fatalf("detectLanguage returned error: %v", err)
	}
	if got != "en" {
		t.Fatalf("detectLanguage returned %q, want en", got)
	}
	if strings.Contains(prompt, url) {
		t.Fatalf("language detection prompt should omit URL %q: %s", url, prompt)
	}
	if !strings.Contains(prompt, "To develop in the browser") {
		t.Fatalf("language detection prompt lost natural-language text: %s", prompt)
	}
}

func TestDetectLanguageKeepsURLOnlyInputValid(t *testing.T) {
	const url = "https://localhost:34115"
	var prompt string
	if _, err := detectLanguage(url, func(value string) (string, error) {
		prompt = value
		return "en", nil
	}); err != nil {
		t.Fatalf("URL-only input should still reach the detector: %v", err)
	}
	if !strings.Contains(prompt, url) {
		t.Fatalf("URL-only input should not produce an empty detector prompt: %s", prompt)
	}
}

func TestValidateDetectedLanguageRejectsEmptyResponse(t *testing.T) {
	if _, err := validateDetectedLanguage(""); err == nil {
		t.Fatal("空识别结果应返回错误")
	}
}

func TestTranslateTextRejectsEmptyResponse(t *testing.T) {
	_, err := translateText("hello", "en", "zh", func(string) (string, error) {
		return "   ", nil
	})
	if err == nil {
		t.Fatal("空译文应返回错误")
	}
}
