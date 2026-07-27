package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"simpleTranslate/config"
	"simpleTranslate/translate"
)

func TestCancelTranslationCancelsRegisteredRequest(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	ctx, finish := app.beginTranslation("cancel-me")
	done := make(chan struct{})
	go func() {
		<-ctx.Done()
		close(done)
	}()

	if !app.CancelTranslation("cancel-me") {
		t.Fatal("CancelTranslation should find the active request")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("request context was not cancelled")
	}
	finish()
	if app.CancelTranslation("cancel-me") {
		t.Fatal("a finished request should no longer be cancellable")
	}
}

func TestCancelTranslationRejectsOversizedRequestID(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	if app.CancelTranslation(strings.Repeat("x", maxRequestIDBytes+1)) {
		t.Fatal("oversized request ID should not be cancellable")
	}
}

func TestTranslateTextCancellationSkipsCache(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	started := make(chan struct{})
	app.translateInvoke = func(ctx context.Context, _, _, _, _ string) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}
	resultCh := make(chan TranslateResult, 1)
	go func() {
		resultCh <- app.TranslateText(TranslateRequest{RequestID: "cancel-text", Text: "hello", Source: "en", Target: "zh", Engine: "tencent"})
	}()
	<-started
	if !app.CancelTranslation("cancel-text") {
		t.Fatal("expected active translation to be cancellable")
	}
	result := <-resultCh
	if result.ErrorCode != ErrCodeCancelled {
		t.Fatalf("ErrorCode = %q, want %q", result.ErrorCode, ErrCodeCancelled)
	}
	if app.translateCache.len() != 0 {
		t.Fatalf("cancelled translation should not populate cache, got %d entries", app.translateCache.len())
	}
}

func TestTranslateMultiCancellationSkipsEventsAndCache(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	started := make(chan struct{})
	app.translateInvoke = func(ctx context.Context, _, _, _, _ string) (string, error) {
		close(started)
		<-ctx.Done()
		return "", ctx.Err()
	}
	emitted := 0
	app.eventEmit = func(EngineTranslateResult) { emitted++ }
	resultCh := make(chan MultiTranslateResult, 1)
	go func() {
		resultCh <- app.TranslateMulti(MultiTranslateRequest{RequestID: "cancel-multi", Text: "hello", Source: "en", Target: "zh", Engines: []string{"tencent"}})
	}()
	<-started
	if !app.CancelTranslation("cancel-multi") {
		t.Fatal("expected active multi translation to be cancellable")
	}
	result := <-resultCh
	engineResult := result.Results["tencent"]
	if engineResult.ErrorCode != ErrCodeCancelled {
		t.Fatalf("engine ErrorCode = %q, want %q", engineResult.ErrorCode, ErrCodeCancelled)
	}
	if emitted != 0 {
		t.Fatalf("cancelled multi translation emitted %d events", emitted)
	}
	if app.translateCache.len() != 0 {
		t.Fatalf("cancelled multi translation should not populate cache, got %d entries", app.translateCache.len())
	}
}

func TestTranslateMultiRejectsLateSuccessAfterCancellation(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	started := make(chan struct{})
	release := make(chan struct{})
	app.translateInvoke = func(context.Context, string, string, string, string) (string, error) {
		close(started)
		<-release
		return "late success", nil
	}
	emitted := 0
	app.eventEmit = func(EngineTranslateResult) { emitted++ }
	resultCh := make(chan MultiTranslateResult, 1)
	go func() {
		resultCh <- app.TranslateMulti(MultiTranslateRequest{
			RequestID: "late-success",
			Text:      "hello",
			Source:    "en",
			Target:    "zh",
			Engines:   []string{"tencent"},
		})
	}()
	<-started
	if !app.CancelTranslation("late-success") {
		t.Fatal("expected active translation to be cancellable")
	}
	close(release)

	result := <-resultCh
	if got := result.Results["tencent"].ErrorCode; got != ErrCodeCancelled {
		t.Fatalf("ErrorCode = %q, want %q", got, ErrCodeCancelled)
	}
	if emitted != 0 {
		t.Fatalf("cancelled late success emitted %d events", emitted)
	}
	if app.translateCache.len() != 0 {
		t.Fatalf("cancelled late success populated %d cache entries", app.translateCache.len())
	}
}

