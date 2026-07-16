package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// TestGetConfig_MissingFile 文件不存在时返回零值配置
func TestGetConfig_MissingFile(t *testing.T) {
	cfg := GetConfig("/tmp/definitely_not_exists_simpleTranslate.json")
	if cfg.DefaultEngine != "" || cfg.IsDark {
		t.Errorf("期望零值配置，得到 %+v", cfg)
	}
}

// TestSaveConfig_RoundTrip 保存后读取应一致
func TestSaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := CloudConfig{
		Tencent:       ServiceConfig{SecretId: "tid", SecretKey: "tkey", Region: "ap-guangzhou"},
		Aliyun:        ServiceConfig{SecretId: "aid", SecretKey: "akey", Region: "cn-hangzhou"},
		DefaultEngine: "tencent",
		IsDark:        true,
		CompareMode:   true,
		CompareEngines: []string{"tencent", "aliyun"},
	}

	if err := SaveConfig(path, want); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}

	got := GetConfig(path)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("读取与写入不一致\nwant=%+v\ngot =%+v", want, got)
	}
}

// TestSaveConfig_FilePermissions 验证文件权限为 0600
func TestSaveConfig_FilePermissions(t *testing.T) {
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

// TestGetConfig_InvalidJSON 损坏文件返回零值配置
func TestGetConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}
	cfg := GetConfig(path)
	if cfg.DefaultEngine != "" {
		t.Errorf("损坏 JSON 应返回零值，得到 %+v", cfg)
	}
}
