package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"simpleTranslate/config"
	"simpleTranslate/internal/storage"
	"simpleTranslate/translate"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
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
	// cacheGeneration isolates results produced before and after a config save.
	cacheGeneration atomic.Uint64
	// activeTranslations tracks cancellable requests by their frontend request ID.
	activeTranslationsMu sync.Mutex
	activeTranslations   map[string]*activeTranslation
	historyMu            sync.Mutex
	saveFileDialog       func(context.Context, runtime.SaveDialogOptions) (string, error)
	// translateInvoke is an optional test seam for exercising request cancellation
	// without making a real cloud request.
	translateInvoke func(context.Context, string, string, string, string) (string, error)
	// connectionTestInvoke allows tests to inspect the bounded connection-test
	// context without making a real cloud request.
	connectionTestInvoke func(context.Context, string, ServiceConfig) error
	// baiduConnectionTestInvoke provides the equivalent seam for Baidu drafts.
	baiduConnectionTestInvoke func(context.Context, BaiduConfig) error
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

func (a *App) newEngineContext() (context.Context, context.CancelFunc) {
	base := a.ctx
	if base == nil {
		base = context.Background()
	}
	return context.WithTimeout(base, engineTimeout)
}

func (a *App) beginTranslation(requestID string) (context.Context, func()) {
	ctx, cancel := a.newEngineContext()
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
	if len(requestID) > maxRequestIDBytes {
		return false
	}
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
	Entries  []HistoryEntry `json:"entries"`
	Total    int            `json:"total"`
	AllTotal int            `json:"allTotal"`
	HasMore  bool           `json:"hasMore"`
}

const (
	historyEntryLimit       = 200
	defaultHistoryPageLimit = 10
	maxHistoryPageLimit     = 50
	maxHistoryOutputBytes   = 256 << 10
	maxHistoryLanguageBytes = maxLanguageCodeBytes
	maxHistoryTimeBytes     = 128
	maxHistoryQueryBytes    = maxInputBytes
	maxHistoryFileBytes     = 64 << 20
)

// TranslateText 给前端调用的统一入口。
// 不再返回 Go error：所有错误通过 TranslateResult.Error/ErrorCode 字段返回，
// 前端总能拿到结构化错误用于差异化提示与重试策略。
func (a *App) TranslateText(req TranslateRequest) TranslateResult {
	requestID := req.RequestID
	if len(requestID) > maxRequestIDBytes {
		requestID = ""
	}
	empty := TranslateResult{RequestID: requestID}
	if len(req.RequestID) > maxRequestIDBytes {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, "", fmt.Sprintf("请求 ID 不能超过 %d 个 UTF-8 字节", maxRequestIDBytes), nil))
	}
	if len(req.Engine) > maxEngineIDBytes {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, "", fmt.Sprintf("翻译引擎名称不能超过 %d 个 UTF-8 字节", maxEngineIDBytes), nil))
	}
	if len(req.Source) > maxLanguageCodeBytes || len(req.Target) > maxLanguageCodeBytes {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, "", fmt.Sprintf("语言代码不能超过 %d 个 UTF-8 字节", maxLanguageCodeBytes), nil))
	}
	if len(req.Text) > maxInputBytes {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, "", fmt.Sprintf("原文不能超过 %d 个 UTF-8 字节", maxInputBytes), nil))
	}

	text, source, target, engine := req.Text, req.Source, req.Target, req.Engine
	engine = strings.ToLower(strings.TrimSpace(engine))
	source = normalizeLanguageCode(source)
	target = normalizeLanguageCode(target)
	empty.Source = source
	empty.Target = target
	if strings.TrimSpace(text) == "" {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, engine, "输入文本为空", nil))
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

	result := a.executeEngineTranslation(requestCtx, req.RequestID, engine, text, src, tgt, prefetched)
	if result.ErrorCode != "" || result.Error != "" {
		empty.AutoSrc = src
		empty.Target = tgt
		empty.Error = result.Error
		empty.ErrorCode = result.ErrorCode
		return empty
	}

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