// TestNormalizeEngines 验证引擎列表归一化：去重、小写、过滤非法值、空列表兜底
func TestNormalizeEngines(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"空列表兜底", []string{}, []string{"tencent", "aliyun", "baidu"}},
		{"nil 兜底", nil, []string{"tencent", "aliyun", "baidu"}},
		{"单引擎", []string{"tencent"}, []string{"tencent"}},
		{"大小写归一", []string{"TENCENT", "Aliyun"}, []string{"tencent", "aliyun"}},
		{"去重", []string{"tencent", "tencent", "aliyun"}, []string{"tencent", "aliyun"}},
		{"过滤非法值", []string{"tencent", "deepseek", "", "  ", "aliyun"}, []string{"tencent", "aliyun"}},
		{"百度引擎", []string{"baidu", "tencent"}, []string{"baidu", "tencent"}},
		{"全非法值兜底", []string{"xxx", "yyy"}, []string{"tencent", "aliyun", "baidu"}},
		{"含空白字符", []string{"  tencent  ", "aliyun"}, []string{"tencent", "aliyun"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := normalizeEngines(c.in)
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("normalizeEngines(%v) = %v, want %v", c.in, got, c.want)
			}
		})
	}
}

// TestNormalizeEngines_PreservesOrder 验证顺序保留（首次出现的顺序）
func TestNormalizeEngines_PreservesOrder(t *testing.T) {
	got := normalizeEngines([]string{"aliyun", "tencent"})
	want := []string{"aliyun", "tencent"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("顺序应保留首次出现: got=%v want=%v", got, want)
	}
}

// TestFallbackTarget 验证源/目标相同时的目标语言兜底逻辑
func TestFallbackTarget(t *testing.T) {
	cases := []struct {
		name   string
		src    string
		target string
		want   string
	}{
		{"zh 互换为 en", "zh", "zh", "en"},
		{"en 互换为 zh", "en", "en", "zh"},
		{"其他语言兜底为 en", "fr", "fr", "en"},
		{"源目标不同则原样返回", "zh", "en", "en"},
		{"空目标保持空", "zh", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := fallbackTarget(c.src, c.target)
			if got != c.want {
				t.Errorf("fallbackTarget(%q, %q) = %q, want %q", c.src, c.target, got, c.want)
			}
		})
	}
}

func TestTranslateTextRejectsInvalidInput(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())

	for _, tt := range []struct {
		name   string
		text   string
		engine string
	}{
		{name: "empty text", text: "  \n\t", engine: "tencent"},
		{name: "unknown engine", text: "hello", engine: "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := app.TranslateText(TranslateRequest{RequestID: "invalid", Text: tt.text, Source: "en", Target: "zh", Engine: tt.engine})
			if res.ErrorCode != ErrCodeInvalidInput {
				t.Fatalf("ErrorCode = %q, want %q", res.ErrorCode, ErrCodeInvalidInput)
			}
		})
	}
}

func TestTranslateTextRejectsOversizedMetadataBeforeTranslation(t *testing.T) {
	oversizedRequestID := strings.Repeat("r", maxRequestIDBytes+1)
	for _, tt := range []struct {
		name   string
		mutate func(*TranslateRequest)
	}{
		{name: "request ID", mutate: func(req *TranslateRequest) { req.RequestID = oversizedRequestID }},
		{name: "engine", mutate: func(req *TranslateRequest) { req.Engine = strings.Repeat("e", maxEngineIDBytes+1) }},
		{name: "source", mutate: func(req *TranslateRequest) { req.Source = strings.Repeat("s", maxLanguageCodeBytes+1) }},
		{name: "target", mutate: func(req *TranslateRequest) { req.Target = strings.Repeat("t", maxLanguageCodeBytes+1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewAppWithDataDir(t.TempDir())
			calls := 0
			app.translateInvoke = func(context.Context, string, string, string, string) (string, error) {
				calls++
				return "unexpected", nil
			}
			req := TranslateRequest{RequestID: "bounded", Text: "hello", Source: "en", Target: "zh", Engine: "tencent"}
			tt.mutate(&req)

			result := app.TranslateText(req)
			if result.ErrorCode != ErrCodeInvalidInput {
				t.Fatalf("ErrorCode = %q, want %q", result.ErrorCode, ErrCodeInvalidInput)
			}
			if calls != 0 {
				t.Fatalf("translation calls = %d, want 0", calls)
			}
			if req.RequestID == oversizedRequestID && result.RequestID != "" {
				t.Fatalf("oversized request ID was reflected: length %d", len(result.RequestID))
			}
		})
	}
}

func TestTranslateTextValidatesAndNormalizesLanguages(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	for _, tt := range []struct {
		name   string
		source string
		target string
	}{
		{name: "missing target", source: "en", target: ""},
		{name: "invalid source", source: "unknown", target: "zh"},
		{name: "invalid target", source: "en", target: "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := app.TranslateText(TranslateRequest{
				RequestID: "invalid-language",
				Text:      "hello",
				Source:    tt.source,
				Target:    tt.target,
				Engine:    "tencent",
			})
			if res.ErrorCode != ErrCodeInvalidInput {
				t.Fatalf("ErrorCode = %q, want %q", res.ErrorCode, ErrCodeInvalidInput)
			}
		})
	}

	if got := normalizeLanguageCode(" JA "); got != "jp" {
		t.Fatalf("normalizeLanguageCode(JA) = %q, want jp", got)
	}
	if got := normalizeLanguageCode("KO"); got != "kr" {
		t.Fatalf("normalizeLanguageCode(KO) = %q, want kr", got)
	}
}

