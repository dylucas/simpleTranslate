package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"simpleTranslate/config"
)

// hy-mt2-pro 模型调用地址（腾讯混元 TokenHub，OpenAI 兼容协议）
const hunyuanEndpoint = "https://tokenhub.tencentmaas.com/v1/chat/completions"
const hunyuanModel = "hy-mt2-pro"

// httpClient 复用连接；30s 超时兼顾响应速度与稳定性
var httpClient = &http.Client{Timeout: 30 * time.Second}

// URLs are language-neutral tokens, but the model can let their Latin host,
// protocol, and port characters influence language detection. Remove them
// from the detection sample while leaving the original text untouched for
// translation.
var languageDetectionURLPattern = regexp.MustCompile(`(?i)(?:https?://|ftp://|www\.)[^\s<>"'，。！？；：、]+`)

// langCodeToName 将应用内部语种代码映射为 hy-mt2-pro 期望的中文目标语种名
var langCodeToName = map[string]string{
	"zh": "中文",
	"en": "英语",
	"jp": "日语",
	"ja": "日语",
	"kr": "韩语",
	"ko": "韩语",
	"fr": "法语",
	"de": "德语",
	"ru": "俄语",
	"es": "西班牙语",
}

// nameToLangCode 将中文语种名 / 别名映射回应用内部代码
var nameToLangCode = map[string]string{
	"中文":   "zh",
	"汉语":   "zh",
	"华语":   "zh",
	"英语":   "en",
	"英文":   "en",
	"日语":   "jp",
	"日文":   "jp",
	"韩语":   "kr",
	"韩文":   "kr",
	"法语":   "fr",
	"德语":   "de",
	"俄语":   "ru",
	"西班牙语": "es",
	"西语":   "es",
}

// getApiKey 读取腾讯混元 API Key（直接复用 config 包的内存缓存）
func getApiKey() (string, error) {
	cfg, err := config.GetConfig(config.GetConfigPath())
	if err != nil {
		return "", fmt.Errorf("读取配置失败: %w", err)
	}
	return strings.TrimSpace(cfg.Tencent.SecretKey), nil
}

// chatCompletion 调用混元 OpenAI 兼容接口，返回 message.content
func chatCompletion(prompt string) (string, error) {
	apiKey, err := getApiKey()
	if err != nil {
		return "", err
	}
	return chatCompletionWithAPIKey(prompt, apiKey)
}

func chatCompletionWithConfig(prompt string, service config.ServiceConfig) (string, error) {
	return chatCompletionWithConfigContext(context.Background(), prompt, service)
}

func chatCompletionWithAPIKey(prompt, apiKey string) (string, error) {
	return chatCompletionWithAPIKeyContext(context.Background(), prompt, apiKey)
}

func chatCompletionWithConfigContext(ctx context.Context, prompt string, service config.ServiceConfig) (string, error) {
	return chatCompletionWithAPIKeyContext(ctx, prompt, strings.TrimSpace(service.SecretKey))
}

