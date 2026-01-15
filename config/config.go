package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ServiceConfig 对应腾讯云/阿里云的单项配置
type ServiceConfig struct {
	SecretId  string `json:"secretId"`
	SecretKey string `json:"secretKey"`
	Region    string `json:"region"`
}

// CloudConfig 对应整个配置文件的结构
type CloudConfig struct {
	Tencent       ServiceConfig `json:"tencent"`
	Aliyun        ServiceConfig `json:"aliyun"`
	DefaultEngine string        `json:"defaultEngine"` // "tencent" 或 "aliyun"
	// Multi-engine compare (optional; older configs may not have these)
	CompareMode    bool     `json:"compareMode"`
	CompareEngines []string `json:"compareEngines"`
	PickBest       bool     `json:"pickBest"`
}

// 获取配置文件的存放路径
func GetConfigPath() string {
	// 获取用户主目录，例如 C:\Users\Name\GoTranslate
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".simple_translate")

	// 确保目录存在
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
	}
	return filepath.Join(dir, "config.json")
}

// GetConfig 提供给前端调用：读取配置
func GetConfig(path string) CloudConfig {
	var config CloudConfig

	data, err := os.ReadFile(path)
	if err != nil {
		// 如果文件不存在，返回空配置
		return config
	}

	json.Unmarshal(data, &config)
	return config
}

// SaveConfig 提供给前端调用：保存配置
func SaveConfig(path string, cfg CloudConfig) error {
	// 将结构体序列化为带缩进的 JSON
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化失败: %v", err)
	}

	// 写入文件
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return fmt.Errorf("写入文件失败: %v", err)
	}

	fmt.Println("配置已成功更新至:", path)
	return nil
}
