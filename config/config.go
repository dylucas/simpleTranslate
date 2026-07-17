package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// ServiceConfig 对应腾讯云/阿里云的单项配置
type ServiceConfig struct {
	SecretId  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
}

// CloudConfig 对应整个配置文件的结构（单一数据源，供 main 包与 translate 包共用）
type CloudConfig struct {
	Tencent          ServiceConfig `json:"tencent"`
	Aliyun           ServiceConfig `json:"aliyun"`
	DefaultEngine    string        `json:"defaultEngine"` // "tencent" 或 "aliyun"
	IsDark           bool          `json:"isDark"`
	SidebarCollapsed bool          `json:"sidebarCollapsed"`
	// Multi-engine compare
	CompareMode    bool     `json:"compareMode"`
	CompareEngines []string `json:"compareEngines"`
	PickBest       bool     `json:"pickBest"`
	// 剪贴板监听：开启后自动翻译复制的内容
	ClipboardWatch bool `json:"clipboardWatch"`
}

// 进程内配置缓存：避免每次翻译都从磁盘读取，SaveConfig 时同步刷新。
var (
	cacheMu     sync.RWMutex
	cached      *CloudConfig
	cachedPath  string
)

// GetConfigPath 返回配置文件路径，目录不存在时自动创建。
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// 极端情况下退回到当前目录，避免返回空路径
		home = "."
	}
	dir := filepath.Join(home, ".simple_translate")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
	return filepath.Join(dir, "config.json")
}

// GetConfig 读取配置：优先命中内存缓存，否则从磁盘读取并回填缓存。
// 文件不存在时返回零值配置（不视为错误）；其他 IO/解析错误返回 error。
func GetConfig(path string) (CloudConfig, error) {
	cacheMu.RLock()
	if cached != nil && cachedPath == path {
		cfg := *cached
		cacheMu.RUnlock()
		return cfg, nil
	}
	cacheMu.RUnlock()

	var cfg CloudConfig
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置文件失败: %w", err)
	}

	updateCache(path, cfg)
	return cfg, nil
}

// SaveConfig 将配置序列化为带缩进的 JSON 写入磁盘（0600 权限），并刷新内存缓存。
func SaveConfig(path string, cfg CloudConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	updateCache(path, cfg)
	return nil
}

// updateCache 写入缓存（拷贝一份，避免外部修改污染缓存）。
func updateCache(path string, cfg CloudConfig) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cfgCopy := cfg
	cached = &cfgCopy
	cachedPath = path
}

// InvalidateCache 清空内存缓存，主要用于测试场景。
func InvalidateCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cached = nil
	cachedPath = ""
}