func TestTranslateMultiRejectsEmptyInput(t *testing.T) {
	res := NewAppWithDataDir(t.TempDir()).TranslateMulti(MultiTranslateRequest{RequestID: "multi-empty", Text: "  ", Source: "en", Target: "zh", Engines: []string{"tencent", "aliyun"}})
	if res.RequestID != "multi-empty" {
		t.Fatalf("request ID = %q, want multi-empty", res.RequestID)
	}
	if len(res.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(res.Results))
	}
	for engine, result := range res.Results {
		if result.ErrorCode != ErrCodeInvalidInput {
			t.Fatalf("%s ErrorCode = %q, want %q", engine, result.ErrorCode, ErrCodeInvalidInput)
		}
	}
}

func TestTranslateMultiRejectsInvalidLanguageRoute(t *testing.T) {
	res := NewAppWithDataDir(t.TempDir()).TranslateMulti(MultiTranslateRequest{
		RequestID: "multi-invalid-language",
		Text:      "hello",
		Source:    "en",
		Target:    "",
		Engines:   []string{"tencent"},
	})
	result := res.Results["tencent"]
	if result.ErrorCode != ErrCodeInvalidInput {
		t.Fatalf("ErrorCode = %q, want %q", result.ErrorCode, ErrCodeInvalidInput)
	}
}

func TestTranslateMultiRejectsOversizedMetadataBeforeTranslation(t *testing.T) {
	oversizedRequestID := strings.Repeat("r", maxRequestIDBytes+1)
	for _, tt := range []struct {
		name   string
		mutate func(*MultiTranslateRequest)
	}{
		{name: "request ID", mutate: func(req *MultiTranslateRequest) { req.RequestID = oversizedRequestID }},
		{name: "source", mutate: func(req *MultiTranslateRequest) { req.Source = strings.Repeat("s", maxLanguageCodeBytes+1) }},
		{name: "target", mutate: func(req *MultiTranslateRequest) { req.Target = strings.Repeat("t", maxLanguageCodeBytes+1) }},
		{name: "engine count", mutate: func(req *MultiTranslateRequest) { req.Engines = make([]string, maxRequestedEngines+1) }},
		{name: "engine name", mutate: func(req *MultiTranslateRequest) { req.Engines = []string{strings.Repeat("e", maxEngineIDBytes+1)} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewAppWithDataDir(t.TempDir())
			calls := 0
			app.translateInvoke = func(context.Context, string, string, string, string) (string, error) {
				calls++
				return "unexpected", nil
			}
			req := MultiTranslateRequest{RequestID: "bounded", Text: "hello", Source: "en", Target: "zh", Engines: []string{"tencent"}}
			tt.mutate(&req)

			result := app.TranslateMulti(req)
			if len(result.Results) == 0 {
				t.Fatal("oversized metadata should return structured engine errors")
			}
			for engine, engineResult := range result.Results {
				if engineResult.ErrorCode != ErrCodeInvalidInput {
					t.Fatalf("%s ErrorCode = %q, want %q", engine, engineResult.ErrorCode, ErrCodeInvalidInput)
				}
				if len(engineResult.RequestID) > maxRequestIDBytes {
					t.Fatalf("%s reflected oversized request ID: length %d", engine, len(engineResult.RequestID))
				}
			}
			if calls != 0 {
				t.Fatalf("translation calls = %d, want 0", calls)
			}
			if req.RequestID == oversizedRequestID && result.RequestID != "" {
				t.Fatalf("oversized request ID was reflected: length %d", len(result.RequestID))
			}
		})
	}
}

func TestTranslateMultiEmitsCachedEngineResults(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	app.translateCache.set(translationResultCacheKey(0, "tencent", "", "en", "zh", "hello"), "cached result")
	var emitted []EngineTranslateResult
	app.eventEmit = func(result EngineTranslateResult) {
		emitted = append(emitted, result)
	}

	result := app.TranslateMulti(MultiTranslateRequest{
		RequestID: "cached-stream",
		Text:      "hello",
		Source:    "en",
		Target:    "zh",
		Engines:   []string{"tencent"},
	})

	if result.Results["tencent"].Text != "cached result" {
		t.Fatalf("cached result = %q, want cached result", result.Results["tencent"].Text)
	}
	if len(emitted) != 1 || emitted[0].Text != "cached result" {
		t.Fatalf("cached engine should emit one streaming result, got %#v", emitted)
	}
}

