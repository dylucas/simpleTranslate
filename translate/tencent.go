package translate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"simpleTranslate/config"
)

// hy-mt2-pro 模型调用地址（腾讯混元 TokenHub，OpenAI 兼容协议）
const hunyuanEndpoint = "https://tokenhub.tencentmaas.com/v1/chat/completions"
const hunyuanModel = "hy-mt2-pro"

// httpClient 复用连接；30s 超时兼顾响应速度与稳定性
var httpClient = &http.Client{Timeout: 30 * time.Second}

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

	req, err := http.NewRequest("POST", hunyuanEndpoint, bytes.NewBuffer(payload))
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

	data, _ := io.ReadAll(resp.Body)
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
func DetectLanguage(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("空文本")
	}
	prompt := fmt.Sprintf(
		"请只回复以下文本对应的语言代码（只能从 zh/en/jp/kr/fr/de/ru/es 中选择一个，不要输出任何其他内容、不要标点）：\n%s",
		text,
	)
	out, err := chatCompletion(prompt)
	if err != nil {
		return "", err
	}
	return normalizeLangCode(out), nil
}

// Translate 使用 hy-mt2-pro 翻译
func Translate(text, source, target string) (string, error) {
	targetName, ok := langCodeToName[target]
	if !ok {
		targetName = target
	}
	prompt := fmt.Sprintf(
		"将以下文本翻译为 %s，注意只需要输出翻译后的结果，不要额外解释：\n%s",
		targetName, text,
	)
	return chatCompletion(prompt)
}
