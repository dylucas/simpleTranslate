package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"simpleTranslate/config"
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

// TestNormalizeEngines 验证引擎列表归一化：去重、小写、过滤非法值、空列表兜底
func TestNormalizeEngines(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"空列表兜底", []string{}, []string{"tencent", "aliyun"}},
		{"nil 兜底", nil, []string{"tencent", "aliyun"}},
		{"单引擎", []string{"tencent"}, []string{"tencent"}},
		{"大小写归一", []string{"TENCENT", "Aliyun"}, []string{"tencent", "aliyun"}},
		{"去重", []string{"tencent", "tencent", "aliyun"}, []string{"tencent", "aliyun"}},
		{"过滤非法值", []string{"tencent", "deepseek", "", "  ", "aliyun"}, []string{"tencent", "aliyun"}},
		{"全非法值兜底", []string{"xxx", "yyy"}, []string{"tencent", "aliyun"}},
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

func TestTranslateMultiEmitsCachedEngineResults(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	app.translateCache.set(cacheKey("tencent", "en", "zh", "hello"), "cached result")
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

	if err := app.SaveHistory(entries); err != nil {
		t.Fatalf("SaveHistory 失败: %v", err)
	}

	got, err := app.LoadHistory()
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

	got, err := app.LoadHistory()
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

	if err := app.SaveHistory(entries); err != nil {
		t.Fatalf("SaveHistory 失败: %v", err)
	}

	got, err := app.LoadHistory()
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

	if err := app.SaveHistory([]HistoryEntry{}); err != nil {
		t.Fatalf("SaveHistory 失败: %v", err)
	}

	got, err := app.LoadHistory()
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

	_, err := app.LoadHistory()
	if err == nil {
		t.Error("损坏 JSON 期望返回错误，得到 nil")
	}
}

func TestLoadHistory_NormalizesNullAndOversizedFiles(t *testing.T) {
	t.Run("null", func(t *testing.T) {
		app := NewAppWithDataDir(t.TempDir())
		if err := os.WriteFile(app.getHistoryPath(), []byte("null"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := app.LoadHistory()
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
		got, err := app.LoadHistory()
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 200 {
			t.Fatalf("oversized history should be capped at 200 entries, got %d", len(got))
		}
	})
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
