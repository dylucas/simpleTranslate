package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"simpleTranslate/translate"
	"strings"
	"sync"
	"time"
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
}

type EngineTranslateResult struct {
	Engine string `json:"engine"`
	Text   string `json:"text"`
	Error  string `json:"error,omitempty"`
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

// TranslateText 给前端调用的统一入口
func (a *App) TranslateText(text string, source string, target string, engine string) (*TranslateResult, error) {
	src := source

	// 自动识别语种（与 TranslateMulti 保持一致：best-effort 跨引擎兜底）
	if src == "" || src == "auto" {
		// 优先使用当前引擎；失败时按引擎列表兜底
		engines := []string{engine}
		if engine != "tencent" && engine != "aliyun" {
			engines = []string{"tencent", "aliyun"}
		} else {
			// 把另一个引擎追加为兜底
			other := "aliyun"
			if engine == "aliyun" {
				other = "tencent"
			}
			engines = append(engines, other)
		}
		detected, err := a.detectLanguageBestEffort(text, engines)
		if err != nil {
			return nil, err
		}
		src = detected
	}

	// 目标语言兜底
	tgt := fallbackTarget(src, target)

	// 命中缓存直接返回，避免重复调用 API
	ck := cacheKey(engine, src, tgt, text)
	if v, ok := a.translateCache.get(ck); ok {
		return &TranslateResult{
			Source:  source,
			AutoSrc: src,
			Target:  tgt,
			Text:    v,
		}, nil
	}

	result := ""
	var err error
	if engine == "aliyun" {
		result, err = translate.TranslateGeneral(text, src, tgt)
	} else {
		result, err = translate.Translate(text, src, tgt)
	}
	if err != nil {
		return nil, err
	}

	// 写入缓存供下次命中
	a.translateCache.set(ck, result)

	return &TranslateResult{
		Source:  source,
		AutoSrc: src,
		Target:  tgt,
		Text:    result,
	}, nil
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
	}
	return "", fmt.Errorf("语言识别失败")
}

// engineTimeout 单引擎翻译最大等待时间。
// 与 translate 包的 HTTP/SDK 超时保持一致，确保超时后底层调用也会自然退出，
// 不会因外层 select 提前返回而遗留 goroutine。
const engineTimeout = 30 * time.Second

// TranslateMulti 多引擎并发翻译：同一句话同时走多个引擎，返回并排结果。
// 单引擎超时 engineTimeout，超时的引擎返回 "翻译超时" 错误而非阻塞整体。
func (a *App) TranslateMulti(text string, source string, target string, engines []string) (*MultiTranslateResult, error) {
	engines = normalizeEngines(engines)
	src := source

	// 自动识别一次，供所有引擎共用
	if src == "" || src == "auto" {
		detected, err := a.detectLanguageBestEffort(text, engines)
		if err != nil {
			return nil, err
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
				if engine == "aliyun" {
					out, err = translate.TranslateGeneral(text, src, tgt)
				} else {
					out, err = translate.Translate(text, src, tgt)
				}
				done <- outcome{out, err}
			}()

			select {
			case o := <-done:
				r.Text = o.text
				if o.err != nil {
					r.Error = o.err.Error()
				} else {
					// 仅缓存成功结果
					a.translateCache.set(ck, o.text)
				}
			case <-time.After(engineTimeout):
				// 底层 HTTP/SDK 已设 30s 超时，内层 goroutine 会自然退出，无需显式取消
				r.Error = "翻译超时"
			}

			mu.Lock()
			results[engine] = r
			mu.Unlock()
		}()
	}
	wg.Wait()

	res := &MultiTranslateResult{
		Source:  source,
		AutoSrc: src,
		Target:  tgt,
		Results: results,
	}
	return res, nil
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
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.MkdirAll(dir, 0755)
	}
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
	return os.WriteFile(path, data, 0600)
}
