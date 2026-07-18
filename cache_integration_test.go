package main

import (
	"testing"

	"simpleTranslate/config"
)

// TestSaveConfig_ClearsCache 保存配置后应清空翻译/识别缓存
// 凭据变更后旧缓存结果不再有效，必须失效
func TestSaveConfig_ClearsCache(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	// 预填缓存
	app.translateCache.set("aliyun|en|zh|hello", "你好")
	app.detectCache.set("aliyun|hello", "en")
	if app.translateCache.len() != 1 || app.detectCache.len() != 1 {
		t.Fatalf("预填缓存失败: translate=%d detect=%d", app.translateCache.len(), app.detectCache.len())
	}

	// 保存配置应清空缓存
	path := app.getConfigPath()
	orig, _ := config.GetConfig(path)

	if err := app.SaveConfig(orig); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	if app.translateCache.len() != 0 {
		t.Errorf("保存后 translateCache 应清空，得到 %d", app.translateCache.len())
	}
	if app.detectCache.len() != 0 {
		t.Errorf("保存后 detectCache 应清空，得到 %d", app.detectCache.len())
	}
}

// TestNewApp_InitializesCache NewApp 应初始化非 nil 的缓存
func TestNewApp_InitializesCache(t *testing.T) {
	app := NewAppWithDataDir(t.TempDir())
	if app.translateCache == nil {
		t.Error("translateCache 不应为 nil")
	}
	if app.detectCache == nil {
		t.Error("detectCache 不应为 nil")
	}
	if app.translateCache.len() != 0 {
		t.Errorf("新缓存应为空，得到 %d", app.translateCache.len())
	}
}

// TestCacheKey_Delimiter 缓存键分隔符不应造成歧义
func TestCacheKey_Delimiter(t *testing.T) {
	// "a"+"b" 不应等于 "ab"+"" 等
	if cacheKey("a", "b") == cacheKey("ab") {
		t.Error("不同分组不应产生相同 key")
	}
	if cacheKey("a", "b", "c") == cacheKey("a", "bc") {
		t.Error("分隔符应避免拼接歧义")
	}
}
