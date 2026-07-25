package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"simpleTranslate/internal/storage"
	"simpleTranslate/translate"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	dataDir string
	// eventEmit is injectable for tests; production emits through the Wails runtime.
	eventEmit func(EngineTranslateResult)
	// 翻译结果缓存：百度额外包含领域模式，避免通用/领域结果混用。
	translateCache *lruCache
	// 语种识别缓存：key = engine|text，同一文本不重复检测
	detectCache *lruCache
	// activeTranslations tracks cancellable requests by their frontend request ID.
	activeTranslationsMu sync.Mutex
	activeTranslations   map[string]*activeTranslation
	historyMu            sync.Mutex
	saveFileDialog       func(context.Context, runtime.SaveDialogOptions) (string, error)
	// translateInvoke is an optional test seam for exercising request cancellation
	// without making a real cloud request.
	translateInvoke func(context.Context, string, string, string, string) (string, error)
}

type activeTranslation struct {
	cancel context.CancelFunc
}

func NewApp() *App {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return NewAppWithDataDir(filepath.Join(home, ".simple_translate"))
}

// NewAppWithDataDir creates an application bound to an explicit data
// directory. Production uses NewApp; tests use this constructor to guarantee
// they never read or mutate a user's real configuration.
func NewAppWithDataDir(dataDir string) *App {
	return &App{
		dataDir:            dataDir,
		translateCache:     newLRUCache(64, 512<<10),
		detectCache:        newLRUCache(64, 16<<10),
		activeTranslations: make(map[string]*activeTranslation),
		saveFileDialog:     runtime.SaveFileDialog,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beginTranslation(requestID string) (context.Context, func()) {
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	ctx, cancel := context.WithTimeout(base, engineTimeout)
	if strings.TrimSpace(requestID) == "" {
		return ctx, cancel
	}

	entry := &activeTranslation{cancel: cancel}
	a.activeTranslationsMu.Lock()
	if a.activeTranslations == nil {
		a.activeTranslations = make(map[string]*activeTranslation)
	}
	previous := a.activeTranslations[requestID]
	a.activeTranslations[requestID] = entry
	a.activeTranslationsMu.Unlock()
	if previous != nil {
		previous.cancel()
	}

	finish := func() {
		cancel()
		a.activeTranslationsMu.Lock()
		if a.activeTranslations[requestID] == entry {
			delete(a.activeTranslations, requestID)
		}
		a.activeTranslationsMu.Unlock()
	}
	return ctx, finish
}

// CancelTranslation stops an active translation identified by requestID.
// Empty or unknown IDs are intentionally treated as non-cancellable.
func (a *App) CancelTranslation(requestID string) bool {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return false
	}
	a.activeTranslationsMu.Lock()
	entry, ok := a.activeTranslations[requestID]
	if ok {
		delete(a.activeTranslations, requestID)
	}
	a.activeTranslationsMu.Unlock()
	if !ok {
		return false
	}
	entry.cancel()
	return true
}

type TranslateResult struct {
	RequestID string `json:"requestId"`
	Source    string `json:"source"`
	AutoSrc   string `json:"autoSrc"`
	Target    string `json:"target"`
	Text      string `json:"text"`
	Notice    string `json:"notice,omitempty"`
	// 结构化错误信息：成功时为空，失败时填充。
	// 不再通过 Go error 返回，确保前端总能通过 result 拿到结构化错误用于差异化提示与重试策略。
	Error     string `json:"error,omitempty"`     // 用户可读错误字符串
	ErrorCode string `json:"errorCode,omitempty"` // 错误类别：credentials/network/timeout/rate_limit/invalid_input/service_unavailable/unknown
}

type EngineTranslateResult struct {
	RequestID string `json:"requestId"`
	Engine    string `json:"engine"`
	Text      string `json:"text"`
	Notice    string `json:"notice,omitempty"`
	Error     string `json:"error,omitempty"`     // 兼容旧前端：用户可读错误字符串
	ErrorCode string `json:"errorCode,omitempty"` // 结构化错误类别（credentials/network/timeout/...），前端按类别差异化提示与重试
}

type MultiTranslateResult struct {
	RequestID string                           `json:"requestId"`
	Source    string                           `json:"source"`
	AutoSrc   string                           `json:"autoSrc"`
	Target    string                           `json:"target"`
	Results   map[string]EngineTranslateResult `json:"results"`
}

type TranslateRequest struct {
	RequestID string `json:"requestId"`
	Text      string `json:"text"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	Engine    string `json:"engine"`
}

type MultiTranslateRequest struct {
	RequestID string   `json:"requestId"`
	Text      string   `json:"text"`
	Source    string   `json:"source"`
	Target    string   `json:"target"`
	Engines   []string `json:"engines"`
}

type engineTranslation struct {
	Text   string
	Notice string
}

type prefetchedTranslation struct {
	Engine string
	Text   string
	Source string
	Target string
}

// HistoryEntry 与前端历史记录结构保持一致
type HistoryEntry struct {
	ID     int64  `json:"id"`
	Input  string `json:"input"`
	Output string `json:"output"`
	Source string `json:"source"`
	Target string `json:"target"`
	Time   string `json:"time"`
}

type HistoryQuery struct {
	Query  string `json:"query"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type HistoryPage struct {
	Entries []HistoryEntry `json:"entries"`
	Total   int            `json:"total"`
	HasMore bool           `json:"hasMore"`
}

// TranslateText 给前端调用的统一入口。
// 不再返回 Go error：所有错误通过 TranslateResult.Error/ErrorCode 字段返回，
// 前端总能拿到结构化错误用于差异化提示与重试策略。
func (a *App) TranslateText(req TranslateRequest) TranslateResult {
	text, source, target, engine := req.Text, req.Source, req.Target, req.Engine
	engine = strings.ToLower(strings.TrimSpace(engine))
	source = normalizeLanguageCode(source)
	target = normalizeLanguageCode(target)
	empty := TranslateResult{RequestID: req.RequestID, Source: source, Target: target}
	if strings.TrimSpace(text) == "" {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, engine, "输入文本为空", nil))
	}
	if len(text) > maxInputBytes {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, engine, fmt.Sprintf("原文不能超过 %d 个 UTF-8 字节", maxInputBytes), nil))
	}
	if !isSupportedEngine(engine) {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, engine, "不支持的翻译引擎", nil))
	}
	if err := validateLanguageRoute(source, target); err != nil {
		err.Engine = engine
		return translateResultError(empty, err)
	}
	requestCtx, finish := a.beginTranslation(req.RequestID)
	defer finish()
	src := source
	var prefetched *prefetchedTranslation

	// 自动识别语种（与 TranslateMulti 保持一致：best-effort 跨引擎兜底）
	if src == "" || src == "auto" {
		// 优先使用当前引擎；失败时按引擎列表兜底
		engines := engineFallbackOrder(engine)
		detected, detectedTranslation, err := a.detectLanguageBestEffort(requestCtx, text, target, engines)
		if err != nil {
			te := classifyError(engine, err)
			te.Message = "语言识别失败：" + te.Message
			return translateResultError(empty, te)
		}
		src = detected
		prefetched = detectedTranslation
	}

	// 目标语言兜底
	tgt := fallbackTarget(src, target)

	// 命中缓存直接返回，避免重复调用 API
	mode, notice, err := a.engineExecutionContext(engine, src, tgt)
	if err != nil {
		return translateResultError(empty, classifyError(engine, err))
	}
	ck := translationResultCacheKey(engine, mode, src, tgt, text)
	if reusablePrefetchedTranslation(prefetched, engine, mode, src, tgt) {
		a.translateCache.set(ck, prefetched.Text)
	}
	if v, ok := a.translateCache.get(ck); ok {
		if err := requestCtx.Err(); err != nil {
			return translateResultError(empty, classifyError(engine, err))
		}
		return TranslateResult{
			RequestID: req.RequestID,
			Source:    source,
			AutoSrc:   src,
			Target:    tgt,
			Text:      v,
			Notice:    notice,
		}
	}

	result, err := a.translateWithEngine(requestCtx, engine, text, src, tgt)
	if err != nil {
		te := wrapTranslateError(engine, err)
		empty.AutoSrc = src
		empty.Target = tgt
		empty.Error = te.Message
		empty.ErrorCode = te.Code
		return empty
	}
	if err := requestCtx.Err(); err != nil {
		empty.AutoSrc = src
		empty.Target = tgt
		return translateResultError(empty, classifyError(engine, err))
	}
	if result.Notice == "" {
		result.Notice = notice
	}

	// 写入缓存供下次命中
	a.translateCache.set(ck, result.Text)

	return TranslateResult{
		RequestID: req.RequestID,
		Source:    source,
		AutoSrc:   src,
		Target:    tgt,
		Text:      result.Text,
		Notice:    result.Notice,
	}
}

