package config

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"simpleTranslate/internal/storage"
	"strings"
	"sync"
)

// ServiceConfig 对应腾讯云/阿里云的单项配置
type ServiceConfig struct {
	SecretId  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
}

// BaiduConfig stores the credentials and translation mode used by Baidu.
// Domain is "general" or one of the supported field translation domains.
type BaiduConfig struct {
	AppID     string `json:"appId"`
	SecretKey string `json:"secretKey"`
	Domain    string `json:"domain"`
}

// CloudConfig 对应整个配置文件的结构（单一数据源，供 main 包与 translate 包共用）
type CloudConfig struct {
	Version          int           `json:"version"`
	Tencent          ServiceConfig `json:"tencent"`
	Aliyun           ServiceConfig `json:"aliyun"`
	Baidu            BaiduConfig   `json:"baidu"`
	DefaultEngine    string        `json:"defaultEngine"` // tencent / aliyun / baidu
	IsDark           bool          `json:"isDark"`
	SidebarCollapsed bool          `json:"sidebarCollapsed"`
	AutoTranslate    bool          `json:"autoTranslate"`
	SourceLanguage   string        `json:"sourceLanguage"`
	TargetLanguage   string        `json:"targetLanguage"`
	// Multi-engine compare
	CompareMode    bool     `json:"compareMode"`
	CompareEngines []string `json:"compareEngines"`
	// 剪贴板监听：开启后自动翻译复制的内容
	ClipboardWatch bool `json:"clipboardWatch"`
}

const CurrentVersion = 3

const BaiduGeneralDomain = "general"

const maxConfigFileBytes = 1 << 20

var validBaiduDomains = map[string]struct{}{
	BaiduGeneralDomain: {},
	"it":               {},
	"finance":          {},
	"machinery":        {},
	"senimed":          {},
	"novel":            {},
	"academic":         {},
	"aerospace":        {},
	"wiki":             {},
	"news":             {},
	"law":              {},
	"contract":         {},
}

// NormalizeBaiduDomain returns the canonical configured domain.
func NormalizeBaiduDomain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := validBaiduDomains[value]; ok {
		return value
	}
	return BaiduGeneralDomain
}

// DefaultCloudConfig is also used as the migration base for older files.
// Unmarshalling on top of it preserves intentional false values while filling
// fields that did not exist in previous schema versions.
func DefaultCloudConfig() CloudConfig {
	return CloudConfig{
		Version:          CurrentVersion,
		Aliyun:           ServiceConfig{Region: "cn-hangzhou"},
		Baidu:            BaiduConfig{Domain: BaiduGeneralDomain},
		DefaultEngine:    "tencent",
		IsDark:           true,
		AutoTranslate:    true,
		SourceLanguage:   "auto",
		TargetLanguage:   "zh",
		CompareEngines:   []string{"tencent", "aliyun", "baidu"},
		ClipboardWatch:   false,
		SidebarCollapsed: false,
	}
}

func normalizeConfig(cfg CloudConfig) CloudConfig {
	cfg.Version = CurrentVersion
	cfg.DefaultEngine = normalizeEngine(cfg.DefaultEngine)
	if cfg.DefaultEngine == "" {
		cfg.DefaultEngine = "tencent"
	}
	cfg.Baidu.Domain = NormalizeBaiduDomain(cfg.Baidu.Domain)
	cfg.SourceLanguage = normalizeLanguage(cfg.SourceLanguage, true, "auto")
	cfg.TargetLanguage = normalizeLanguage(cfg.TargetLanguage, false, "zh")
	cfg.Aliyun.Region = strings.TrimSpace(cfg.Aliyun.Region)
	if cfg.Aliyun.Region == "" {
		cfg.Aliyun.Region = "cn-hangzhou"
	}
	seen := map[string]bool{}
	engines := make([]string, 0, len(cfg.CompareEngines))
	for _, engine := range cfg.CompareEngines {
		engine = normalizeEngine(engine)
		if engine != "" && !seen[engine] {
			engines = append(engines, engine)
			seen[engine] = true
		}
	}
	if len(engines) == 0 {
		engines = []string{"tencent", "aliyun", "baidu"}
	}
	cfg.CompareEngines = engines
	return cfg
}

