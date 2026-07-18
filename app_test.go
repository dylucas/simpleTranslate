package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"simpleTranslate/config"
)

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
	app := NewApp()

	for _, tt := range []struct {
		name   string
		text   string
		engine string
	}{
		{name: "empty text", text: "  \n\t", engine: "tencent"},
		{name: "unknown engine", text: "hello", engine: "unknown"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			res := app.TranslateText(tt.text, "en", "zh", tt.engine)
			if res.ErrorCode != ErrCodeInvalidInput {
				t.Fatalf("ErrorCode = %q, want %q", res.ErrorCode, ErrCodeInvalidInput)
			}
		})
	}
}

func TestTranslateMultiRejectsEmptyInput(t *testing.T) {
	res := NewApp().TranslateMulti("  ", "en", "zh", []string{"tencent", "aliyun"})
	if len(res.Results) != 2 {
		t.Fatalf("results length = %d, want 2", len(res.Results))
	}
	for engine, result := range res.Results {
		if result.ErrorCode != ErrCodeInvalidInput {
			t.Fatalf("%s ErrorCode = %q, want %q", engine, result.ErrorCode, ErrCodeInvalidInput)
		}
	}
}

// TestSaveLoadHistory_RoundTrip 保存后读取应一致
func TestSaveLoadHistory_RoundTrip(t *testing.T) {
	app := NewApp()
	// 重定向历史路径到临时目录：通过覆盖 home 目录无法实现，这里直接测试 Save/Load
	// 使用真实的 getHistoryPath，但先清空
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
	app := NewApp()
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
	app := NewApp()
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
	app := NewApp()
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
	app := NewApp()
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

// TestGetHistoryPath_CreatesDir 历史路径所在目录会被自动创建
func TestGetHistoryPath_CreatesDir(t *testing.T) {
	app := NewApp()
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
	// 直接写入空配置到真实路径
	path := config.GetConfigPath()
	orig := readOriginalConfig(t, path)
	defer func() {
		_ = config.SaveConfig(path, orig)
		config.InvalidateCache()
	}()

	if err := config.SaveConfig(path, config.CloudConfig{}); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	config.InvalidateCache()

	app := NewApp()
	res := app.TranslateText("hello", "en", "zh", "aliyun")
	if res.ErrorCode == "" {
		t.Error("未配置阿里云凭据期望返回结构化错误码")
	}
	if res.ErrorCode != "credentials" {
		t.Errorf("期望错误码 credentials，得到 %q", res.ErrorCode)
	}
	res = app.TranslateText("hello", "en", "zh", "tencent")
	if res.ErrorCode == "" {
		t.Error("未配置混元凭据期望返回结构化错误码")
	}
	if res.ErrorCode != "credentials" {
		t.Errorf("期望错误码 credentials，得到 %q", res.ErrorCode)
	}
}

// readOriginalConfig 读取并返回原始配置（便于测试后恢复）
func readOriginalConfig(t *testing.T, path string) config.CloudConfig {
	t.Helper()
	cfg, err := config.GetConfig(path)
	if err != nil {
		return config.CloudConfig{}
	}
	return cfg
}