func translateResultError(result TranslateResult, err *TranslateError) TranslateResult {
	result.Error = err.Message
	result.ErrorCode = err.Code
	return result
}

func isSupportedEngine(engine string) bool {
	return engine == "tencent" || engine == "aliyun" || engine == "baidu"
}

func normalizeLanguageCode(code string) string {
	code = strings.ToLower(strings.TrimSpace(code))
	switch code {
	case "ja":
		return "jp"
	case "ko":
		return "kr"
	default:
		return code
	}
}

func isSupportedLanguage(code string) bool {
	switch code {
	case "zh", "en", "jp", "kr", "fr", "de", "ru", "es":
		return true
	default:
		return false
	}
}

func validateLanguageRoute(source, target string) *TranslateError {
	if source != "" && source != "auto" && !isSupportedLanguage(source) {
		return newTranslateError(ErrCodeInvalidInput, "", "不支持的源语言", nil)
	}
	if !isSupportedLanguage(target) {
		return newTranslateError(ErrCodeInvalidInput, "", "不支持的目标语言", nil)
	}
	return nil
}

func engineFallbackOrder(engine string) []string {
	all := []string{"tencent", "aliyun", "baidu"}
	if !isSupportedEngine(engine) {
		return all
	}
	out := []string{engine}
	for _, candidate := range all {
		if candidate != engine {
			out = append(out, candidate)
		}
	}
	return out
}