func TestBaiduDomainCacheIsolationAndFallbackNotice(t *testing.T) {
	general := translationResultCacheKey(0, "baidu", "general", "fr", "zh", "bonjour")
	field := translationResultCacheKey(0, "baidu", "it", "fr", "zh", "bonjour")
	if general == field {
		t.Fatal("Baidu general and field modes must not share cache keys")
	}

	config.InvalidateCache()
	app := NewAppWithDataDir(t.TempDir())
	cfg := config.DefaultCloudConfig()
	cfg.Baidu = config.BaiduConfig{AppID: "draft", SecretKey: "draft", Domain: "it"}
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	app.translateInvoke = func(_ context.Context, engine, _, _, _ string) (string, error) {
		if engine != "baidu" {
			t.Fatalf("engine = %q", engine)
		}
		return "你好", nil
	}
	result := app.TranslateText(TranslateRequest{
		RequestID: "baidu-fallback", Text: "bonjour", Source: "fr", Target: "zh", Engine: "baidu",
	})
	if result.Text != "你好" || result.Notice == "" {
		t.Fatalf("fallback result = %+v", result)
	}
}

func TestConfigSaveIsolatesLateTranslationCacheWrite(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	app.translateInvoke = func(context.Context, string, string, string, string) (string, error) {
		calls++
		if calls == 1 {
			close(started)
			<-release
			return "old credentials result", nil
		}
		return "new credentials result", nil
	}

	firstResult := make(chan TranslateResult, 1)
	go func() {
		firstResult <- app.TranslateText(TranslateRequest{
			RequestID: "before-save", Text: "hello", Source: "en", Target: "zh", Engine: "tencent",
		})
	}()
	<-started

	cfg := config.DefaultCloudConfig()
	cfg.Tencent.SecretKey = "new-key"
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	close(release)
	if got := (<-firstResult).Text; got != "old credentials result" {
		t.Fatalf("in-flight result = %q, want old credentials result", got)
	}

	second := app.TranslateText(TranslateRequest{
		RequestID: "after-save", Text: "hello", Source: "en", Target: "zh", Engine: "tencent",
	})
	if second.Text != "new credentials result" {
		t.Fatalf("post-save result = %q, want new credentials result", second.Text)
	}
	if calls != 2 {
		t.Fatalf("translation calls = %d, want 2", calls)
	}
}

func TestSaveConfigInvalidatesChangedAliyunClient(t *testing.T) {
	service := config.ServiceConfig{
		SecretId:  "access-id",
		SecretKey: "access-key",
		Region:    "cn-hangzhou",
	}
	translate.InvalidateAliyunClientIfConfigChanged(config.ServiceConfig{})
	defer translate.InvalidateAliyunClientIfConfigChanged(config.ServiceConfig{})

	initial, err := translate.CreateClientWithConfig(service)
	if err != nil {
		t.Fatal(err)
	}
	app := NewAppWithDataDir(t.TempDir())
	cfg := config.DefaultCloudConfig()
	cfg.Aliyun = service
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	retained, err := translate.CreateClientWithConfig(service)
	if err != nil {
		t.Fatal(err)
	}
	if retained != initial {
		t.Fatal("equivalent saved config should preserve the Aliyun client")
	}

	cfg.Aliyun.SecretKey = "replacement-key"
	if err := app.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	rebuilt, err := translate.CreateClientWithConfig(service)
	if err != nil {
		t.Fatal(err)
	}
	if rebuilt == initial {
		t.Fatal("changed Aliyun credentials should release the cached client")
	}
}

// TestSaveLoadHistory_RoundTrip 保存后读取应一致
func TestSaveLoadHistory_RoundTrip(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	path := app.getHistoryPath()
	_ = os.Remove(path)
	defer os.Remove(path)

	entries := []HistoryEntry{
		{ID: 1, Input: "hello", Output: "你好", Source: "en", Target: "zh", Time: "10:00"},
		{ID: 2, Input: "world", Output: "世界", Source: "en", Target: "zh", Time: "10:01"},
	}

	if err := app.saveHistory(entries); err != nil {
		t.Fatalf("SaveHistory 失败: %v", err)
	}

	got, err := app.loadHistory()
	if err != nil {
		t.Fatalf("LoadHistory 失败: %v", err)
	}
	if !reflect.DeepEqual(got, entries) {
		t.Errorf("读取与写入不一致\nwant=%+v\ngot =%+v", entries, got)
	}
}

// TestLoadHistory_MissingFile 文件不存在时返回空列表无错误
func TestLoadHistory_MissingFile(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	path := app.getHistoryPath()
	_ = os.Remove(path)
	defer os.Remove(path)

	got, err := app.loadHistory()
	if err != nil {
		t.Fatalf("期望 nil 错误，得到 %v", err)
	}
	if len(got) != 0 {
		t.Errorf("期望空列表，得到 %d 条", len(got))
	}
}

