package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestGetConfig_MissingFile 文件不存在时返回默认配置且无错误
func TestGetConfig_MissingFile(t *testing.T) {
	InvalidateCache()
	cfg, err := GetConfig("/tmp/definitely_not_exists_simpleTranslate.json")
	if err != nil {
		t.Fatalf("期望 nil 错误，得到 %v", err)
	}
	if !reflect.DeepEqual(cfg, DefaultCloudConfig()) {
		t.Errorf("期望默认配置，得到 %+v", cfg)
	}
}

// TestSaveConfig_RoundTrip 保存后读取应一致
func TestSaveConfig_RoundTrip(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := CloudConfig{
		Tencent:        ServiceConfig{SecretId: "tid", SecretKey: "tkey", Region: "ap-guangzhou"},
		Aliyun:         ServiceConfig{SecretId: "aid", SecretKey: "akey", Region: "cn-hangzhou"},
		Baidu:          BaiduConfig{AppID: "bid", SecretKey: "bkey", Domain: "it"},
		DefaultEngine:  "tencent",
		IsDark:         true,
		CompareMode:    true,
		CompareEngines: []string{"tencent", "aliyun"},
	}
	want = normalizeConfig(want)

	if err := SaveConfig(path, want); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}

	got, err := GetConfig(path)
	if err != nil {
		t.Fatalf("GetConfig 失败: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("读取与写入不一致\nwant=%+v\ngot =%+v", want, got)
	}
}

// TestSaveConfig_FilePermissions 验证文件权限为 0600
func TestSaveConfig_FilePermissions(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := SaveConfig(path, CloudConfig{}); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat 失败: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("文件权限期望 0600，得到 %o", perm)
	}
}

// TestSaveConfig_InvalidPath 写入不可写路径应返回错误
func TestSaveConfig_InvalidPath(t *testing.T) {
	err := SaveConfig("/nonexistent_dir_xyz/sub/config.json", CloudConfig{})
	if err == nil {
		t.Error("期望返回错误，得到 nil")
	}
}

// TestGetConfig_InvalidJSON 损坏文件返回错误与零值配置
func TestGetConfig_InvalidJSON(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	cfg, err := GetConfig(path)
	if err == nil {
		t.Error("损坏 JSON 期望返回错误，得到 nil")
	}
	if !reflect.DeepEqual(cfg, DefaultCloudConfig()) {
		t.Errorf("损坏 JSON 应返回默认配置，得到 %+v", cfg)
	}
}

func TestGetConfigRejectsOversizedFile(t *testing.T) {
	InvalidateCache()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, maxConfigFileBytes+1); err != nil {
		t.Fatal(err)
	}

	if _, err := GetConfig(path); err == nil || !strings.Contains(err.Error(), "超过") {
		t.Fatalf("oversized config error = %v", err)
	}
}

func TestSaveConfigRejectsOversizedFileWithoutChangingCache(t *testing.T) {
	InvalidateCache()
	path := filepath.Join(t.TempDir(), "config.json")
	baseline := DefaultCloudConfig()
	baseline.DefaultEngine = "aliyun"
	if err := SaveConfig(path, baseline); err != nil {
		t.Fatal(err)
	}

	oversized := baseline
	oversized.DefaultEngine = "baidu"
	oversized.Tencent.SecretKey = strings.Repeat("x", maxConfigFileBytes)
	if err := SaveConfig(path, oversized); err == nil || !strings.Contains(err.Error(), "超过") {
		t.Fatalf("oversized config save error = %v", err)
	}

	got, err := GetConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.DefaultEngine != baseline.DefaultEngine || got.Tencent.SecretKey != baseline.Tencent.SecretKey {
		t.Fatalf("failed save changed cached config: %+v", got)
	}
}

// TestGetConfig_CacheHit 第二次读取应命中缓存（即使删除文件也仍能取到）
func TestGetConfig_CacheHit(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := normalizeConfig(CloudConfig{DefaultEngine: "aliyun", IsDark: true})
	if err := SaveConfig(path, want); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}

	// 第一次读取（应回填缓存）
	got, err := GetConfig(path)
	if err != nil {
		t.Fatalf("第一次 GetConfig 失败: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("第一次读取不一致: got=%+v", got)
	}

	// 删除文件后再次读取，应命中缓存
	if err := os.Remove(path); err != nil {
		t.Fatalf("删除文件失败: %v", err)
	}
	got2, err := GetConfig(path)
	if err != nil {
		t.Fatalf("第二次 GetConfig（应命中缓存）失败: %v", err)
	}
	if !reflect.DeepEqual(got2, want) {
		t.Errorf("缓存读取不一致: got=%+v", got2)
	}
}