func (a *App) translateWithEngine(ctx context.Context, engine, text, source, target string) (engineTranslation, error) {
	if a.translateInvoke != nil {
		text, err := a.translateInvoke(ctx, engine, text, source, target)
		return engineTranslation{Text: text}, err
	}
	cfg, err := a.GetConfig()
	if err != nil {
		return engineTranslation{}, err
	}
	switch engine {
	case "aliyun":
		out, err := translate.TranslateGeneralWithContext(ctx, text, source, target, cfg.Aliyun)
		return engineTranslation{Text: out}, err
	case "baidu":
		out, err := translate.TranslateBaiduWithContext(ctx, text, source, target, cfg.Baidu)
		return engineTranslation{Text: out.Text, Notice: out.Notice}, err
	default:
		out, err := translate.TranslateWithContext(ctx, text, source, target, cfg.Tencent)
		return engineTranslation{Text: out}, err
	}
}

func (a *App) engineExecutionContext(engine, source, target string) (mode, notice string, err error) {
	if engine != "baidu" {
		return "", "", nil
	}
	cfg, err := a.GetConfig()
	if err != nil {
		return "", "", err
	}
	return cfg.Baidu.Domain, translate.BaiduRouteNotice(cfg.Baidu.Domain, source, target), nil
}

func translationResultCacheKey(engine, mode, source, target, text string) string {
	if engine == "baidu" {
		return cacheKey(engine, mode, source, target, text)
	}
	return cacheKey(engine, source, target, text)
}

func reusablePrefetchedTranslation(prefetched *prefetchedTranslation, engine, mode, source, target string) bool {
	if prefetched == nil || engine != "baidu" || prefetched.Engine != engine {
		return false
	}
	if prefetched.Source != source || prefetched.Target != target {
		return false
	}
	return mode == "general" || !translate.BaiduDomainSupportsRoute(mode, source, target)
}

// fallbackTarget 当源语言与目标语言相同时，按习惯切换目标语言
func fallbackTarget(src, target string) string {
	if src != target {
		return target
	}
	switch src {
	case "zh":
		return "en"
	case "en":
		return "zh"
	default:
		return "en"
	}
}

