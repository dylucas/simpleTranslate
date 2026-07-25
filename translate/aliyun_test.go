package translate

import (
	"encoding/json"
	"strings"
	"testing"

	"simpleTranslate/config"
)

// TestCreateClient_MissingCreds 未配置凭据时应返回明确错误
func TestCreateClient_MissingCreds(t *testing.T) {
	_, err := CreateClientWithConfig(config.ServiceConfig{})
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
	_, err := CreateClientWithConfig(config.ServiceConfig{SecretId: "only-id"})
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

func TestAliyunEndpoint(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "default", input: "", want: defaultAliyunEndpoint},
		{name: "region", input: "cn-hangzhou", want: defaultAliyunEndpoint},
		{name: "endpoint", input: "mt.cn-shanghai.aliyuncs.com", want: "mt.cn-shanghai.aliyuncs.com"},
		{name: "URL", input: "https://mt.cn-beijing.aliyuncs.com/", want: "mt.cn-beijing.aliyuncs.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := aliyunEndpoint(tt.input); got != tt.want {
				t.Fatalf("aliyunEndpoint(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAliyunLanguageCode(t *testing.T) {
	tests := map[string]string{
		"jp":   "ja",
		"JP":   "ja",
		"kr":   "ko",
		" KR ": "ko",
		"zh":   "zh",
		"en":   "en",
	}
	for input, want := range tests {
		if got := aliyunLanguageCode(input); got != want {
			t.Errorf("aliyunLanguageCode(%q) = %q, want %q", input, got, want)
		}
	}
}

// TestFlexInt_UnmarshalJSON 验证 flexInt 能兼容字符串和数字两种形式
func TestFlexInt_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  int
	}{
		{"字符串形式", `"200"`, 200},
		{"数字形式", `200`, 200},
		{"带空格字符串", `"  200  "`, 200},
		{"零值", `"0"`, 0},
		{"负数字符串", `"-1"`, -1},
		{"负数数字", `-1`, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got flexInt
			if err := json.Unmarshal([]byte(c.input), &got); err != nil {
				t.Fatalf("json.Unmarshal(%q) 失败: %v", c.input, err)
			}
			if int(got) != c.want {
				t.Errorf("flexInt 解析 %q = %d, want %d", c.input, int(got), c.want)
			}
		})
	}
}

// TestFlexInt_InvalidInput 非法输入应返回错误
func TestFlexInt_InvalidInput(t *testing.T) {
	cases := []string{`"abc"`, `true`, `null`, `1.5`}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			var got flexInt
			if err := json.Unmarshal([]byte(c), &got); err == nil {
				t.Errorf("期望解析 %q 返回错误，得到 %d", c, int(got))
			}
		})
	}
}

// TestAPIResponse_UnmarshalStringCode 验证 Code 为字符串时能正确反序列化（回归测试）
func TestAPIResponse_UnmarshalStringCode(t *testing.T) {
	// 模拟阿里云真实返回：Code 为字符串 "200"，WordCount 为字符串
	raw := `{
		"body": {
			"Code": "200",
			"Data": {"Translated": "你好", "WordCount": "2"},
			"RequestId": "req-123"
		},
		"headers": {},
		"statusCode": 200
	}`

	var resp APIResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if int(resp.Body.Code) != 200 {
		t.Errorf("Code 期望 200，得到 %d", int(resp.Body.Code))
	}
	if resp.Body.Data.Translated != "你好" {
		t.Errorf("Translated 期望 你好，得到 %q", resp.Body.Data.Translated)
	}
	if int(resp.Body.Data.WordCount) != 2 {
		t.Errorf("WordCount 期望 2，得到 %d", int(resp.Body.Data.WordCount))
	}
}

// TestAPIResponse_UnmarshalNumericCode 验证 Code 为数字时也能正确反序列化
func TestAPIResponse_UnmarshalNumericCode(t *testing.T) {
	raw := `{
		"body": {
			"Code": 200,
			"Data": {"Translated": "hello", "WordCount": 1},
			"RequestId": "req-456"
		},
		"headers": {},
		"statusCode": 200
	}`

	var resp APIResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}
	if int(resp.Body.Code) != 200 {
		t.Errorf("Code 期望 200，得到 %d", int(resp.Body.Code))
	}
	if int(resp.Body.Data.WordCount) != 1 {
		t.Errorf("WordCount 期望 1，得到 %d", int(resp.Body.Data.WordCount))
	}
}

func TestValidateTranslateResponse(t *testing.T) {
	tests := []struct {
		name    string
		result  APIResponse
		wantErr bool
	}{
		{
			name:   "success",
			result: APIResponse{StatusCode: 200, Body: ResponseBody{Code: flexInt(200), Data: TranslatedData{Translated: "ok"}}},
		},
		{
			name:    "HTTP error",
			result:  APIResponse{StatusCode: 503, Body: ResponseBody{Code: flexInt(200)}},
			wantErr: true,
		},
		{
			name:    "business error",
			result:  APIResponse{StatusCode: 200, Body: ResponseBody{Code: flexInt(10010), Message: "invalid access key"}},
			wantErr: true,
		},
		{
			name:    "empty translation",
			result:  APIResponse{StatusCode: 200, Body: ResponseBody{Code: flexInt(200)}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTranslateResponse(tt.result)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateTranslateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
