package main

import (
	"context"
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
	ctx context.Context
	// 翻译结果缓存：key = engine|source|target|text，避免重复调用 API
	translateCache *lruCache
	// 语种识别缓存：key = engine|text，同一文本不重复检测
	detectCache *lruCache
}

func NewApp() *App {
	return &App{
		translateCache: newLRUCache(128),
		detectCache:    newLRUCache(128),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type TranslateResult struct {
	Source  string `json:"source"`
	AutoSrc string `json:"autoSrc"`
	Target  string `json:"target"`
	Text    string `json:"text"`
	// 结构化错误信息：成功时为空，失败时填充。
	// 不再通过 Go error 返回，确保前端总能通过 result 拿到结构化错误用于差异化提示与重试策略。
	Error     string `json:"error,omitempty"`     // 用户可读错误字符串
	ErrorCode string `json:"errorCode,omitempty"` // 错误类别：credentials/network/timeout/rate_limit/invalid_input/service_unavailable/unknown
}

type EngineTranslateResult struct {
	Engine    string `json:"engine"`
	Text      string `json:"text"`
	Error     string `json:"error,omitempty"`     // 兼容旧前端：用户可读错误字符串
	ErrorCode string `json:"errorCode,omitempty"` // 结构化错误类别（credentials/network/timeout/...），前端按类别差异化提示与重试
}

type MultiTranslateResult struct {
	Source  string                           `json:"source"`
	AutoSrc string                           `json:"autoSrc"`
	Target  string                           `json:"target"`
	Results map[string]EngineTranslateResult `json:"results"`
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

// TranslateText 给前端调用的统一入口。
// 不再返回 Go error：所有错误通过 TranslateResult.Error/ErrorCode 字段返回，
// 前端总能拿到结构化错误用于差异化提示与重试策略。
func (a *App) TranslateText(text string, source string, target string, engine string) TranslateResult {
	empty := TranslateResult{Source: source, Target: target}
	engine = strings.ToLower(strings.TrimSpace(engine))
	if strings.TrimSpace(text) == "" {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, engine, "输入文本为空", nil))
	}
	if !isSupportedEngine(engine) {
		return translateResultError(empty, newTranslateError(ErrCodeInvalidInput, engine, "不支持的翻译引擎", nil))
	}
	src := source

	// 自动识别语种（与 TranslateMulti 保持一致：best-effort 跨引擎兜底）
	if src == "" || src == "auto" {
		// 优先使用当前引擎；失败时按引擎列表兜底
		engines := engineFallbackOrder(engine)
		detected, err := a.detectLanguageBestEffort(text, engines)
		if err != nil {
			te := classifyError(engine, err)
			te.Message = "语言识别失败：" + te.Message
			return translateResultError(empty, te)
		}
		src = detected
	}

	// 目标语言兜底
	tgt := fallbackTarget(src, target)

	// 命中缓存直接返回，避免重复调用 API
	ck := cacheKey(engine, src, tgt, text)
	if v, ok := a.translateCache.get(ck); ok {
		return TranslateResult{
			Source:  source,
			AutoSrc: src,
			Target:  tgt,
			Text:    v,
		}
	}

	result, err := translateWithEngine(engine, text, src, tgt)
	if err != nil {
		te := wrapTranslateError(engine, err)
		empty.AutoSrc = src
		empty.Target = tgt
		empty.Error = te.Message
		empty.ErrorCode = te.Code
		return empty
	}

	// 写入缓存供下次命中
	a.translateCache.set(ck, result)

	return TranslateResult{
		Source:  source,
		AutoSrc: src,
		Target:  tgt,
		Text:    result,
	}
}

func translateResultError(result TranslateResult, err *TranslateError) TranslateResult {
	result.Error = err.Message
	result.ErrorCode = err.Code
	return result
}

func isSupportedEngine(engine string) bool {
	return engine == "tencent" || engine == "aliyun"
}

func engineFallbackOrder(engine string) []string {
	if engine == "aliyun" {
		return []string{"aliyun", "tencent"}
	}
	return []string{"tencent", "aliyun"}
}

func translateWithEngine(engine, text, source, target string) (string, error) {
	if engine == "aliyun" {
		return translate.TranslateGeneral(text, source, target)
	}
	return translate.Translate(text, source, target)
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
		if e != "tencent" && e != "aliyun" {
			continue
		}
		if seen[e] {
			continue
		}
		seen[e] = true
		out = append(out, e)
	}
	if len(out) == 0 {
		out = []string{"tencent", "aliyun"}
	}
	return out
}

// cacheKey 构造缓存键，用 \x1f（单元分隔符）分隔避免拼接歧义
func cacheKey(parts ...string) string {
	return strings.Join(parts, "\x1f")
}

// detectLanguageBestEffort 尝试多引擎识别语种，优先命中缓存。
// Try in given order; prefer aliyun if present because it has dedicated API.
func (a *App) detectLanguageBestEffort(text string, engines []string) (string, error) {
	var engineErrors []error
	for _, e := range engines {
		// 命中缓存直接返回，避免重复调用
		if v, ok := a.detectCache.get(cacheKey(e, text)); ok {
			return v, nil
		}
		var (
			lang string
			err  error
		)
		if e == "aliyun" {
			lang, err = translate.GetDetectLanguage(text)
		} else {
			lang, err = translate.DetectLanguage(text)
		}
		if err == nil && lang != "" {
			a.detectCache.set(cacheKey(e, text), lang)
			return lang, nil
		}
		if err != nil {
			engineErrors = append(engineErrors, fmt.Errorf("%s: %w", e, err))
		} else {
			engineErrors = append(engineErrors, fmt.Errorf("%s: 未返回识别结果", e))
		}
	}
	return "", errors.Join(engineErrors...)
}

