package main

import (
	"simpleTranslate/config"
)

// CloudConfig 复用 config 包的定义，避免前后端结构体不同步。
type CloudConfig = config.CloudConfig

// ServiceConfig 复用 config 包的定义。
type ServiceConfig = config.ServiceConfig

// getConfigPath 返回配置文件路径（App 方法包装）。
func (a *App) getConfigPath() string {
	return config.GetConfigPath()
}

// GetConfig 提供给前端调用：读取配置
func (a *App) GetConfig() CloudConfig {
	return config.GetConfig(a.getConfigPath())
}

// SaveConfig 提供给前端调用：保存配置
func (a *App) SaveConfig(cfg CloudConfig) error {
	return config.SaveConfig(a.getConfigPath(), cfg)
}
