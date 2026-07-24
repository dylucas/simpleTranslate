package translate

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"simpleTranslate/config"
)

type baiduDoerFunc func(*http.Request) (*http.Response, error)

func (fn baiduDoerFunc) Do(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestBaiduSignOfficialVectors(t *testing.T) {
	if got := baiduSign("2015063000000001", "apple", "65478", "", "1234567890"); got != "a1a7461d92e5194c5cae3182b5b24de1" {
		t.Fatalf("general sign = %q", got)
	}
	if got := baiduSign("2015063000000001", "amyotrophic lateral sclerosis", "1435660288", "medicine", "12345678"); got != "a649f9a644b25d717beee5ce600b40ae" {
		t.Fatalf("field sign = %q", got)
	}
}

func TestBaiduTranslationSignsRawQueryBeforeFormEncoding(t *testing.T) {
	const query = "a+b & 中文"
	var rawBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		rawBody = string(body)
		form, err := url.ParseQuery(rawBody)
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("q") != query {
			t.Fatalf("decoded q = %q", form.Get("q"))
		}
		wantSign := baiduSign("app", query, "salt", "", "secret")
		if form.Get("sign") != wantSign {
			t.Fatalf("sign = %q, want %q", form.Get("sign"), wantSign)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"from":"en","to":"zh","trans_result":[{"src":"a","dst":"甲"},{"src":"b","dst":"乙"}]}`)
	}))
	defer server.Close()

	client := &baiduClient{
		httpClient:      server.Client(),
		generalEndpoint: server.URL,
		fieldEndpoint:   server.URL,
		detectEndpoint:  server.URL,
		limiter:         newBaiduRequestLimiter(0),
		salt:            func() (string, error) { return "salt", nil },
	}
	result, err := client.translate(context.Background(), query, "en", "zh", "", config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "甲\n乙" || result.From != "en" {
		t.Fatalf("result = %#v", result)
	}
	if !strings.Contains(rawBody, "q=a%2Bb+%26+") {
		t.Fatalf("query was not form encoded after signing: %s", rawBody)
	}
}

func TestBaiduFieldTranslationIncludesDomainInSignature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("domain") != "it" {
			t.Fatalf("domain = %q", r.Form.Get("domain"))
		}
		want := baiduSign("app", "hello", "salt", "it", "secret")
		if r.Form.Get("sign") != want {
			t.Fatalf("field sign = %q, want %q", r.Form.Get("sign"), want)
		}
		_, _ = io.WriteString(w, `{"from":"en","to":"zh","trans_result":[{"src":"hello","dst":"你好"}]}`)
	}))
	defer server.Close()
	client := &baiduClient{
		httpClient: server.Client(), fieldEndpoint: server.URL,
		limiter: newBaiduRequestLimiter(0), salt: func() (string, error) { return "salt", nil },
	}
	result, err := client.translate(context.Background(), "hello", "en", "zh", "it", config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	if err != nil || result.Text != "你好" {
		t.Fatalf("field result = %#v, error = %v", result, err)
	}
}

func TestBaiduDetectTruncatesAtUTF8Boundary(t *testing.T) {
	var sent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		sent = r.Form.Get("q")
		_, _ = io.WriteString(w, `{"error_code":0,"error_msg":"success","data":{"src":"kor"}}`)
	}))
	defer server.Close()
	client := &baiduClient{
		httpClient:     server.Client(),
		detectEndpoint: server.URL,
		limiter:        newBaiduRequestLimiter(0),
		salt:           func() (string, error) { return "salt", nil },
	}
	lang, err := client.detect(context.Background(), strings.Repeat("中", 1000), config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if lang != "kr" {
		t.Fatalf("lang = %q", lang)
	}
	if len([]byte(sent)) > baiduMaxDetectBytes || !utf8.ValidString(sent) {
		t.Fatalf("invalid detection sample: bytes=%d valid=%v", len([]byte(sent)), utf8.ValidString(sent))
	}
}

func TestBaiduLanguageMappings(t *testing.T) {
	for input, want := range map[string]string{"kr": "kor", "fr": "fra", "es": "spa", "ja": "jp", "de": "de"} {
		if got := baiduLanguageCode(input); got != want {
			t.Errorf("baiduLanguageCode(%q) = %q, want %q", input, got, want)
		}
	}
	for input, want := range map[string]string{"kor": "kr", "fra": "fr", "spa": "es", "ja": "jp"} {
		if got := appLanguageCode(input); got != want {
			t.Errorf("appLanguageCode(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBaiduDomainRoutesAndNotice(t *testing.T) {
	for _, tt := range []struct {
		domain, source, target string
		supported              bool
	}{
		{"it", "zh", "en", true},
		{"it", "en", "zh", true},
		{"novel", "zh", "en", true},
		{"novel", "en", "zh", false},
		{"wiki", "en", "zh", false},
		{"finance", "fr", "zh", false},
		{"general", "zh", "en", false},
	} {
		if got := BaiduDomainSupportsRoute(tt.domain, tt.source, tt.target); got != tt.supported {
			t.Errorf("route %#v = %v", tt, got)
		}
		wantNotice := ""
		if tt.domain != "general" && !tt.supported {
			wantNotice = baiduFallbackNotice
		}
		if got := BaiduRouteNotice(tt.domain, tt.source, tt.target); got != wantNotice {
			t.Errorf("notice %#v = %q, want %q", tt, got, wantNotice)
		}
	}
}

func TestBaiduAPIErrorAndQueryValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"error_code":"54003","error_msg":"Access limit"}`)
	}))
	defer server.Close()
	client := &baiduClient{
		httpClient:      server.Client(),
		generalEndpoint: server.URL,
		limiter:         newBaiduRequestLimiter(0),
		salt:            func() (string, error) { return "salt", nil },
	}
	_, err := client.translate(context.Background(), "hello", "en", "zh", "", config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	var apiErr *BaiduAPIError
	if !errors.As(err, &apiErr) || apiErr.Code != "54003" {
		t.Fatalf("error = %v", err)
	}
	_, err = client.translate(context.Background(), strings.Repeat("a", baiduMaxQueryBytes+1), "en", "zh", "", config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	if err == nil || !strings.Contains(err.Error(), "6000") {
		t.Fatalf("oversized query error = %v", err)
	}
}

func TestBaiduTranslationRejectsEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"from":"en","to":"zh","trans_result":[]}`)
	}))
	defer server.Close()
	client := &baiduClient{
		httpClient: server.Client(), generalEndpoint: server.URL,
		limiter: newBaiduRequestLimiter(0), salt: func() (string, error) { return "salt", nil },
	}
	_, err := client.translate(context.Background(), "hello", "en", "zh", "", config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	if err == nil || !strings.Contains(err.Error(), "空译文") {
		t.Fatalf("empty response error = %v", err)
	}
}

func TestBaiduRequestHonorsPreCancelledContext(t *testing.T) {
	called := false
	client := &baiduClient{
		httpClient: baiduDoerFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, errors.New("unexpected request")
		}),
		generalEndpoint: "https://example.invalid", limiter: newBaiduRequestLimiter(0),
		salt: func() (string, error) { return "salt", nil },
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := client.translate(ctx, "hello", "en", "zh", "", config.BaiduConfig{AppID: "app", SecretKey: "secret"})
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("cancelled request error = %v, HTTP called = %v", err, called)
	}
}

func TestBaiduLimiterSpacesRequestsAndHonorsCancellation(t *testing.T) {
	if defaultBaiduClient.limiter.interval != time.Second {
		t.Fatalf("default Baidu interval = %v, want 1s", defaultBaiduClient.limiter.interval)
	}
	limiter := newBaiduRequestLimiter(20 * time.Millisecond)
	if err := limiter.wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if err := limiter.wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(start); elapsed < 15*time.Millisecond {
		t.Fatalf("second request was not delayed: %v", elapsed)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := limiter.wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait error = %v", err)
	}
}

func TestBaiduLiveAPIs(t *testing.T) {
	if os.Getenv("BAIDU_LIVE_TEST") != "1" {
		t.Skip("set BAIDU_LIVE_TEST=1 to run live Baidu API probes")
	}
	service := config.BaiduConfig{
		AppID:     os.Getenv("BAIDU_APP_ID"),
		SecretKey: os.Getenv("BAIDU_SECRET_KEY"),
		Domain:    "it",
	}
	if service.AppID == "" || service.SecretKey == "" {
		t.Fatal("BAIDU_APP_ID and BAIDU_SECRET_KEY are required for live probes")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if _, err := DetectBaiduLanguageWithContext(ctx, "hello", service); err != nil {
		t.Fatal(fmt.Errorf("language detection probe failed: %w", err))
	}
	if _, err := TranslateBaiduGeneralWithContext(ctx, "hello", "en", "zh", service); err != nil {
		t.Fatal(fmt.Errorf("general translation probe failed: %w", err))
	}
	if _, err := TranslateBaiduWithContext(ctx, "hello", "en", "zh", service); err != nil {
		t.Fatal(fmt.Errorf("field translation probe failed: %w", err))
	}
}