// TestSaveHistory_TruncatesTo200 验证历史记录上限 200 条
func TestSaveHistory_TruncatesTo200(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	path := app.getHistoryPath()
	_ = os.Remove(path)
	defer os.Remove(path)

	// 构造 250 条
	entries := make([]HistoryEntry, 250)
	for i := range entries {
		entries[i] = HistoryEntry{ID: int64(i), Input: "text", Output: "out", Source: "en", Target: "zh", Time: "10:00"}
	}

	if err := app.saveHistory(entries); err != nil {
		t.Fatalf("SaveHistory 失败: %v", err)
	}

	got, err := app.loadHistory()
	if err != nil {
		t.Fatalf("LoadHistory 失败: %v", err)
	}
	if len(got) != 200 {
		t.Errorf("期望截断为 200 条，得到 %d 条", len(got))
	}
}

// TestSaveHistory_EmptyList 空列表可正常保存与读取
func TestSaveHistory_EmptyList(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	path := app.getHistoryPath()
	_ = os.Remove(path)
	defer os.Remove(path)

	if err := app.saveHistory([]HistoryEntry{}); err != nil {
		t.Fatalf("SaveHistory 失败: %v", err)
	}

	got, err := app.loadHistory()
	if err != nil {
		t.Fatalf("LoadHistory 失败: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("期望空列表，得到 %d 条", len(got))
	}
}

// TestLoadHistory_InvalidJSON 损坏的 JSON 文件应返回错误
func TestLoadHistory_InvalidJSON(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	path := app.getHistoryPath()
	_ = os.Remove(path)
	defer os.Remove(path)

	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	_, err := app.loadHistory()
	if err == nil {
		t.Error("损坏 JSON 期望返回错误，得到 nil")
	}
}

func TestLoadHistoryRejectsOversizedFile(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	path := app.getHistoryPath()
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, maxHistoryFileBytes+1); err != nil {
		t.Fatal(err)
	}

	if _, err := app.loadHistory(); err == nil || !strings.Contains(err.Error(), "超过") {
		t.Fatalf("oversized history error = %v", err)
	}
}

func TestLoadHistory_NormalizesNullAndOversizedFiles(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		app := NewAppWithDataDir(t.TempDir())
		if err := os.WriteFile(app.getHistoryPath(), []byte("null"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := app.loadHistory()
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("null history should become a non-nil empty slice, got %#v", got)
		}
	})

	t.Run("oversized", func(t *testing.T) {
		app := NewAppWithDataDir(t.TempDir())
		entries := make([]HistoryEntry, 201)
		for i := range entries {
			entries[i].ID = int64(i)
		}
		data, err := json.Marshal(entries)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(app.getHistoryPath(), data, 0600); err != nil {
			t.Fatal(err)
		}
		got, err := app.loadHistory()
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 200 {
			t.Fatalf("oversized history should be capped at 200 entries, got %d", len(got))
		}
	})
}

func TestLoadHistoryValidatesEntriesBeyondRetentionLimit(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	var data strings.Builder
	data.WriteByte('[')
	for i := 0; i < historyEntryLimit; i++ {
		if i > 0 {
			data.WriteByte(',')
		}
		data.WriteString(`{"input":"valid"}`)
	}
	data.WriteString(`,{"input":]`)
	if err := os.WriteFile(app.getHistoryPath(), []byte(data.String()), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := app.loadHistory(); err == nil {
		t.Fatal("malformed entry beyond retention limit should be rejected")
	}
}

// TestGetHistoryPath_CreatesDir 历史路径所在目录会被自动创建
func TestGetHistoryPath_CreatesDir(t *testing.T) {
	app := NewAppWithDataDir(filepath.Join(t.TempDir(), "nested"))
	path := app.getHistoryPath()
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("期望目录已创建: %s", dir)
	}
}

func TestQueryAppendClearHistory(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	for i := 0; i < 25; i++ {
		entry := HistoryEntry{ID: int64(i), Input: fmt.Sprintf("input-%02d", i), Output: fmt.Sprintf("output-%02d", i), Source: "en", Target: "zh", Time: "now"}
		added, err := app.AppendHistory(entry)
		if err != nil || !added {
			t.Fatalf("append %d: added=%v err=%v", i, added, err)
		}
	}
	page, err := app.QueryHistory(HistoryQuery{Offset: 0, Limit: 10})
	if err != nil || len(page.Entries) != 10 || page.Total != 25 || page.AllTotal != 25 || !page.HasMore {
		t.Fatalf("first page = %+v err=%v", page, err)
	}
	search, err := app.QueryHistory(HistoryQuery{Query: "INPUT-02", Limit: 10})
	if err != nil || search.Total != 1 || search.AllTotal != 25 || search.Entries[0].Input != "input-02" {
		t.Fatalf("search = %+v err=%v", search, err)
	}
	added, err := app.AppendHistory(page.Entries[0])
	if err != nil || added {
		t.Fatalf("latest duplicate: added=%v err=%v", added, err)
	}
	if err := app.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	empty, err := app.QueryHistory(HistoryQuery{})
	if err != nil || empty.Total != 0 || empty.AllTotal != 0 || empty.Entries == nil {
		t.Fatalf("cleared page = %+v err=%v", empty, err)
	}
}

