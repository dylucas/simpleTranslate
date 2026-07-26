package translate

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"simpleTranslate/config"
)

const (
	baiduGeneralEndpoint = "https://fanyi-api.baidu.com/api/trans/vip/translate"
	baiduFieldEndpoint   = "https://fanyi-api.baidu.com/api/trans/vip/fieldtranslate"
	baiduDetectEndpoint  = "https://fanyi-api.baidu.com/api/trans/vip/language"
	baiduMaxQueryBytes   = 6000
	baiduMaxDetectBytes  = 2000
	baiduFallbackNotice  = "领域不可用，已回退百度通用"
)

// BaiduAPIError preserves Baidu's error code so the application can expose a
// stable, provider-neutral error category to the frontend.
type BaiduAPIError struct {
	Code       string
	Message    string
	HTTPStatus int
}

func (e *BaiduAPIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("百度翻译 API 错误 %s: %s", e.Code, apiErrorExcerpt(e.Message))
	}
	return fmt.Sprintf("百度翻译 API 调用失败: HTTP %d", e.HTTPStatus)
}

// BaiduTranslation contains the translated text and the source language
// reported by Baidu when automatic detection is used.
type BaiduTranslation struct {
	Text   string
	From   string
	Notice string
}

type baiduDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type baiduRequestLimiter struct {
	mu       sync.Mutex
	next     time.Time
	interval time.Duration
}

func newBaiduRequestLimiter(interval time.Duration) *baiduRequestLimiter {
	return &baiduRequestLimiter{interval: interval}
}

func (l *baiduRequestLimiter) wait(ctx context.Context) error {
	if l == nil || l.interval <= 0 {
		return ctx.Err()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		l.mu.Lock()
		now := time.Now()
		delay := l.next.Sub(now)
		if delay <= 0 {
			l.next = now.Add(l.interval)
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Another waiter may claim this slot first, so re-check under the lock.
		}
	}
}

type baiduClient struct {
	httpClient      baiduDoer
	generalEndpoint string
	fieldEndpoint   string
	detectEndpoint  string
	limiter         *baiduRequestLimiter
	salt            func() (string, error)
}

var defaultBaiduClient = &baiduClient{
	httpClient:      &http.Client{Timeout: 30 * time.Second},
	generalEndpoint: baiduGeneralEndpoint,
	fieldEndpoint:   baiduFieldEndpoint,
	detectEndpoint:  baiduDetectEndpoint,
	limiter:         newBaiduRequestLimiter(time.Second),
	salt:            newBaiduSalt,
}

func newBaiduSalt() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成百度翻译随机数失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func baiduSign(appID, query, salt, domain, secretKey string) string {
	// 直接写入哈希，避免拼接可达 6KB+ 的 payload 字符串
	h := md5.New()
	io.WriteString(h, appID)
	io.WriteString(h, query)
	io.WriteString(h, salt)
	if domain != "" {
		io.WriteString(h, domain)
	}
	io.WriteString(h, secretKey)
	var sum [md5.Size]byte
	h.Sum(sum[:0])
	return hex.EncodeToString(sum[:])
}

func validateBaiduConfig(service config.BaiduConfig) error {
	if strings.TrimSpace(service.AppID) == "" || strings.TrimSpace(service.SecretKey) == "" {
		return fmt.Errorf("未配置百度翻译 APP ID 或密钥，请在设置中填写")
	}
	return nil
}

func baiduLanguageCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "kr", "ko":
		return "kor"
	case "fr":
		return "fra"
	case "es":
		return "spa"
	case "ja":
		return "jp"
	default:
		return strings.ToLower(strings.TrimSpace(code))
	}
}

func appLanguageCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "kor", "ko":
		return "kr"
	case "fra":
		return "fr"
	case "spa":
		return "es"
	case "ja":
		return "jp"
	default:
		return strings.ToLower(strings.TrimSpace(code))
	}
}