func TestGetConfig_MigratesLegacyFields(t *testing.T) {
	InvalidateCache()
	path := filepath.Join(t.TempDir(), "config.json")
	legacy := `{
		"tencent":{"secretKey":"keep-me"},
		"defaultEngine":"aliyun",
		"isDark":false,
		"compareEngines":["aliyun"]
	}`
	if err := os.WriteFile(path, []byte(legacy), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := GetConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != CurrentVersion || cfg.Tencent.SecretKey != "keep-me" {
		t.Fatalf("legacy credentials were not preserved: %+v", cfg)
	}
	if cfg.AutoTranslate != true || cfg.SourceLanguage != "auto" || cfg.TargetLanguage != "zh" {
		t.Fatalf("new defaults were not migrated: %+v", cfg)
	}
	if cfg.IsDark {
		t.Fatal("an explicit legacy isDark=false must be preserved")
	}
}

func TestGetConfigMigratesV2WithoutChangingSelectedEngines(t *testing.T) {
	InvalidateCache()
	path := filepath.Join(t.TempDir(), "config.json")
	v2 := `{
		"version":2,
		"defaultEngine":"aliyun",
		"compareEngines":["aliyun","tencent"]
	}`
	if err := os.WriteFile(path, []byte(v2), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := GetConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 3 || cfg.DefaultEngine != "aliyun" {
		t.Fatalf("v2 migration changed engine selection: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.CompareEngines, []string{"aliyun", "tencent"}) {
		t.Fatalf("v2 migration changed compare engines: %v", cfg.CompareEngines)
	}
	if cfg.Baidu.Domain != BaiduGeneralDomain {
		t.Fatalf("migrated Baidu domain = %q, want general", cfg.Baidu.Domain)
	}
}

func TestNormalizeConfigBaidu(t *testing.T) {
	cfg := normalizeConfig(CloudConfig{
		DefaultEngine:  "baidu",
		Baidu:          BaiduConfig{Domain: " WIKI "},
		CompareEngines: []string{"baidu", "tencent", "baidu", "invalid"},
	})
	if cfg.DefaultEngine != "baidu" || cfg.Baidu.Domain != "wiki" {
		t.Fatalf("Baidu config was not normalized: %+v", cfg)
	}
	if !reflect.DeepEqual(cfg.CompareEngines, []string{"baidu", "tencent"}) {
		t.Fatalf("compare engines = %v", cfg.CompareEngines)
	}

	cfg = normalizeConfig(CloudConfig{Baidu: BaiduConfig{Domain: "medicine"}})
	if cfg.Baidu.Domain != BaiduGeneralDomain {
		t.Fatalf("invalid Baidu domain = %q, want general", cfg.Baidu.Domain)
	}
}

func TestNormalizeConfigEngines(t *testing.T) {
	cfg := normalizeConfig(CloudConfig{
		DefaultEngine:  " Aliyun ",
		CompareEngines: []string{" BAIDU ", "aliyun", "baidu", "invalid"},
		Aliyun:         ServiceConfig{Region: " cn-shanghai "},
	})
	if cfg.DefaultEngine != "aliyun" {
		t.Fatalf("default engine = %q, want aliyun", cfg.DefaultEngine)
	}
	if !reflect.DeepEqual(cfg.CompareEngines, []string{"baidu", "aliyun"}) {
		t.Fatalf("compare engines = %v, want [baidu aliyun]", cfg.CompareEngines)
	}
	if cfg.Aliyun.Region != "cn-shanghai" {
		t.Fatalf("Aliyun region = %q, want cn-shanghai", cfg.Aliyun.Region)
	}

	fallback := normalizeConfig(CloudConfig{
		DefaultEngine:  "unknown",
		CompareEngines: []string{"unknown"},
		Aliyun:         ServiceConfig{Region: "  "},
	})
	if fallback.DefaultEngine != "tencent" {
		t.Fatalf("invalid default engine = %q, want tencent", fallback.DefaultEngine)
	}
	if !reflect.DeepEqual(fallback.CompareEngines, []string{"tencent", "aliyun", "baidu"}) {
		t.Fatalf("invalid compare engines = %v", fallback.CompareEngines)
	}
	if fallback.Aliyun.Region != "cn-hangzhou" {
		t.Fatalf("blank Aliyun region = %q, want cn-hangzhou", fallback.Aliyun.Region)
	}
}

func TestNormalizeConfigLanguages(t *testing.T) {
	cfg := normalizeConfig(CloudConfig{SourceLanguage: " JA ", TargetLanguage: "KO"})
	if cfg.SourceLanguage != "jp" || cfg.TargetLanguage != "kr" {
		t.Fatalf("language aliases were not normalized: source=%q target=%q", cfg.SourceLanguage, cfg.TargetLanguage)
	}

	cfg = normalizeConfig(CloudConfig{SourceLanguage: "invalid", TargetLanguage: ""})
	if cfg.SourceLanguage != "auto" || cfg.TargetLanguage != "zh" {
		t.Fatalf("invalid languages did not fall back: source=%q target=%q", cfg.SourceLanguage, cfg.TargetLanguage)
	}
}

// TestSaveConfig_UpdatesCache 保存后立即读取应反映新值
func TestSaveConfig_UpdatesCache(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := SaveConfig(path, CloudConfig{DefaultEngine: "tencent"}); err != nil {
		t.Fatalf("第一次 SaveConfig 失败: %v", err)
	}
	if err := SaveConfig(path, CloudConfig{DefaultEngine: "aliyun"}); err != nil {
		t.Fatalf("第二次 SaveConfig 失败: %v", err)
	}

	got, err := GetConfig(path)
	if err != nil {
		t.Fatalf("GetConfig 失败: %v", err)
	}
	if got.DefaultEngine != "aliyun" {
		t.Errorf("期望 defaultEngine=aliyun，得到 %q", got.DefaultEngine)
	}
}

func TestGetConfig_CacheReturnsDeepCopy(t *testing.T) {
	InvalidateCache()
	path := filepath.Join(t.TempDir(), "config.json")
	want := []string{"tencent", "aliyun"}
	if err := SaveConfig(path, CloudConfig{CompareEngines: want}); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}

	got, err := GetConfig(path)
	if err != nil {
		t.Fatalf("GetConfig 失败: %v", err)
	}
	got.CompareEngines[0] = "modified"

	again, err := GetConfig(path)
	if err != nil {
		t.Fatalf("第二次 GetConfig 失败: %v", err)
	}
	if !reflect.DeepEqual(again.CompareEngines, want) {
		t.Fatalf("调用方修改污染了配置缓存: got=%v want=%v", again.CompareEngines, want)
	}
}

// TestGetConfig_CacheIsolatedByPath 不同路径的缓存互不影响
func TestGetConfig_CacheIsolatedByPath(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.json")
	pathB := filepath.Join(dir, "b.json")

	if err := SaveConfig(pathA, CloudConfig{DefaultEngine: "tencent"}); err != nil {
		t.Fatalf("SaveConfig A 失败: %v", err)
	}
	if err := SaveConfig(pathB, CloudConfig{DefaultEngine: "aliyun"}); err != nil {
		t.Fatalf("SaveConfig B 失败: %v", err)
	}

	gotA, _ := GetConfig(pathA)
	gotB, _ := GetConfig(pathB)
	if gotA.DefaultEngine != "tencent" {
		t.Errorf("path A 期望 tencent，得到 %q", gotA.DefaultEngine)
	}
	if gotB.DefaultEngine != "aliyun" {
		t.Errorf("path B 期望 aliyun，得到 %q", gotB.DefaultEngine)
	}
}

func TestGetConfigDoesNotOverwriteConcurrentSave(t *testing.T) {
	InvalidateCache()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := SaveConfig(path, CloudConfig{DefaultEngine: "tencent"}); err != nil {
		t.Fatal(err)
	}
	InvalidateCache()

	originalReadFile := readConfigFile
	defer func() { readConfigFile = originalReadFile }()
	readStarted := make(chan struct{})
	releaseRead := make(chan struct{})
	var intercept sync.Once
	readConfigFile = func(path string) ([]byte, error) {
		data, err := os.ReadFile(path)
		intercept.Do(func() {
			close(readStarted)
			<-releaseRead
		})
		return data, err
	}

	readDone := make(chan error, 1)
	go func() {
		_, err := GetConfig(path)
		readDone <- err
	}()
	<-readStarted

	saveDone := make(chan error, 1)
	go func() {
		saveDone <- SaveConfig(path, CloudConfig{DefaultEngine: "baidu"})
	}()
	select {
	case err := <-saveDone:
		close(releaseRead)
		t.Fatalf("SaveConfig completed before the in-flight read: %v", err)
	case <-time.After(100 * time.Millisecond):
		close(releaseRead)
	}
	if err := <-readDone; err != nil {
		t.Fatal(err)
	}
	if err := <-saveDone; err != nil {
		t.Fatal(err)
	}

	cfg, err := GetConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DefaultEngine != "baidu" {
		t.Fatalf("cached default engine = %q, want baidu", cfg.DefaultEngine)
	}
}