func TestAppendHistoryRejectsOversizedOutput(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	added, err := app.AppendHistory(HistoryEntry{
		Input:  "hello",
		Output: strings.Repeat("x", maxHistoryOutputBytes+1),
		Source: "en",
		Target: "zh",
	})
	if err == nil || added {
		t.Fatalf("oversized output: added=%v err=%v", added, err)
	}
	page, queryErr := app.QueryHistory(HistoryQuery{})
	if queryErr != nil || page.AllTotal != 0 {
		t.Fatalf("history changed after rejected append: page=%+v err=%v", page, queryErr)
	}
}

func TestAppendHistoryRejectsOversizedMetadata(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*HistoryEntry)
	}{
		{name: "source", mutate: func(entry *HistoryEntry) { entry.Source = strings.Repeat("x", maxHistoryLanguageBytes+1) }},
		{name: "target", mutate: func(entry *HistoryEntry) { entry.Target = strings.Repeat("x", maxHistoryLanguageBytes+1) }},
		{name: "time", mutate: func(entry *HistoryEntry) { entry.Time = strings.Repeat("x", maxHistoryTimeBytes+1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewAppWithDataDir(t.TempDir())
			entry := HistoryEntry{Input: "hello", Output: "output", Source: "en", Target: "zh", Time: "now"}
			tt.mutate(&entry)
			added, err := app.AppendHistory(entry)
			if err == nil || added {
				t.Fatalf("oversized %s: added=%v err=%v", tt.name, added, err)
			}
			page, queryErr := app.QueryHistory(HistoryQuery{})
			if queryErr != nil || page.AllTotal != 0 {
				t.Fatalf("history changed after rejected append: page=%+v err=%v", page, queryErr)
			}
		})
	}
}

