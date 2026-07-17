package translate

import (
	"strings"
	"testing"

	"simpleTranslate/config"
)

// TestCreateClient_MissingCreds 未配置凭据时应返回明确错误
func TestCreateClient_MissingCreds(t *testing.T) {
	config.InvalidateCache()
	path := config.GetConfigPath()
	orig := mustReadConfig(t, path)
	defer func() {
		_ = config.SaveConfig(path, orig)
		config.InvalidateCache()
	}()

	if err := config.SaveConfig(path, config.CloudConfig{}); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	config.InvalidateCache()

	_, err := CreateClient()
	if err == nil {
		t.Error("未配置阿里云凭据期望返回错误")
		return
	}
	if !strings.Contains(err.Error(), "AccessKey") {
		t.Errorf("错误信息应提示 AccessKey，得到: %v", err)
	}
}

// TestCreateClient_PartialCreds 仅配置 SecretId 而无 SecretKey 应返回错误
func TestCreateClient_PartialCreds(t *testing.T) {
	config.InvalidateCache()
	path := config.GetConfigPath()
	orig := mustReadConfig(t, path)
	defer func() {
		_ = config.SaveConfig(path, orig)
		config.InvalidateCache()
	}()

	cfg := config.CloudConfig{
		Aliyun: config.ServiceConfig{SecretId: "only-id", SecretKey: ""},
	}
	if err := config.SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig 失败: %v", err)
	}
	config.InvalidateCache()

	_, err := CreateClient()
	if err == nil {
		t.Error("仅配置 SecretId 期望返回错误")
	}
}

// TestCreateApiInfo 验证 API 参数构造正确
func TestCreateApiInfo(t *testing.T) {
	cases := []struct {
		apiName string
	}{
		{"GetDetectLanguage"},
		{"TranslateGeneral"},
	}
	for _, c := range cases {
		params := CreateApiInfo(c.apiName)
		if *params.Action != c.apiName {
			t.Errorf("Action 期望 %q，得到 %q", c.apiName, *params.Action)
		}
		if *params.Version != "2018-10-12" {
			t.Errorf("Version 期望 2018-10-12，得到 %q", *params.Version)
		}
		if *params.Protocol != "HTTPS" {
			t.Errorf("Protocol 期望 HTTPS，得到 %q", *params.Protocol)
		}
		if *params.Method != "POST" {
			t.Errorf("Method 期望 POST，得到 %q", *params.Method)
		}
		if *params.Style != "RPC" {
			t.Errorf("Style 期望 RPC，得到 %q", *params.Style)
		}
	}
}

// mustReadConfig 读取配置，失败时返回零值（不使测试失败）
func mustReadConfig(t *testing.T, path string) config.CloudConfig {
	t.Helper()
	cfg, err := config.GetConfig(path)
	if err != nil {
		return config.CloudConfig{}
	}
	return cfg
}