func normalizeEngines(engines []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(engines))
	for _, e := range engines {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if e != "tencent" && e != "aliyun" && e != "baidu" {
			continue
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	if len(out) == 0 {
		out = []string{"tencent", "aliyun", "baidu"}
	}
	return out
}

// cacheKey returns a fixed-size digest so the cache never retains source text
// through its keys. Length-prefixing keeps field boundaries unambiguous.
func cacheKey(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = fmt.Fprintf(h, "%d:", len(part))
		_, _ = h.Write([]byte(part))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// detectLanguageBestEffort tries each engine in order. When Baidu's dedicated
// detector cannot resolve one of the application's languages, general
// translation with from=auto provides both the language and a reusable result.
func (a *App) detectLanguageBestEffort(ctx context.Context, text, target string, engines []string) (string, *prefetchedTranslation, error) {
	var engineErrors []error
	for _, e := range engines {
		if err := ctx.Err(); err != nil {
			return "", nil, err
		}
		// 命中缓存直接返回，避免重复调用
		if v, ok := a.detectCache.get(cacheKey(e, text)); ok {
			return v, nil, nil
		}
		var (
			lang string
			err  error
		)
		cfg, cfgErr := a.GetConfig()
		if cfgErr != nil {
			engineErrors = append(engineErrors, fmt.Errorf("%s: %w", e, cfgErr))
			continue
		} else if e == "aliyun" {
			lang, err = translate.GetDetectLanguageWithContext(ctx, text, cfg.Aliyun)
		} else if e == "baidu" {
			lang, err = translate.DetectBaiduLanguageWithContext(ctx, text, cfg.Baidu)
		} else {
			lang, err = translate.DetectLanguageWithContext(ctx, text, cfg.Tencent)
		}
		if err == nil && isSupportedLanguage(lang) {
			if ctx.Err() != nil {
				return "", nil, ctx.Err()
			}
			a.detectCache.set(cacheKey(e, text), lang)
			return lang, nil, nil
		}

		if e == "baidu" {
			general, generalErr := translate.TranslateBaiduGeneralWithContext(ctx, text, "auto", target, cfg.Baidu)
			if generalErr == nil && isSupportedLanguage(general.From) {
				if ctx.Err() != nil {
					return "", nil, ctx.Err()
				}
				a.detectCache.set(cacheKey(e, text), general.From)
				return general.From, &prefetchedTranslation{
					Engine: "baidu",
					Text:   general.Text,
					Source: general.From,
					Target: target,
				}, nil
			}
			if err != nil {
				engineErrors = append(engineErrors, fmt.Errorf("%s: %w", e, err))
			} else if lang != "" {
				engineErrors = append(engineErrors, fmt.Errorf("%s: 不支持的识别结果 %q", e, lang))
			}
			if generalErr != nil {
				engineErrors = append(engineErrors, fmt.Errorf("%s general auto: %w", e, generalErr))
			} else if general.From != "" {
				engineErrors = append(engineErrors, fmt.Errorf("%s general auto: 不支持的识别结果 %q", e, general.From))
			}
			continue
		}
		if err != nil {
			engineErrors = append(engineErrors, fmt.Errorf("%s: %w", e, err))
		} else if lang != "" {
			engineErrors = append(engineErrors, fmt.Errorf("%s: 不支持的识别结果 %q", e, lang))
		} else {
			engineErrors = append(engineErrors, fmt.Errorf("%s: 未返回识别结果", e))
		}
	}
	return "", nil, errors.Join(engineErrors...)
}

// engineTimeout 单引擎翻译最大等待时间。
// 与 translate 包的 HTTP/SDK 超时保持一致，确保超时后底层调用也会自然退出，
// 不会因外层 select 提前返回而遗留 goroutine。
const engineTimeout = 30 * time.Second
const maxInputBytes = 6000

// TranslateMulti 多引擎并发翻译：同一句话同时走多个引擎，返回并排结果。
// 单引擎超时 engineTimeout，超时的引擎返回 "翻译超时" 错误而非阻塞整体。
// 不再返回 Go error：识别失败时为每个引擎填充结构化错误，确保前端总能拿到 errorCode。

func (a *App) TranslateMulti(req MultiTranslateRequest) MultiTranslateResult {
	text, source, target, engines := req.Text, req.Source, req.Target, req.Engines
	source = normalizeLanguageCode(source)
	target = normalizeLanguageCode(target)
	engines = normalizeEngines(engines)
	src := source
	var prefetched *prefetchedTranslation
	emptyRes := MultiTranslateResult{RequestID: req.RequestID, Source: source, Target: target, Results: map[string]EngineTranslateResult{}}
	if strings.TrimSpace(text) == "" {
		err := newTranslateError(ErrCodeInvalidInput, "", "输入文本为空", nil)
		emptyRes.Results = engineErrorResults(req.RequestID, engines, err)
		return emptyRes
	}
	if len(text) > maxInputBytes {
		err := newTranslateError(ErrCodeInvalidInput, "", fmt.Sprintf("原文不能超过 %d 个 UTF-8 字节", maxInputBytes), nil)
		emptyRes.Results = engineErrorResults(req.RequestID, engines, err)
		return emptyRes
	}
	if err := validateLanguageRoute(source, target); err != nil {
		emptyRes.Results = engineErrorResults(req.RequestID, engines, err)
		return emptyRes
	}
	requestCtx, finish := a.beginTranslation(req.RequestID)
	defer finish()

	// 自动识别一次，供所有引擎共用
	if src == "" || src == "auto" {
		detected, detectedTranslation, err := a.detectLanguageBestEffort(requestCtx, text, target, engines)
		if err != nil {
			te := classifyError("", err)
			te.Message = "语言识别失败：" + te.Message
			// 为所有引擎填充同一错误，前端按 errorCode 显示
			emptyRes.Results = engineErrorResults(req.RequestID, engines, te)
			return emptyRes
		}
		src = detected
		prefetched = detectedTranslation
	}

	tgt := fallbackTarget(src, target)

	results := make(map[string]EngineTranslateResult, len(engines))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, engine := range engines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := EngineTranslateResult{RequestID: req.RequestID, Engine: engine}
			if err := requestCtx.Err(); err != nil {
				te := classifyError(engine, err)
				r.Error = te.Message
				r.ErrorCode = te.Code
				mu.Lock()
				results[engine] = r
				mu.Unlock()
				return
			}

			mode, notice, configErr := a.engineExecutionContext(engine, src, tgt)
			if configErr != nil {
				te := classifyError(engine, configErr)
				r.Error = te.Message
				r.ErrorCode = te.Code
				mu.Lock()
				results[engine] = r
				mu.Unlock()
				return
			}
			// 命中缓存直接用，跳过 API 调用
			ck := translationResultCacheKey(engine, mode, src, tgt, text)
			if reusablePrefetchedTranslation(prefetched, engine, mode, src, tgt) {
				a.translateCache.set(ck, prefetched.Text)
			}
			if v, ok := a.translateCache.get(ck); ok {
				if err := requestCtx.Err(); err != nil {
					te := classifyError(engine, err)
					r.Error = te.Message
					r.ErrorCode = te.Code
				} else {
					r.Text = v
					r.Notice = notice
				}
				mu.Lock()
				results[engine] = r
				mu.Unlock()
				if requestCtx.Err() == nil {
					a.emitEngineResult(r)
				}
				return
			}

			engineCtx, cancel := context.WithTimeout(requestCtx, engineTimeout)
			out, err := a.translateWithEngine(engineCtx, engine, text, src, tgt)
			cancel()
			if err == nil && requestCtx.Err() != nil {
				err = requestCtx.Err()
			}
			if err != nil {
				te := wrapTranslateError(engine, err)
				r.Error = te.Message
				r.ErrorCode = te.Code
			} else {
				r.Text = out.Text
				r.Notice = out.Notice
				if r.Notice == "" {
					r.Notice = notice
				}
				// Only successful, non-cancelled requests may populate the cache.
				a.translateCache.set(ck, out.Text)
			}

			mu.Lock()
			results[engine] = r
			mu.Unlock()

			// Cancelled requests are invalidated locally and must not emit late events.
			if requestCtx.Err() == nil {
				a.emitEngineResult(r)
			}
		}()
	}
	wg.Wait()

	res := MultiTranslateResult{
		RequestID: req.RequestID,
		Source:    source,
		AutoSrc:   src,
		Target:    tgt,
		Results:   results,
	}
	return res
}

