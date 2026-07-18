package main

import (
	"path/filepath"
	"simpleTranslate/config"
)

// CloudConfig 复用 config 包的定义，避免前后端结构体不同步。
type CloudConfig = config.CloudConfig

// ServiceConfig 复用 config 包的定义。
type ServiceConfig = config.ServiceConfig

// getConfigPath 返回配置文件路径（App 方法包装）。
func (a *App) getConfigPath() string {
	return filepath.Join(a.dataDir, "config.json")
}

// GetConfig 提供给前端调用：读取配置
func (a *App) GetConfig() (CloudConfig, error) {
	return config.GetConfig(a.getConfigPath())
}

// SaveConfig 提供给前端调用：保存配置
// 凭据可能变更，需清空翻译/识别缓存以避免旧凭据结果被误用
func (a *App) SaveConfig(cfg CloudConfig) error {
	if err := config.SaveConfig(a.getConfigPath(), cfg); err != nil {
		return err
	}
	if a.translateCache != nil {
		a.translateCache.clear()
	}
	if a.detectCache != nil {
		a.detectCache.clear()
	}
	return nil
}