func normalizeEngine(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "tencent", "aliyun", "baidu":
		return value
	default:
		return ""
	}
}

func normalizeLanguage(value string, allowAuto bool, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "ja":
		value = "jp"
	case "ko":
		value = "kr"
	}
	if allowAuto && value == "auto" {
		return value
	}
	switch value {
	case "zh", "en", "jp", "kr", "fr", "de", "ru", "es":
		return value
	default:
		return fallback
	}
}

// 进程内配置缓存：避免每次翻译都从磁盘读取，SaveConfig 时同步刷新。
var (
	cacheMu        sync.RWMutex
	cached         *CloudConfig
	cachedPath     string
	readConfigFile = readConfigFileBounded
)

func readConfigFileBounded(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxConfigFileBytes {
		return nil, fmt.Errorf("配置文件超过 %d 字节限制", maxConfigFileBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxConfigFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxConfigFileBytes {
		return nil, fmt.Errorf("配置文件超过 %d 字节限制", maxConfigFileBytes)
	}
	return data, nil
}

// GetConfigPath 返回配置文件路径，目录不存在时自动创建。
// Deprecated: 新代码应显式传入配置路径，避免隐式读写用户目录。
func GetConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// 极端情况下退回到当前目录，避免返回空路径
		home = "."
	}
	dir := filepath.Join(home, ".simple_translate")
	_ = os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "config.json")
}

// GetConfig 读取配置：优先命中内存缓存，否则从磁盘读取并回填缓存。
// 文件不存在时返回默认配置（不视为错误）；其他 IO/解析错误返回 error。
func GetConfig(path string) (CloudConfig, error) {
	cacheMu.RLock()
	if cached != nil && cachedPath == path {
		cfg := cloneConfig(*cached)
		cacheMu.RUnlock()
		return cfg, nil
	}
	cacheMu.RUnlock()

	cacheMu.Lock()
	defer cacheMu.Unlock()
	// A save or another cache miss may have completed while this caller waited.
	if cached != nil && cachedPath == path {
		return cloneConfig(*cached), nil
	}

	cfg := DefaultCloudConfig()
	data, err := readConfigFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = normalizeConfig(cfg)
			updateCacheLocked(path, cfg)
			return cloneConfig(cfg), nil
		}
		return cfg, fmt.Errorf("读取配置文件失败: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置文件失败: %w", err)
	}

	cfg = normalizeConfig(cfg)
	updateCacheLocked(path, cfg)
	return cfg, nil
}

// SaveConfig 将配置序列化为带缩进的 JSON 写入磁盘（0600 权限），并刷新内存缓存。
func SaveConfig(path string, cfg CloudConfig) error {
	cfg = normalizeConfig(cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}
	if len(data) > maxConfigFileBytes {
		return fmt.Errorf("配置文件超过 %d 字节限制", maxConfigFileBytes)
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if err := storage.WriteFileAtomic(path, data, 0600); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	updateCacheLocked(path, cfg)
	return nil
}

// updateCacheLocked writes a defensive copy while cacheMu is held.
func updateCacheLocked(path string, cfg CloudConfig) {
	cfgCopy := cloneConfig(cfg)
	cached = &cfgCopy
	cachedPath = path
}

func cloneConfig(cfg CloudConfig) CloudConfig {
	cfg.CompareEngines = append([]string(nil), cfg.CompareEngines...)
	return cfg
}

// InvalidateCache 清空内存缓存，主要用于测试场景。
func InvalidateCache() {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	cached = nil
	cachedPath = ""
}
