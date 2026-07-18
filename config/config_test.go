package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestGetConfig_MissingFile 文件不存在时返回零值配置且无错误
func TestGetConfig_MissingFile(t *testing.T) {
	InvalidateCache()
	cfg, err := GetConfig("/tmp/definitely_not_exists_simpleTranslate.json")
	if err != nil {
		t.Fatalf("期望 nil 错误，得到 %v", err)
	}
	if cfg.DefaultEngine != "" || cfg.IsDark {
		t.Errorf("期望零值配置，得到 %+v", cfg)
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
		DefaultEngine:  "tencent",
		IsDark:         true,
		CompareMode:    true,
		CompareEngines: []string{"tencent", "aliyun"},
	}

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
	if cfg.DefaultEngine != "" {
		t.Errorf("损坏 JSON 应返回零值，得到 %+v", cfg)
	}
}

// TestGetConfig_CacheHit 第二次读取应命中缓存（即使删除文件也仍能取到）
func TestGetConfig_CacheHit(t *testing.T) {
	InvalidateCache()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := CloudConfig{DefaultEngine: "aliyun", IsDark: true}
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