func (c *baiduClient) postForm(ctx context.Context, endpoint string, form url.Values) ([]byte, error) {
	if err := c.limiter.wait(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := readAPIResponse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取百度翻译响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &BaiduAPIError{HTTPStatus: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	}
	return data, nil
}

type baiduCode string

func (c *baiduCode) UnmarshalJSON(data []byte) error {
	value := strings.TrimSpace(string(data))
	if value == "" || value == "null" {
		*c = ""
		return nil
	}
	if value[0] == '"' {
		var decoded string
		if err := json.Unmarshal(data, &decoded); err != nil {
			return err
		}
		*c = baiduCode(decoded)
		return nil
	}
	*c = baiduCode(value)
	return nil
}

type baiduTranslationResponse struct {
	From        string    `json:"from"`
	To          string    `json:"to"`
	ErrorCode   baiduCode `json:"error_code"`
	ErrorMsg    string    `json:"error_msg"`
	TransResult []struct {
		Source      string `json:"src"`
		Destination string `json:"dst"`
	} `json:"trans_result"`
}

func (c *baiduClient) translate(ctx context.Context, text, source, target, domain string, service config.BaiduConfig) (BaiduTranslation, error) {
	if err := validateBaiduConfig(service); err != nil {
		return BaiduTranslation{}, err
	}
	if strings.TrimSpace(text) == "" {
		return BaiduTranslation{}, fmt.Errorf("空文本")
	}
	if len([]byte(text)) > baiduMaxQueryBytes {
		return BaiduTranslation{}, fmt.Errorf("百度翻译原文不能超过 %d 个 UTF-8 字节", baiduMaxQueryBytes)
	}

	salt, err := c.salt()
	if err != nil {
		return BaiduTranslation{}, err
	}
	appID := strings.TrimSpace(service.AppID)
	secretKey := strings.TrimSpace(service.SecretKey)
	form := url.Values{
		"q":     {text},
		"from":  {baiduLanguageCode(source)},
		"to":    {baiduLanguageCode(target)},
		"appid": {appID},
		"salt":  {salt},
	}
	endpoint := c.generalEndpoint
	if domain != "" {
		endpoint = c.fieldEndpoint
		form.Set("domain", domain)
	}
	form.Set("sign", baiduSign(appID, text, salt, domain, secretKey))

	data, err := c.postForm(ctx, endpoint, form)
	if err != nil {
		return BaiduTranslation{}, err
	}
	var response baiduTranslationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return BaiduTranslation{}, fmt.Errorf("解析百度翻译响应失败: %w", err)
	}
	if response.ErrorCode != "" && response.ErrorCode != "52000" && response.ErrorCode != "0" {
		return BaiduTranslation{}, &BaiduAPIError{Code: string(response.ErrorCode), Message: apiErrorExcerpt(response.ErrorMsg)}
	}
	parts := make([]string, 0, len(response.TransResult))
	for _, item := range response.TransResult {
		parts = append(parts, item.Destination)
	}
	translated := strings.TrimSpace(strings.Join(parts, "\n"))
	if translated == "" {
		return BaiduTranslation{}, fmt.Errorf("百度翻译返回空译文")
	}
	return BaiduTranslation{Text: translated, From: appLanguageCode(response.From)}, nil
}

// BaiduDomainSupportsRoute reports whether a configured field domain supports
// the resolved language direction.
func BaiduDomainSupportsRoute(domain, source, target string) bool {
	domain = config.NormalizeBaiduDomain(domain)
	if domain == config.BaiduGeneralDomain {
		return false
	}
	if domain == "novel" || domain == "wiki" {
		return source == "zh" && target == "en"
	}
	return (source == "zh" && target == "en") || (source == "en" && target == "zh")
}

// BaiduRouteNotice returns the user-visible notice for a deterministic field
// to general fallback.
func BaiduRouteNotice(domain, source, target string) string {
	domain = config.NormalizeBaiduDomain(domain)
	if domain != config.BaiduGeneralDomain && !BaiduDomainSupportsRoute(domain, source, target) {
		return baiduFallbackNotice
	}
	return ""
}

// TranslateBaiduWithContext uses the selected domain when its route is
// supported, otherwise it deliberately falls back to general translation.
func TranslateBaiduWithContext(ctx context.Context, text, source, target string, service config.BaiduConfig) (BaiduTranslation, error) {
	domain := config.NormalizeBaiduDomain(service.Domain)
	requestDomain := ""
	notice := ""
	if domain != config.BaiduGeneralDomain {
		if BaiduDomainSupportsRoute(domain, source, target) {
			requestDomain = domain
		} else {
			notice = baiduFallbackNotice
		}
	}
	result, err := defaultBaiduClient.translate(ctx, text, source, target, requestDomain, service)
	if err != nil {
		return BaiduTranslation{}, err
	}
	result.Notice = notice
	return result, nil
}

// TranslateBaiduGeneralWithContext always uses the general endpoint. It is
// used to recover automatic language detection while retaining the response.
func TranslateBaiduGeneralWithContext(ctx context.Context, text, source, target string, service config.BaiduConfig) (BaiduTranslation, error) {
	return defaultBaiduClient.translate(ctx, text, source, target, "", service)
}

type baiduDetectResponse struct {
	ErrorCode baiduCode `json:"error_code"`
	ErrorMsg  string    `json:"error_msg"`
	Data      struct {
		Source string `json:"src"`
	} `json:"data"`
}

func (c *baiduClient) detect(ctx context.Context, text string, service config.BaiduConfig) (string, error) {
	if err := validateBaiduConfig(service); err != nil {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("空文本")
	}
	text = utf8Prefix(text, baiduMaxDetectBytes)
	salt, err := c.salt()
	if err != nil {
		return "", err
	}
	appID := strings.TrimSpace(service.AppID)
	form := url.Values{
		"q":     {text},
		"appid": {appID},
		"salt":  {salt},
		"sign":  {baiduSign(appID, text, salt, "", strings.TrimSpace(service.SecretKey))},
	}
	data, err := c.postForm(ctx, c.detectEndpoint, form)
	if err != nil {
		return "", err
	}
	var response baiduDetectResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("解析百度语种识别响应失败: %w", err)
	}
	if response.ErrorCode != "" && response.ErrorCode != "0" {
		return "", &BaiduAPIError{Code: string(response.ErrorCode), Message: apiErrorExcerpt(response.ErrorMsg)}
	}
	lang := appLanguageCode(response.Data.Source)
	if lang == "" {
		return "", fmt.Errorf("百度语种识别未返回结果")
	}
	return lang, nil
}

func DetectBaiduLanguageWithContext(ctx context.Context, text string, service config.BaiduConfig) (string, error) {
	return defaultBaiduClient.detect(ctx, text, service)
}

func DetectBaiduLanguageWithConfig(text string, service config.BaiduConfig) (string, error) {
	return DetectBaiduLanguageWithContext(context.Background(), text, service)
}