func (a *App) emitEngineResult(result EngineTranslateResult) {
	if a.eventEmit != nil {
		a.eventEmit(result)
		return
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "translate:engine-result", result)
	}
}

func engineErrorResults(requestID string, engines []string, err *TranslateError) map[string]EngineTranslateResult {
	results := make(map[string]EngineTranslateResult, len(engines))
	for _, engine := range engines {
		results[engine] = EngineTranslateResult{
			RequestID: requestID,
			Engine:    engine,
			Error:     err.Message,
			ErrorCode: err.Code,
		}
	}
	return results
}

// TestConnection 用一次最小请求验证指定引擎的凭据是否可用
func (a *App) TestConnection(engine string, service ServiceConfig) error {
	engine = strings.ToLower(strings.TrimSpace(engine))
	probe := "hello"
	switch engine {
	case "aliyun":
		_, err := translate.GetDetectLanguageWithConfig(probe, service)
		if err != nil {
			return fmt.Errorf("阿里云连接测试失败: %v", err)
		}
		return nil
	case "tencent", "":
		_, err := translate.DetectLanguageWithConfig(probe, service)
		if err != nil {
			return fmt.Errorf("混元连接测试失败: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("未知引擎: %s", engine)
	}
}

// TestBaiduConnection validates draft Baidu credentials without persisting
// them. The selected translation endpoint is exercised with a supported route.
func (a *App) TestBaiduConnection(service BaiduConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), engineTimeout)
	defer cancel()
	service.Domain = strings.ToLower(strings.TrimSpace(service.Domain))
	probe, source, target := "hello", "en", "zh"
	if service.Domain == "novel" || service.Domain == "wiki" {
		probe, source, target = "你好", "zh", "en"
	}
	if _, err := translate.TranslateBaiduWithContext(ctx, probe, source, target, service); err != nil {
		return fmt.Errorf("百度翻译连接测试失败: %v", err)
	}
	return nil
}