func chatCompletionWithAPIKeyContext(ctx context.Context, prompt, apiKey string) (string, error) {
	if apiKey == "" {
		return "", fmt.Errorf("未配置腾讯混元 API Key，请在设置中填写")
	}

	body := map[string]interface{}{
		"model": hunyuanModel,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	payload, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", hunyuanEndpoint, bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取混元 API 响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("混元 API 调用失败: HTTP %d: %s", resp.StatusCode, string(data))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", err
	}
	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("混元 API 错误: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("混元 API 返回为空")
	}
	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

// normalizeLangCode 将模型可能返回的代码 / 名称统一为前端使用的代码
func normalizeLangCode(raw string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	code = strings.Trim(code, "`'.\"，。 ")
	switch code {
	case "ja":
		return "jp"
	case "ko":
		return "kr"
	case "zh-cn", "zhongwen", "chinese":
		return "zh"
	case "en-us", "english":
		return "en"
	}
	if c, ok := nameToLangCode[code]; ok {
		return c
	}
	// 模型可能返回 "中文" 等中文名（未转小写）
	if c, ok := nameToLangCode[strings.TrimSpace(raw)]; ok {
		return c
	}
	if _, ok := langCodeToName[code]; ok {
		return code
	}
	return code
}

// DetectLanguage 使用 hy-mt2-pro 识别语种，返回前端约定的语言代码
// Deprecated: 新代码应使用 DetectLanguageWithConfig。
func DetectLanguage(text string) (string, error) {
	return detectLanguage(text, chatCompletion)
}

func DetectLanguageWithConfig(text string, service config.ServiceConfig) (string, error) {
	return DetectLanguageWithContext(context.Background(), text, service)
}

func DetectLanguageWithContext(ctx context.Context, text string, service config.ServiceConfig) (string, error) {
	return detectLanguageWithContext(ctx, text, func(callCtx context.Context, prompt string) (string, error) {
		return chatCompletionWithConfigContext(callCtx, prompt, service)
	})
}

func detectLanguage(text string, complete func(string) (string, error)) (string, error) {
	return detectLanguageWithContext(context.Background(), text, func(_ context.Context, prompt string) (string, error) {
		return complete(prompt)
	})
}

func detectLanguageWithContext(ctx context.Context, text string, complete func(context.Context, string) (string, error)) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("空文本")
	}
	detectionText := languageDetectionURLPattern.ReplaceAllString(text, " ")
	if strings.TrimSpace(detectionText) == "" {
		// Keep URL-only input valid for the remote detector. There is no natural
		// language to infer, but sending an empty sample would be rejected here.
		detectionText = text
	}
	prompt := fmt.Sprintf(
		"请判断以下文本中自然语言主体的语言，忽略 URL/链接、代码、变量名、数字和标点；若有多种语言，以占主要内容的语言为准。只回复语言代码（只能从 zh/en/jp/kr/fr/de/ru/es 中选择一个，不要输出任何其他内容、不要标点）：\n%s",
		detectionText,
	)
	out, err := complete(ctx, prompt)
	if err != nil {
		return "", err
	}
	lang, err := validateDetectedLanguage(out)
	if err != nil {
		return "", err
	}
	return lang, nil
}

func validateDetectedLanguage(raw string) (string, error) {
	lang := normalizeLangCode(raw)
	if _, ok := langCodeToName[lang]; !ok {
		return "", fmt.Errorf("不支持的语言识别结果: %q", strings.TrimSpace(raw))
	}
	return lang, nil
}

// Translate 使用 hy-mt2-pro 翻译
// Deprecated: 新代码应使用 TranslateWithConfig。
func Translate(text, source, target string) (string, error) {
	return translateText(text, source, target, chatCompletion)
}

func TranslateWithConfig(text, source, target string, service config.ServiceConfig) (string, error) {
	return TranslateWithContext(context.Background(), text, source, target, service)
}

func TranslateWithContext(ctx context.Context, text, source, target string, service config.ServiceConfig) (string, error) {
	return translateTextWithContext(ctx, text, source, target, func(callCtx context.Context, prompt string) (string, error) {
		return chatCompletionWithConfigContext(callCtx, prompt, service)
	})
}

func translateText(text, source, target string, complete func(string) (string, error)) (string, error) {
	return translateTextWithContext(context.Background(), text, source, target, func(_ context.Context, prompt string) (string, error) {
		return complete(prompt)
	})
}

func translateTextWithContext(ctx context.Context, text, source, target string, complete func(context.Context, string) (string, error)) (string, error) {
	targetName, ok := langCodeToName[target]
	if !ok {
		targetName = target
	}
	prompt := fmt.Sprintf(
		"将以下文本翻译为 %s，注意只需要输出翻译后的结果，不要额外解释：\n%s",
		targetName, text,
	)
	result, err := complete(ctx, prompt)
	if err != nil {
		return "", err
	}
	result = strings.TrimSpace(result)
	if result == "" {
		return "", fmt.Errorf("混元 API 返回空译文")
	}
	return result, nil
}