// engineTimeout 单引擎翻译最大等待时间。
// 与 translate 包的 HTTP/SDK 超时保持一致，确保超时后底层调用也会自然退出，
// 不会因外层 select 提前返回而遗留 goroutine。
const engineTimeout = 30 * time.Second

// TranslateMulti 多引擎并发翻译：同一句话同时走多个引擎，返回并排结果。
// 单引擎超时 engineTimeout，超时的引擎返回 "翻译超时" 错误而非阻塞整体。
// 不再返回 Go error：识别失败时为每个引擎填充结构化错误，确保前端总能拿到 errorCode。
func (a *App) TranslateMulti(text string, source string, target string, engines []string) MultiTranslateResult {
	engines = normalizeEngines(engines)
	src := source
	emptyRes := MultiTranslateResult{Source: source, Target: target, Results: map[string]EngineTranslateResult{}}
	if strings.TrimSpace(text) == "" {
		err := newTranslateError(ErrCodeInvalidInput, "", "输入文本为空", nil)
		emptyRes.Results = engineErrorResults(engines, err)
		return emptyRes
	}

	// 自动识别一次，供所有引擎共用
	if src == "" || src == "auto" {
		detected, err := a.detectLanguageBestEffort(text, engines)
		if err != nil {
			te := classifyError("", err)
			te.Message = "语言识别失败：" + te.Message
			// 为所有引擎填充同一错误，前端按 errorCode 显示
			emptyRes.Results = engineErrorResults(engines, te)
			return emptyRes
		}
		src = detected
	}

	tgt := fallbackTarget(src, target)

	results := make(map[string]EngineTranslateResult, len(engines))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, engine := range engines {
		engine := engine
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := EngineTranslateResult{Engine: engine}

			// 命中缓存直接用，跳过 API 调用
			ck := cacheKey(engine, src, tgt, text)
			if v, ok := a.translateCache.get(ck); ok {
				r.Text = v
				mu.Lock()
				results[engine] = r
				mu.Unlock()
				return
			}

			type outcome struct {
				text string
				err  error
			}
			done := make(chan outcome, 1)
			go func() {
				var (
					out string
					err error
				)
				out, err = translateWithEngine(engine, text, src, tgt)
				done <- outcome{out, err}
			}()

			select {
			case o := <-done:
				r.Text = o.text
				if o.err != nil {
					te := wrapTranslateError(engine, o.err)
					r.Error = te.Message
					r.ErrorCode = te.Code
				} else {
					// 仅缓存成功结果
					a.translateCache.set(ck, o.text)
				}
			case <-time.After(engineTimeout):
				// 底层 HTTP/SDK 已设 30s 超时，内层 goroutine 会自然退出，无需显式取消
				te := newTranslateError(ErrCodeTimeout, engine, "翻译超时，请稍后重试", nil)
				r.Error = te.Message
				r.ErrorCode = te.Code
			}

			mu.Lock()
			results[engine] = r
			mu.Unlock()

			// 流式推送：单引擎完成后立即触发事件，前端可独立渲染与 loading
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "translate:engine-result", r)
			}
		}()
	}
	wg.Wait()

	res := MultiTranslateResult{
		Source:  source,
		AutoSrc: src,
		Target:  tgt,
		Results: results,
	}
	return res
}

func engineErrorResults(engines []string, err *TranslateError) map[string]EngineTranslateResult {
	results := make(map[string]EngineTranslateResult, len(engines))
	for _, engine := range engines {
		results[engine] = EngineTranslateResult{
			Engine:    engine,
			Error:     err.Message,
			ErrorCode: err.Code,
		}
	}
	return results
}

// TestConnection 用一次最小请求验证指定引擎的凭据是否可用
func (a *App) TestConnection(engine string) error {
	engine = strings.ToLower(strings.TrimSpace(engine))
	probe := "hello"
	switch engine {
	case "aliyun":
		_, err := translate.GetDetectLanguage(probe)
		if err != nil {
			return fmt.Errorf("阿里云连接测试失败: %v", err)
		}
		return nil
	case "tencent", "":
		_, err := translate.DetectLanguage(probe)
		if err != nil {
			return fmt.Errorf("混元连接测试失败: %v", err)
		}
		return nil
	default:
		return fmt.Errorf("未知引擎: %s", engine)
	}
}

// getHistoryPath 返回历史记录文件路径
func (a *App) getHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".simple_translate")
	_ = os.MkdirAll(dir, 0700)
	return filepath.Join(dir, "history.json")
}

// LoadHistory 读取本地持久化的历史记录
func (a *App) LoadHistory() ([]HistoryEntry, error) {
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
	return entries, nil
}

// SaveHistory 将历史记录写入本地文件（上限 200 条，0600 权限）
func (a *App) SaveHistory(entries []HistoryEntry) error {
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