func translationResultCacheKey(generation uint64, engine, mode, source, target, text string) string {
	if engine == "baidu" {
		return versionedCacheKey(generation, engine, mode, source, target, text)
	}
	return versionedCacheKey(generation, engine, source, target, text)
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

// executeEngineTranslation owns the shared single-engine execution lifecycle.
// Callers remain responsible for request validation, language detection, and
// aggregating or emitting the returned result.
func (a *App) executeEngineTranslation(
	ctx context.Context,
	requestID, engine, text, source, target string,
	prefetched *prefetchedTranslation,
) EngineTranslateResult {
	result := EngineTranslateResult{RequestID: requestID, Engine: engine}
	setError := func(err error) EngineTranslateResult {
		translatedErr := wrapTranslateError(engine, err)
		result.Error = translatedErr.Message
		result.ErrorCode = translatedErr.Code
		return result
	}

	if err := ctx.Err(); err != nil {
		return setError(err)
	}
	generation := a.cacheGeneration.Load()
	mode, fallbackNotice, err := a.engineExecutionContext(engine, source, target)
	if err != nil {
		return setError(err)
	}
	if err := ctx.Err(); err != nil {
		return setError(err)
	}

	key := translationResultCacheKey(generation, engine, mode, source, target, text)
	if reusablePrefetchedTranslation(prefetched, engine, mode, source, target) {
		a.translateCache.set(key, prefetched.Text)
	}
	if cached, ok := a.translateCache.get(key); ok {
		if err := ctx.Err(); err != nil {
			return setError(err)
		}
		result.Text = cached
		result.Notice = fallbackNotice
		return result
	}

	translated, err := a.translateWithEngine(ctx, engine, text, source, target)
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		return setError(err)
	}
	result.Text = translated.Text
	result.Notice = translated.Notice
	if result.Notice == "" {
		result.Notice = fallbackNotice
	}
	a.translateCache.set(key, result.Text)
	return result
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
	out := make([]string, 0, len(engines))
	for _, e := range engines {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		if e != "tencent" && e != "aliyun" && e != "baidu" {
			continue
		}
		// 引擎最多 3 个，线性扫描比 map 更省内存
		dup := false
		for _, existing := range out {
			if existing == e {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		out = append(out, e)
	}
	if len(out) == 0 {
		out = defaultEngines()
	}
	return out
}

func defaultEngines() []string {
	return []string{"tencent", "aliyun", "baidu"}
}

const maxPooledCacheKeyBufferBytes = 16 << 10

var cacheKeyBufferPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

// cacheKey returns a fixed-size digest so the cache never retains source text
// through its keys. Length-prefixing keeps field boundaries unambiguous.
func cacheKey(parts ...string) string {
	return buildCacheKey(nil, parts...)
}

func versionedCacheKey(generation uint64, parts ...string) string {
	var generationBuf [20]byte
	generationPart := strconv.AppendUint(generationBuf[:0], generation, 10)
	return buildCacheKey(generationPart, parts...)
}

func buildCacheKey(prefix []byte, parts ...string) string {
	buf := cacheKeyBufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	var lenBuf [20]byte
	if prefix != nil {
		n := strconv.AppendInt(lenBuf[:0], int64(len(prefix)), 10)
		buf.Write(n)
		buf.WriteByte(':')
		buf.Write(prefix)
	}
	for _, part := range parts {
		n := strconv.AppendInt(lenBuf[:0], int64(len(part)), 10)
		buf.Write(n)
		buf.WriteByte(':')
		buf.WriteString(part)
	}

	sum := sha256.Sum256(buf.Bytes())
	clear(buf.Bytes())
	if buf.Cap() <= maxPooledCacheKeyBufferBytes {
		buf.Reset()
		cacheKeyBufferPool.Put(buf)
	}

	var encoded [sha256.Size * 2]byte
	hex.Encode(encoded[:], sum[:])
	return string(encoded[:])
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
		generation := a.cacheGeneration.Load()
		detectKey := versionedCacheKey(generation, e, text)
		// 命中缓存直接返回，避免重复调用
		if v, ok := a.detectCache.get(detectKey); ok {
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
			a.detectCache.set(detectKey, lang)
			return lang, nil, nil
		}

		if e == "baidu" {
			general, generalErr := translate.TranslateBaiduGeneralWithContext(ctx, text, "auto", target, cfg.Baidu)
			if generalErr == nil && isSupportedLanguage(general.From) {
				if ctx.Err() != nil {
					return "", nil, ctx.Err()
				}
				a.detectCache.set(detectKey, general.From)
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
const (
	engineTimeout        = 30 * time.Second
	maxInputBytes        = 6000
	maxRequestIDBytes    = 128
	maxEngineIDBytes     = 32
	maxLanguageCodeBytes = 16
	maxRequestedEngines  = 16
)

// TranslateMulti 多引擎并发翻译：同一句话同时走多个引擎，返回并排结果。
// 单引擎超时 engineTimeout，超时的引擎返回 "翻译超时" 错误而非阻塞整体。
// 不再返回 Go error：识别失败时为每个引擎填充结构化错误，确保前端总能拿到 errorCode。

func (a *App) TranslateMulti(req MultiTranslateRequest) MultiTranslateResult {
	requestID := req.RequestID
	if len(requestID) > maxRequestIDBytes {
		requestID = ""
	}
	invalidRequest := func(message string) MultiTranslateResult {
		err := newTranslateError(ErrCodeInvalidInput, "", message, nil)
		return MultiTranslateResult{
			RequestID: requestID,
			Results:   engineErrorResults(requestID, defaultEngines(), err),
		}
	}
	if len(req.RequestID) > maxRequestIDBytes {
		return invalidRequest(fmt.Sprintf("请求 ID 不能超过 %d 个 UTF-8 字节", maxRequestIDBytes))
	}
	if len(req.Source) > maxLanguageCodeBytes || len(req.Target) > maxLanguageCodeBytes {
		return invalidRequest(fmt.Sprintf("语言代码不能超过 %d 个 UTF-8 字节", maxLanguageCodeBytes))
	}
	if len(req.Engines) > maxRequestedEngines {
		return invalidRequest(fmt.Sprintf("翻译引擎不能超过 %d 个", maxRequestedEngines))
	}
	for _, engine := range req.Engines {
		if len(engine) > maxEngineIDBytes {
			return invalidRequest(fmt.Sprintf("翻译引擎名称不能超过 %d 个 UTF-8 字节", maxEngineIDBytes))
		}
	}
	if len(req.Text) > maxInputBytes {
		return invalidRequest(fmt.Sprintf("原文不能超过 %d 个 UTF-8 字节", maxInputBytes))
	}

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
			r := a.executeEngineTranslation(requestCtx, req.RequestID, engine, text, src, tgt, prefetched)

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
	if len(engine) > maxEngineIDBytes {
		return fmt.Errorf("翻译引擎名称不能超过 %d 个 UTF-8 字节", maxEngineIDBytes)
	}
	if err := config.ValidateServiceConfig(service); err != nil {
		return fmt.Errorf("连接配置无效: %w", err)
	}
	engine = strings.ToLower(strings.TrimSpace(engine))
	if engine == "" {
		engine = "tencent"
	}
	if engine != "tencent" && engine != "aliyun" {
		return fmt.Errorf("未知引擎: %s", engine)
	}

	ctx, cancel := a.newEngineContext()
	defer cancel()

	err := a.probeConnection(ctx, engine, service)
	if err == nil {
		return nil
	}
	if engine == "aliyun" {
		return fmt.Errorf("阿里云连接测试失败: %w", err)
	}
	return fmt.Errorf("混元连接测试失败: %w", err)
}

func (a *App) probeConnection(ctx context.Context, engine string, service ServiceConfig) error {
	if a.connectionTestInvoke != nil {
		return a.connectionTestInvoke(ctx, engine, service)
	}
	const probe = "hello"
	if engine == "aliyun" {
		_, err := translate.GetDetectLanguageWithContext(ctx, probe, service)
		return err
	}
	_, err := translate.DetectLanguageWithContext(ctx, probe, service)
	return err
}

// TestBaiduConnection validates draft Baidu credentials without persisting
// them. The selected translation endpoint is exercised with a supported route.
func (a *App) TestBaiduConnection(service BaiduConfig) error {
	if err := config.ValidateBaiduConfig(service); err != nil {
		return fmt.Errorf("连接配置无效: %w", err)
	}
	ctx, cancel := a.newEngineContext()
	defer cancel()
	service.Domain = strings.ToLower(strings.TrimSpace(service.Domain))
	probe, source, target := "hello", "en", "zh"
	if service.Domain == "novel" || service.Domain == "wiki" {
		probe, source, target = "你好", "zh", "en"
	}
	var err error
	if a.baiduConnectionTestInvoke != nil {
		err = a.baiduConnectionTestInvoke(ctx, service)
	} else {
		_, err = translate.TranslateBaiduWithContext(ctx, probe, source, target, service)
	}
	if err != nil {
		return fmt.Errorf("百度翻译连接测试失败: %w", err)
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
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HistoryEntry{}, nil
		}
		return nil, err
	}
	if info.Size() > maxHistoryFileBytes {
		return nil, fmt.Errorf("历史记录文件超过 %d 字节限制", maxHistoryFileBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	limited := &io.LimitedReader{R: file, N: maxHistoryFileBytes + 1}
	entries, decodeErr := decodeHistory(limited)
	if limited.N == 0 {
		return nil, fmt.Errorf("历史记录文件超过 %d 字节限制", maxHistoryFileBytes)
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	return entries, nil
}

// decodeHistory retains only the bounded history window while still parsing
// the complete document so malformed discarded entries are not silently
// accepted. This avoids allocating an attacker-controlled number of structs.
func decodeHistory(reader io.Reader) ([]HistoryEntry, error) {
	decoder := json.NewDecoder(reader)
	first, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if first == nil {
		if err := requireJSONEOF(decoder); err != nil {
			return nil, err
		}
		return []HistoryEntry{}, nil
	}
	start, ok := first.(json.Delim)
	if !ok || start != '[' {
		return nil, fmt.Errorf("历史记录必须是 JSON 数组")
	}

	entries := make([]HistoryEntry, 0, historyEntryLimit)
	entryIndex := 0
	for decoder.More() {
		var entry HistoryEntry
		if err := decoder.Decode(&entry); err != nil {
			return nil, err
		}
		entryIndex++
		if err := validateHistoryEntryBounds(entry); err != nil {
			return nil, fmt.Errorf("历史记录第 %d 条无效: %w", entryIndex, err)
		}
		if len(entries) < historyEntryLimit {
			entries = append(entries, entry)
		}
	}
	closing, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if closing != json.Delim(']') {
		return nil, fmt.Errorf("历史记录 JSON 数组未正确结束")
	}
	if err := requireJSONEOF(decoder); err != nil {
		return nil, err
	}
	return entries, nil
}

func validateHistoryEntryBounds(entry HistoryEntry) error {
	if len(entry.Input) > maxInputBytes {
		return fmt.Errorf("历史原文不能超过 %d 个 UTF-8 字节", maxInputBytes)
	}
	if len(entry.Output) > maxHistoryOutputBytes {
		return fmt.Errorf("历史译文不能超过 %d 个 UTF-8 字节", maxHistoryOutputBytes)
	}
	if len(entry.Source) > maxHistoryLanguageBytes || len(entry.Target) > maxHistoryLanguageBytes {
		return fmt.Errorf("历史语言代码不能超过 %d 个 UTF-8 字节", maxHistoryLanguageBytes)
	}
	if len(entry.Time) > maxHistoryTimeBytes {
		return fmt.Errorf("历史时间标签不能超过 %d 个 UTF-8 字节", maxHistoryTimeBytes)
	}
	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var trailing json.RawMessage
	err := decoder.Decode(&trailing)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return fmt.Errorf("历史记录包含多余的 JSON 内容")
	}
	return err
}

// saveHistory writes the compatibility JSON format for internal operations.
func (a *App) saveHistory(entries []HistoryEntry) error {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	return a.saveHistoryUnlocked(entries)
}

func (a *App) saveHistoryUnlocked(entries []HistoryEntry) error {
	if len(entries) > historyEntryLimit {
		entries = entries[:historyEntryLimit]
	}
	for i, entry := range entries {
		if err := validateHistoryEntryBounds(entry); err != nil {
			return fmt.Errorf("历史记录第 %d 条无效: %w", i+1, err)
		}
	}
	path := a.getHistoryPath()
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > maxHistoryFileBytes {
		return fmt.Errorf("历史记录文件超过 %d 字节限制", maxHistoryFileBytes)
	}
	return storage.WriteFileAtomic(path, data, 0600)
}

// QueryHistory returns one bounded page without retaining history in memory.
// 单次遍历完成过滤 + 分页，避免先全量拷贝匹配项再切片。
func (a *App) QueryHistory(query HistoryQuery) (HistoryPage, error) {
	if len(query.Query) > maxHistoryQueryBytes {
		return HistoryPage{}, fmt.Errorf("历史搜索词不能超过 %d 个 UTF-8 字节", maxHistoryQueryBytes)
	}
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	entries, err := a.loadHistoryUnlocked()
	if err != nil {
		return HistoryPage{}, err
	}
	needle := strings.ToLower(strings.TrimSpace(query.Query))
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}
	limit := query.Limit
	if limit <= 0 {
		limit = defaultHistoryPageLimit
	}
	if limit > maxHistoryPageLimit {
		limit = maxHistoryPageLimit
	}

	pageEntries := make([]HistoryEntry, 0, limit)
	matched := 0
	for _, entry := range entries {
		if needle != "" {
			if !strings.Contains(strings.ToLower(entry.Input), needle) && !strings.Contains(strings.ToLower(entry.Output), needle) {
				continue
			}
		}
		if matched >= offset && len(pageEntries) < limit {
			pageEntries = append(pageEntries, entry)
		}
		matched++
	}
	return HistoryPage{
		Entries:  pageEntries,
		Total:    matched,
		AllTotal: len(entries),
		HasMore:  matched > offset+len(pageEntries),
	}, nil
}

// AppendHistory atomically prepends a non-duplicate entry and keeps 200 items.
func (a *App) AppendHistory(entry HistoryEntry) (bool, error) {
	if err := validateHistoryEntryBounds(entry); err != nil {
		return false, err
	}
	if strings.TrimSpace(entry.Input) == "" {
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
	if err := storage.WriteFileAtomic(path, data, 0600); err != nil {
		return false, err
	}
	return true, nil
}

// CacheStats 缓存运行时统计，供内存监控端点返回。
type CacheStats struct {
	Items int `json:"items"`
	Bytes int `json:"bytes"`
}

// MemoryStats 内存使用快照，供前端监控与阈值告警使用。
type MemoryStats struct {
	AllocBytes         int64      `json:"allocBytes"`  // Go 堆已分配字节数
	SysBytes           int64      `json:"sysBytes"`    // 从 OS 获取的总内存
	HeapObjects        uint64     `json:"heapObjects"` // 堆对象数量
	NumGC              uint32     `json:"numGC"`       // GC 次数
	TranslateCache     CacheStats `json:"translateCache"`
	DetectCache        CacheStats `json:"detectCache"`
	ActiveTranslations int        `json:"activeTranslations"` // 进行中的翻译请求数
	ThresholdBytes     int64      `json:"thresholdBytes"`     // 告警阈值
	ExceedsThreshold   bool       `json:"exceedsThreshold"`   // 是否超阈值
}

// 内存告警阈值：200MB。桌面翻译应用正常运行时远低于此值，
// 超过则提示用户重启或清理历史。可在设置面板中展示告警状态。
const memoryThresholdBytes = 200 << 20

// GetMemoryStats 返回当前内存使用快照，供前端监控面板周期性查询。
// 注意：runtime.ReadMemStats 会触发短暂 STW，建议调用间隔不低于 5s。
func (a *App) GetMemoryStats() MemoryStats {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	active := 0
	a.activeTranslationsMu.Lock()
	active = len(a.activeTranslations)
	a.activeTranslationsMu.Unlock()
	return MemoryStats{
		AllocBytes:         int64(m.Alloc),
		SysBytes:           int64(m.Sys),
		HeapObjects:        m.HeapObjects,
		NumGC:              m.NumGC,
		TranslateCache:     CacheStats{Items: a.translateCache.len(), Bytes: a.translateCache.bytes()},
		DetectCache:        CacheStats{Items: a.detectCache.len(), Bytes: a.detectCache.bytes()},
		ActiveTranslations: active,
		ThresholdBytes:     memoryThresholdBytes,
		ExceedsThreshold:   int64(m.Alloc) > memoryThresholdBytes,
	}
}

// RunGC 主动触发垃圾回收，用于用户在设置中手动清理内存。
// 清空历史后调用可立即释放历史记录占用的堆内存。
func (a *App) RunGC() {
	goruntime.GC()
}
