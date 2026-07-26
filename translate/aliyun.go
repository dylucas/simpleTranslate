package translate

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"simpleTranslate/config"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
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
	aliyunClientID aliyunClientKey
)

type aliyunClientKey struct {
	SecretID  string
	SecretKey string
	Endpoint  string
}

// CreateClient 获取或构建阿里云 openapi.Client。
// 当凭据签名未变化时直接复用缓存的客户端，避免重复初始化开销。
// Deprecated: 新代码应使用 CreateClientWithConfig，避免隐式读写用户目录。
func CreateClient() (*openapi.Client, error) {
	cfg, err := config.GetConfig(config.GetConfigPath())
	if err != nil {
		return nil, fmt.Errorf("读取配置失败: %w", err)
	}
	return CreateClientWithConfig(cfg.Aliyun)
}

func CreateClientWithConfig(service config.ServiceConfig) (*openapi.Client, error) {
	service = normalizeAliyunService(service)
	if service.SecretId == "" || service.SecretKey == "" {
		return nil, fmt.Errorf("未配置阿里云 AccessKey，请在设置中填写")
	}

	clientID := aliyunClientKeyForService(service)

	aliyunClientMu.Lock()
	defer aliyunClientMu.Unlock()
	if aliyunClient != nil && aliyunClientID == clientID {
		return aliyunClient, nil
	}

	credentialsConfig := new(credentials.Config).
		SetType("access_key").
		SetAccessKeyId(service.SecretId).
		SetAccessKeySecret(service.SecretKey)
	credentialClient, err := credentials.NewCredential(credentialsConfig)
	if err != nil {
		return nil, err
	}

	ecsConfig := &openapi.Config{}
	ecsConfig.Endpoint = tea.String(clientID.Endpoint)
	ecsConfig.Credential = credentialClient

	client, err := openapi.NewClient(ecsConfig)
	if err != nil {
		return nil, err
	}

	aliyunClient = client
	aliyunClientID = clientID
	return client, nil
}

func normalizeAliyunService(service config.ServiceConfig) config.ServiceConfig {
	service.SecretId = strings.TrimSpace(service.SecretId)
	service.SecretKey = strings.TrimSpace(service.SecretKey)
	service.Region = strings.TrimSpace(service.Region)
	return service
}

func aliyunClientKeyForService(service config.ServiceConfig) aliyunClientKey {
	service = normalizeAliyunService(service)
	return aliyunClientKey{
		SecretID:  service.SecretId,
		SecretKey: service.SecretKey,
		Endpoint:  aliyunEndpoint(service.Region),
	}
}

// InvalidateAliyunClientIfConfigChanged releases credentials retained by the
// cached SDK client when the persisted Aliyun settings no longer match. Calls
// for unrelated configuration saves preserve the reusable client.
func InvalidateAliyunClientIfConfigChanged(service config.ServiceConfig) {
	configuredKey := aliyunClientKeyForService(service)
	aliyunClientMu.Lock()
	defer aliyunClientMu.Unlock()
	if aliyunClientID == configuredKey {
		return
	}
	aliyunClient = nil
	aliyunClientID = aliyunClientKey{}
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

// aliyunLanguageCode converts the application's legacy language codes to the
// ISO codes expected by the Aliyun translation API.
func aliyunLanguageCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "jp":
		return "ja"
	case "kr":
		return "ko"
	default:
		return strings.ToLower(strings.TrimSpace(code))
	}
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
		return fmt.Errorf("flexInt: 无法解析 %q", apiErrorExcerpt(string(data)))
	}
	*f = flexInt(n)
	return nil
}

// GetDetectLanguage 调用阿里云接口识别语种，返回前端约定的语言代码
// Deprecated: 新代码应使用 GetDetectLanguageWithConfig。
func GetDetectLanguage(text string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		return "", err
	}
	return getDetectLanguage(text, client)
}

func GetDetectLanguageWithConfig(text string, service config.ServiceConfig) (string, error) {
	return GetDetectLanguageWithContext(context.Background(), text, service)
}

func getDetectLanguage(text string, client *openapi.Client) (string, error) {
	return getDetectLanguageWithContext(context.Background(), text, client)
}

func GetDetectLanguageWithContext(ctx context.Context, text string, service config.ServiceConfig) (string, error) {
	client, err := CreateClientWithConfig(service)
	if err != nil {
		return "", err
	}
	return getDetectLanguageWithContext(ctx, text, client)
}

func getDetectLanguageWithContext(ctx context.Context, text string, client *openapi.Client) (string, error) {
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

	resp, err := client.CallApiWithCtx(ctx, params, request, runtime)
	if err != nil {
		return "", err
	}

	var result GetDetectLanguageResponse
	if err := decodeThroughBuffer(resp, &result); err != nil {
		return "", err
	}

	if result.StatusCode != 200 {
		return "", fmt.Errorf("阿里云语言识别失败: HTTP %d", result.StatusCode)
	}

	return validateDetectedLanguage(result.Body.DetectedLanguage)
}

// TranslateGeneral 调用阿里云通用翻译接口
// Deprecated: 新代码应使用 TranslateGeneralWithConfig。
func TranslateGeneral(text string, source string, target string) (string, error) {
	client, err := CreateClient()
	if err != nil {
		return "", err
	}
	return translateGeneral(text, source, target, client)
}

func TranslateGeneralWithConfig(text string, source string, target string, service config.ServiceConfig) (string, error) {
	return TranslateGeneralWithContext(context.Background(), text, source, target, service)
}

func translateGeneral(text string, source string, target string, client *openapi.Client) (string, error) {
	return translateGeneralWithContext(context.Background(), text, source, target, client)
}

func TranslateGeneralWithContext(ctx context.Context, text string, source string, target string, service config.ServiceConfig) (string, error) {
	client, err := CreateClientWithConfig(service)
	if err != nil {
		return "", err
	}
	return translateGeneralWithContext(ctx, text, source, target, client)
}

func translateGeneralWithContext(ctx context.Context, text string, source string, target string, client *openapi.Client) (string, error) {

	params := CreateApiInfo("TranslateGeneral")
	body := map[string]interface{}{}
	body["FormatType"] = tea.String("text")
	body["SourceLanguage"] = tea.String(aliyunLanguageCode(source))
	body["TargetLanguage"] = tea.String(aliyunLanguageCode(target))
	body["SourceText"] = tea.String(text)
	body["Scene"] = tea.String("general")
	// 显式设置读写超时，与外层 30s 超时保持一致，避免 SDK 调用 Hang 导致 goroutine 泄漏
	runtime := &util.RuntimeOptions{
		ReadTimeout:    tea.Int(30000),
		ConnectTimeout: tea.Int(30000),
	}
	request := &openapi.OpenApiRequest{
		Body: body,
	}

	resp, err := client.CallApiWithCtx(ctx, params, request, runtime)
	if err != nil {
		return "", err
	}

	var result APIResponse
	if err := decodeThroughBuffer(resp, &result); err != nil {
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
			return fmt.Errorf("阿里云翻译失败: Code %d: %s", result.Body.Code, apiErrorExcerpt(result.Body.Message))
		}
		return fmt.Errorf("阿里云翻译失败: Code %d", result.Body.Code)
	}
	if strings.TrimSpace(result.Body.Data.Translated) == "" {
		return fmt.Errorf("阿里云翻译返回空译文")
	}
	return nil
}
