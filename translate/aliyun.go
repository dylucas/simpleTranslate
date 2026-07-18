package translate

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"simpleTranslate/config"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/openapi-util/service"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
)

// 默认阿里云翻译服务地址
const defaultAliyunEndpoint = "mt.cn-hangzhou.aliyuncs.com"

// 客户端缓存：凭据未变时复用 openapi.Client，避免每次请求都重建凭据客户端。
var (
	aliyunClientMu sync.Mutex
	aliyunClient   *openapi.Client
	aliyunCredsSig string // secretId+secretKey+region 的签名，用于感知凭据变更
)

// CreateClient 获取或构建阿里云 openapi.Client。
// 当凭据签名未变化时直接复用缓存的客户端，避免重复初始化开销。
func CreateClient() (*openapi.Client, error) {
	cfg, err := config.GetConfig(config.GetConfigPath())
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	if strings.TrimSpace(cfg.Aliyun.SecretId) == "" || strings.TrimSpace(cfg.Aliyun.SecretKey) == "" {
		return nil, fmt.Errorf("未配置阿里云 AccessKey，请在设置中填写")
	}

	credsSig := cfg.Aliyun.SecretId + ":" + cfg.Aliyun.SecretKey + ":" + cfg.Aliyun.Region

	aliyunClientMu.Lock()
	defer aliyunClientMu.Unlock()
	if aliyunClient != nil && aliyunCredsSig == credsSig {
		return aliyunClient, nil
	}

	credentialsConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(cfg.Aliyun.SecretId).
		SetAccessKeySecret(cfg.Aliyun.SecretKey)
	credentialClient, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return nil, err
	}

	ecsConfig := &openapi.Config{}
	ecsConfig.Endpoint = tea.String(aliyunEndpoint(cfg.Aliyun.Region))
	ecsConfig.Credential = credentialClient

	client, err := openapi.NewClient(ecsConfig)
	if err != nil {
		return nil, err
	}

	aliyunClient = client
	aliyunCredsSig = credsSig
	return client, nil
}

// aliyunEndpoint accepts either a region (for example cn-hangzhou) or a full
// endpoint. Older configurations and the UI default store the region value.
func aliyunEndpoint(regionOrEndpoint string) string {
	value := strings.TrimSpace(regionOrEndpoint)
	if value == "" {
		return defaultAliyunEndpoint
	}
	if parsed, err := url.Parse(value); err == nil && parsed.Host != "" {
		value = parsed.Host
	}
	value = strings.TrimSuffix(value, "/")
	if strings.Contains(value, ".") {
		return value
	}
	return "mt." + value + ".aliyuncs.com"
}

// CreateApiInfo 构造阿里云 RPC 接口请求参数
func CreateApiInfo(apiName string) *openapi.Params {
	return &openapi.Params{
		Action:      tea.String(apiName),
		Version:     tea.String("2018-10-12"),
		Protocol:    tea.String("HTTPS"),
		Method:      tea.String("POST"),
		AuthType:    tea.String("AK"),
		Style:       tea.String("RPC"),
		Pathname:    tea.String("/"),
		ReqBodyType: tea.String("formData"),
		BodyType:    tea.String("json"),
	}
}

type GetDetectLanguageResponse struct {
	Body struct {
		DetectedLanguage      string `json:"DetectedLanguage"`
		LanguageProbabilities string `json:"LanguageProbabilities"`
		RequestID             string `json:"RequestId"`
	} `json:"body"`
	Headers    map[string]string `json:"headers"`
	StatusCode int               `json:"statusCode"`
}

// APIResponse 最外层响应结构
type APIResponse struct {
	Body       ResponseBody      `json:"body"`
	Headers    map[string]string `json:"headers"`
	StatusCode int               `json:"statusCode"`
}

// ResponseBody 阿里云翻译接口响应体
type ResponseBody struct {
	Code      flexInt        `json:"Code"`
	Message   string         `json:"Message"`
	Data      TranslatedData `json:"Data"`
	RequestID string         `json:"RequestId"`
}

// TranslatedData 翻译结果数据
type TranslatedData struct {
	Translated string  `json:"Translated"`
	WordCount  flexInt `json:"WordCount"`
}

// flexInt 兼容阿里云 API 中 Code/WordCount 等字段可能以字符串或数字形式返回的情况。
// 例如 "200"（字符串）与 200（数字）都能正确解析为整数。
type flexInt int

// UnmarshalJSON 同时接受 JSON 数字和带引号的字符串数字
func (f *flexInt) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(string(data))
	if len(s) == 0 {
		return nil
	}
	// 去除可能的引号
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fmt.Errorf("flexInt: 无法解析 %q: %w", string(data), err)
	}
	*f = flexInt(n)
	return nil
}

// GetDetectLanguage 调用阿里云接口识别语种，返回前端约定的语言代码
func GetDetectLanguage(text string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")

	params := CreateApiInfo("GetDetectLanguage")
	body := map[string]interface{}{}
	body["SourceText"] = tea.String(text)
	// 显式设置读写超时，确保即使外层 select 触发超时，SDK 调用也能自然退出，避免 goroutine 泄漏
	runtime := &util.RuntimeOptions{
		ReadTimeout:    tea.Int(30000),
		ConnectTimeout: tea.Int(30000),
	}
	request := &openapi.OpenApiRequest{
		Body: body,
	}

	resp, err := client.CallApi(params, request, runtime)
	if err != nil {
		return "", err
	}

	var result GetDetectLanguageResponse
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return "", err
	}

	if result.StatusCode != 200 {
		return "", fmt.Errorf("阿里云语言识别失败: HTTP %d", result.StatusCode)
	}

	return result.Body.DetectedLanguage, nil
}

// TranslateGeneral 调用阿里云通用翻译接口
func TranslateGeneral(text string, source string, target string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		return "", err
	}

	params := CreateApiInfo("TranslateGeneral")
	queries := map[string]interface{}{}
	body := map[string]interface{}{}
	body["FormatType"] = tea.String("text")
	body["SourceLanguage"] = tea.String(source)
	body["TargetLanguage"] = tea.String(target)
	body["SourceText"] = tea.String(text)
	body["Scene"] = tea.String("general")
	// 显式设置读写超时，与外层 30s 超时保持一致，避免 SDK 调用 Hang 导致 goroutine 泄漏
	runtime := &util.RuntimeOptions{
		ReadTimeout:    tea.Int(30000),
		ConnectTimeout: tea.Int(30000),
	}
	request := &openapi.OpenApiRequest{
		Query: openapiutil.Query(queries),
		Body:  body,
	}

	resp, err := client.CallApi(params, request, runtime)
	if err != nil {
		return "", err
	}

	var result APIResponse
	bytes, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return "", err
	}

	if err := validateTranslateResponse(result); err != nil {
		return "", err
	}

	return result.Body.Data.Translated, nil
}

func validateTranslateResponse(result APIResponse) error {
	if result.StatusCode != 200 {
		return fmt.Errorf("阿里云翻译失败: HTTP %d", result.StatusCode)
	}
	if int(result.Body.Code) != 200 {
		if result.Body.Message != "" {
			return fmt.Errorf("阿里云翻译失败: Code %d: %s", result.Body.Code, result.Body.Message)
		}
		return fmt.Errorf("阿里云翻译失败: Code %d", result.Body.Code)
	}
	return nil
}