func TestQueryHistoryRejectsOversizedSearchBeforeReadingFile(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	if err := os.WriteFile(app.getHistoryPath(), []byte("invalid history"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := app.QueryHistory(HistoryQuery{Query: strings.Repeat("x", maxHistoryQueryBytes+1)})
	if err == nil || !strings.Contains(err.Error(), "搜索词") {
		t.Fatalf("oversized history query error = %v", err)
	}
}

func TestQueryHistoryHandlesExtremePaginationValues(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	for i := 0; i < 3; i++ {
		if _, err := app.AppendHistory(HistoryEntry{ID: int64(i), Input: fmt.Sprintf("input-%d", i), Output: "output"}); err != nil {
			t.Fatal(err)
		}
	}

	page, err := app.QueryHistory(HistoryQuery{Offset: int(^uint(0) >> 1), Limit: maxHistoryPageLimit})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 0 || page.Total != 3 || page.HasMore {
		t.Fatalf("extreme offset page = %+v", page)
	}

	page, err = app.QueryHistory(HistoryQuery{Offset: -1, Limit: int(^uint(0) >> 1)})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Entries) != 3 || page.Total != 3 || page.HasMore {
		t.Fatalf("normalized bounds page = %+v", page)
	}
}

func TestAppendHistoryConcurrentKeepsLimit(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 240; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, err := app.AppendHistory(HistoryEntry{ID: int64(id), Input: fmt.Sprintf("input-%03d", id), Output: "output", Source: "en", Target: "zh"})
			if err != nil {
				t.Errorf("append %d: %v", id, err)
			}
		}(i)
	}
	wg.Wait()
	page, err := app.QueryHistory(HistoryQuery{Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 200 || len(page.Entries) != 50 || !page.HasMore {
		t.Fatalf("concurrent page = %+v", page)
	}
}

func TestExportHistoryCancelAndSuccess(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	if _, err := app.AppendHistory(HistoryEntry{ID: 1, Input: "hello", Output: "你好", Source: "en", Target: "zh"}); err != nil {
		t.Fatal(err)
	}
	app.saveFileDialog = func(context.Context, runtime.SaveDialogOptions) (string, error) { return "", nil }
	if exported, err := app.ExportHistory(); err != nil || exported {
		t.Fatalf("cancel: exported=%v err=%v", exported, err)
	}
	path := filepath.Join(t.TempDir(), "export.json")
	app.saveFileDialog = func(context.Context, runtime.SaveDialogOptions) (string, error) { return path, nil }
	if exported, err := app.ExportHistory(); err != nil || !exported {
		t.Fatalf("success: exported=%v err=%v", exported, err)
	}
	if data, err := os.ReadFile(path); err != nil || !bytes.Contains(data, []byte("hello")) {
		t.Fatalf("export data=%q err=%v", data, err)
	}
}

func TestTranslateRejectsOversizedInput(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	input := strings.Repeat("a", maxInputBytes+1)
	result := app.TranslateText(TranslateRequest{Text: input, Source: "en", Target: "zh", Engine: "tencent"})
	if result.ErrorCode != ErrCodeInvalidInput {
		t.Fatalf("single error code = %q", result.ErrorCode)
	}
	multi := app.TranslateMulti(MultiTranslateRequest{Text: input, Source: "en", Target: "zh", Engines: []string{"tencent"}})
	if multi.Results["tencent"].ErrorCode != ErrCodeInvalidInput {
		t.Fatalf("multi result = %+v", multi.Results)
	}
}

// TestTranslateText_MissingCreds 未配置凭据时应返回错误而非 panic
// 通过在临时配置路径下写入空配置来模拟
func TestTranslateText_MissingCreds(t *testing.T) {
	config.InvalidateCache()
	app := NewAppWithDataDir(t.TempDir())
	path := app.getConfigPath()

	if err := config.SaveConfig(path, config.CloudConfig{}); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	config.InvalidateCache()

	res := app.TranslateText(TranslateRequest{RequestID: "aliyun-missing", Text: "hello", Source: "en", Target: "zh", Engine: "aliyun"})
	if res.ErrorCode == "" {
		t.Error("未配置阿里云凭据期望返回结构化错误码")
	}
	if res.ErrorCode != "credentials" {
		t.Errorf("期望错误码 credentials，得到 %q", res.ErrorCode)
	}
	res = app.TranslateText(TranslateRequest{RequestID: "tencent-missing", Text: "hello", Source: "en", Target: "zh", Engine: "tencent"})
	if res.ErrorCode == "" {
		t.Error("未配置混元凭据期望返回结构化错误码")
	}
	if res.ErrorCode != "credentials" {
		t.Errorf("期望错误码 credentials，得到 %q", res.ErrorCode)
	}
	res = app.TranslateText(TranslateRequest{RequestID: "baidu-missing", Text: "hello", Source: "en", Target: "zh", Engine: "baidu"})
	if res.ErrorCode != ErrCodeCredentials {
		t.Errorf("百度缺少凭据错误码 = %q, want credentials", res.ErrorCode)
	}
}

func TestConnectionDraftDoesNotPersist(t *testing.T) {
	config.InvalidateCache()
	app := NewAppWithDataDir(t.TempDir())
	persisted := config.DefaultCloudConfig()
	persisted.Tencent.SecretKey = "persisted-key"
	if err := app.SaveConfig(persisted); err != nil {
		t.Fatal(err)
	}

	if err := app.TestConnection("tencent", ServiceConfig{}); err == nil {
		t.Fatal("empty draft credentials should fail connection validation")
	}
	got, err := app.GetConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got.Tencent.SecretKey != "persisted-key" {
		t.Fatalf("connection test mutated persisted config: %q", got.Tencent.SecretKey)
	}
}

func TestConnectionUsesBoundedContext(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	probeErr := errors.New("probe failed")
	app.connectionTestInvoke = func(ctx context.Context, engine string, _ ServiceConfig) error {
		if engine != "aliyun" {
			t.Fatalf("engine = %q, want aliyun", engine)
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("connection test context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > engineTimeout {
			t.Fatalf("connection test deadline remaining = %v, want (0, %v]", remaining, engineTimeout)
		}
		return probeErr
	}

	err := app.TestConnection(" Aliyun ", ServiceConfig{})
	if !errors.Is(err, probeErr) {
		t.Fatalf("TestConnection error = %v, want wrapped probe error", err)
	}
}

func TestConnectionRejectsOversizedEngineBeforeProbe(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	calls := 0
	app.connectionTestInvoke = func(context.Context, string, ServiceConfig) error {
		calls++
		return nil
	}
	engine := strings.Repeat("e", maxEngineIDBytes+1)
	err := app.TestConnection(engine, ServiceConfig{})
	if err == nil {
		t.Fatal("oversized engine should be rejected")
	}
	if strings.Contains(err.Error(), engine) {
		t.Fatal("oversized engine was reflected in the error")
	}
	if calls != 0 {
		t.Fatalf("connection probe calls = %d, want 0", calls)
	}
}

func TestConnectionRejectsOversizedServiceBeforeProbe(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*ServiceConfig)
	}{
		{name: "secret ID", mutate: func(service *ServiceConfig) { service.SecretId = strings.Repeat("x", 4097) }},
		{name: "secret key", mutate: func(service *ServiceConfig) { service.SecretKey = strings.Repeat("x", 4097) }},
		{name: "region", mutate: func(service *ServiceConfig) { service.Region = strings.Repeat("x", 257) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewAppWithDataDir(t.TempDir())
			calls := 0
			app.connectionTestInvoke = func(context.Context, string, ServiceConfig) error {
				calls++
				return nil
			}
			service := ServiceConfig{}
			tt.mutate(&service)
			if err := app.TestConnection("tencent", service); err == nil {
				t.Fatal("oversized service should be rejected")
			}
			if calls != 0 {
				t.Fatalf("connection probe calls = %d, want 0", calls)
			}
		})
	}
}