// getHistoryPath 返回历史记录文件路径
func (a *App) getHistoryPath() string {
	_ = os.MkdirAll(a.dataDir, 0700)
	return filepath.Join(a.dataDir, "history.json")
}

// loadHistory reads the compatibility JSON format for internal operations.
func (a *App) loadHistory() ([]HistoryEntry, error) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	return a.loadHistoryUnlocked()
}

func (a *App) loadHistoryUnlocked() ([]HistoryEntry, error) {
	path := a.getHistoryPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	if entries == nil {
		return []HistoryEntry{}, nil
	}
	if len(entries) > 200 {
		entries = entries[:200]
	}
	return entries, nil
}

// saveHistory writes the compatibility JSON format for internal operations.
func (a *App) saveHistory(entries []HistoryEntry) error {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	return a.saveHistoryUnlocked(entries)
}

func (a *App) saveHistoryUnlocked(entries []HistoryEntry) error {
	if len(entries) > 200 {
		entries = entries[:200]
	}
	path := a.getHistoryPath()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, data, 0600)
}

// QueryHistory returns one bounded page without retaining history in memory.
func (a *App) QueryHistory(query HistoryQuery) (HistoryPage, error) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	entries, err := a.loadHistoryUnlocked()
	if err != nil {
		return HistoryPage{}, err
	}
	needle := strings.ToLower(strings.TrimSpace(query.Query))
	filtered := entries
	if needle != "" {
		filtered = make([]HistoryEntry, 0, len(entries))
		for _, entry := range entries {
			if strings.Contains(strings.ToLower(entry.Input), needle) || strings.Contains(strings.ToLower(entry.Output), needle) {
				filtered = append(filtered, entry)
			}
		}
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	end := offset + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	pageEntries := append([]HistoryEntry(nil), filtered[offset:end]...)
	if pageEntries == nil {
		pageEntries = []HistoryEntry{}
	}
	return HistoryPage{Entries: pageEntries, Total: len(filtered), HasMore: end < len(filtered)}, nil
}

// AppendHistory atomically prepends a non-duplicate entry and keeps 200 items.
func (a *App) AppendHistory(entry HistoryEntry) (bool, error) {
	if strings.TrimSpace(entry.Input) == "" || len(entry.Input) > maxInputBytes {
		return false, fmt.Errorf("历史原文必须为 1-%d 个 UTF-8 字节", maxInputBytes)
	}
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	entries, err := a.loadHistoryUnlocked()
	if err != nil {
		return false, err
	}
	if len(entries) > 0 {
		latest := entries[0]
		if latest.Input == entry.Input && latest.Source == entry.Source && latest.Target == entry.Target {
			return false, nil
		}
	}
	entries = append([]HistoryEntry{entry}, entries...)
	return true, a.saveHistoryUnlocked(entries)
}

func (a *App) ClearHistory() error {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	return a.saveHistoryUnlocked([]HistoryEntry{})
}

// ExportHistory writes the complete on-disk history through the native dialog.
// Cancelling the dialog is a successful no-op and returns false.
func (a *App) ExportHistory() (bool, error) {
	a.historyMu.Lock()
	entries, err := a.loadHistoryUnlocked()
	if err != nil {
		a.historyMu.Unlock()
		return false, err
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	a.historyMu.Unlock()
	if err != nil {
		return false, err
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	path, err := a.saveFileDialog(ctx, runtime.SaveDialogOptions{
		Title:                "导出历史记录",
		DefaultFilename:      fmt.Sprintf("simpleTranslate-history-%s.json", time.Now().Format("2006-01-02")),
		CanCreateDirectories: true,
		Filters:              []runtime.FileFilter{{DisplayName: "JSON 文件", Pattern: "*.json"}},
	})
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(path) == "" {
		return false, nil
	}
	return true, storage.WriteFileAtomic(path, data, 0600)
}
