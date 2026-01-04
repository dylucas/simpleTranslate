package translate

import (
	"encoding/json"
	"fmt"

	"simpleTranslate/config"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/openapi-util/service"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
)

// Description:
//
// 使用凭据初始化账号Client
//
// @return Client
//
// @throws Exception
func CreateClient() (_result *openapi.Client, _err error) {
	// 工程代码建议使用更安全的无AK方式，凭据配置方式请参见：https://help.aliyun.com/document_detail/378661.html。
	// 使用AK 初始化Credentials Client。
	configPath := config.GetConfigPath()
	config := config.GetConfig(configPath)
	credentialsConfig := new(credentials.Config).
		// 凭证类型。
		SetType("access_key").
		// 设置为AccessKey ID值。
		SetAccessKeyId(config.Aliyun.SecretId).
		// 设置为AccessKey Secret值。
		SetAccessKeySecret(config.Aliyun.SecretKey)
	credentialClient, _err := credentials.NewCredential(credentialsConfig)
	if _err != nil {
		panic(_err)
	}

	ecsConfig := &openapi.Config{}

	// Endpoint 请参考 https://api.aliyun.com/product/alimt
	if config.Aliyun.Region == "" {
		ecsConfig.Endpoint = tea.String("mt.cn-hangzhou.aliyuncs.com")
	} else {
		ecsConfig.Endpoint = tea.String(config.Aliyun.Region)
	}
	ecsConfig.Credential = credentialClient
	_result = &openapi.Client{}
	_result, _err = openapi.NewClient(ecsConfig)
	return _result, _err
}

// Description:
//
// # API 相关
//
// @param path - string Path parameters
//
// @return OpenApi.Params
func CreateApiInfo(apiName string) (_result *openapi.Params) {
	params := &openapi.Params{
		// 接口名称
		Action: tea.String(apiName),
		// 接口版本
		Version: tea.String("2018-10-12"),
		// 接口协议
		Protocol: tea.String("HTTPS"),
		// 接口 HTTP 方法
		Method:   tea.String("POST"),
		AuthType: tea.String("AK"),
		Style:    tea.String("RPC"),
		// 接口 PATH
		Pathname: tea.String("/"),
		// 接口请求体内容格式
		ReqBodyType: tea.String("formData"),
		// 接口响应体内容格式
		BodyType: tea.String("json"),
	}
	_result = params
	return _result
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

// 最外层响应结构
type APIResponse struct {
	Body       ResponseBody      `json:"body"`
	Headers    map[string]string `json:"headers"` // 可选：如果不需要可省略
	StatusCode int               `json:"statusCode"`
}

// Body 内容
type ResponseBody struct {
	Code      int            `json:"Code"`
	Data      TranslatedData `json:"Data"`
	RequestID string         `json:"RequestId"`
}

// Data 内容（翻译结果）
type TranslatedData struct {
	Translated string `json:"Translated"`
	WordCount  int    `json:"WordCount"`
}

func GetDetectLanguage(text string) (string, error) {
	client, _err := CreateClient()
	if _err != nil {
		return "", _err
	}

	params := CreateApiInfo("GetDetectLanguage")
	// body params
	body := map[string]interface{}{}
	body["SourceText"] = tea.String(text)
	// runtime options
	runtime := &util.RuntimeOptions{}
	request := &openapi.OpenApiRequest{
		Body: body,
	}
	// 返回值实际为 Map 类型，可从 Map 中获得三类数据：响应体 body、响应头 headers、HTTP 返回的状态码 statusCode。
	resp, _err := client.CallApi(params, request, runtime)
	if _err != nil {
		return "", _err
	}

	var result GetDetectLanguageResponse
	bytes, _err := json.Marshal(resp)
	if _err != nil {
		return "", _err
	}
	json.Unmarshal(bytes, &result)

	if result.StatusCode != 200 {
		return "", nil
	}

	return result.Body.DetectedLanguage, nil
}

func TranslateGeneral(text string, source string, target string) (string, error) {
	client, _err := CreateClient()
	if _err != nil {
		return "", _err
	}

	params := CreateApiInfo("TranslateGeneral")
	// query params
	queries := map[string]interface{}{}
	// queries["Context"] = tea.String("早上我在家里吃了面包")
	// body params
	body := map[string]interface{}{}
	body["FormatType"] = tea.String("text")
	body["SourceLanguage"] = tea.String(source)
	body["TargetLanguage"] = tea.String(target)
	body["SourceText"] = tea.String(text)
	body["Scene"] = tea.String("general")
	// runtime options
	runtime := &util.RuntimeOptions{}
	request := &openapi.OpenApiRequest{
		Query: openapiutil.Query(queries),
		Body:  body,
	}
	// 返回值实际为 Map 类型，可从 Map 中获得三类数据：响应体 body、响应头 headers、HTTP 返回的状态码 statusCode。
	resp, _err := client.CallApi(params, request, runtime)
	if _err != nil {
		return "", _err
	}

	var result APIResponse
	bytes, _err := json.Marshal(resp)
	if _err != nil {
		return "", _err
	}
	json.Unmarshal(bytes, &result)

	if result.StatusCode != 200 {
		return "", nil
	}

	return result.Body.Data.Translated, nil
}

func main() {
	resp, err := TranslateGeneral("hello world", "en", "zh")
	if err != nil {
		panic(err)
	}
	fmt.Printf("[LOG] %v\n", resp)
}