func TestBaiduConnectionRejectsOversizedServiceBeforeProbe(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*BaiduConfig)
	}{
		{name: "app ID", mutate: func(service *BaiduConfig) { service.AppID = strings.Repeat("x", 4097) }},
		{name: "secret key", mutate: func(service *BaiduConfig) { service.SecretKey = strings.Repeat("x", 4097) }},
		{name: "domain", mutate: func(service *BaiduConfig) { service.Domain = strings.Repeat("x", 33) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := NewAppWithDataDir(t.TempDir())
			calls := 0
			app.baiduConnectionTestInvoke = func(context.Context, BaiduConfig) error {
				calls++
				return nil
			}
			service := BaiduConfig{AppID: "app", SecretKey: "secret", Domain: "general"}
			tt.mutate(&service)
			if err := app.TestBaiduConnection(service); err == nil {
				t.Fatal("oversized Baidu service should be rejected")
			}
			if calls != 0 {
				t.Fatalf("Baidu connection probe calls = %d, want 0", calls)
			}
		})
	}
}

func TestBaiduConnectionUsesBoundedContextAndNormalizesDomain(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	probeErr := errors.New("probe failed")
	app.baiduConnectionTestInvoke = func(ctx context.Context, service BaiduConfig) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("Baidu connection test context has no deadline")
		}
		remaining := time.Until(deadline)
		if remaining <= 0 || remaining > engineTimeout {
			t.Fatalf("Baidu connection test deadline remaining = %v, want (0, %v]", remaining, engineTimeout)
		}
		if service.Domain != "wiki" {
			t.Fatalf("Baidu domain = %q, want wiki", service.Domain)
		}
		return probeErr
	}

	err := app.TestBaiduConnection(BaiduConfig{AppID: "app", SecretKey: "secret", Domain: " WIKI "})
	if !errors.Is(err, probeErr) {
		t.Fatalf("TestBaiduConnection error = %v, want wrapped probe error", err)
	}
}

func TestConnectionInheritsAppCancellation(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	appCtx, cancelApp := context.WithCancel(context.Background())
	app.startup(appCtx)
	app.connectionTestInvoke = func(ctx context.Context, _ string, _ ServiceConfig) error {
		<-ctx.Done()
		return ctx.Err()
	}

	cancelApp()
	err := app.TestConnection("tencent", ServiceConfig{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("TestConnection error = %v, want context.Canceled", err)
	}

	err = app.TestBaiduConnection(BaiduConfig{AppID: "app", SecretKey: "secret"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("TestBaiduConnection error = %v, want context.Canceled", err)
	}
}

func TestGetMemoryStats(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	// 写入一些缓存数据使统计非零
	app.translateCache.set("k1", strings.Repeat("v", 100))
	app.detectCache.set("k2", "zh")

	stats := app.GetMemoryStats()
	if stats.ThresholdBytes != memoryThresholdBytes {
		t.Errorf("threshold = %d, want %d", stats.ThresholdBytes, memoryThresholdBytes)
	}
	if stats.TranslateCache.Items != 1 || stats.TranslateCache.Bytes <= 0 {
		t.Errorf("translateCache = %+v, want 1 item with bytes>0", stats.TranslateCache)
	}
	if stats.DetectCache.Items != 1 || stats.DetectCache.Bytes <= 0 {
		t.Errorf("detectCache = %+v, want 1 item with bytes>0", stats.DetectCache)
	}
	if stats.AllocBytes <= 0 || stats.SysBytes <= 0 {
		t.Errorf("runtime stats = alloc=%d sys=%d, both should be >0", stats.AllocBytes, stats.SysBytes)
	}
	if stats.ExceedsThreshold {
		t.Error("fresh app should not exceed memory threshold")
	}
}

func TestRunGC(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	// 分配一些垃圾
	for i := 0; i < 1000; i++ {
		_ = strings.Repeat("x", 1024)
	}
	before := app.GetMemoryStats()
	app.RunGC()
	after := app.GetMemoryStats()
	if after.NumGC <= before.NumGC {
		t.Errorf("NumGC did not increase: before=%d after=%d", before.NumGC, after.NumGC)
	}
}
