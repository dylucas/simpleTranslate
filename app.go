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
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
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

// 给前端调用的统一入口
func (a *App) TranslateText(text string, source string, target string, engine string) (*TranslateResult, error) {
	src := source

	// 自动识别语种
	if src == "" || src == "auto" {
		detected := ""
		var err error
		if engine == "aliyun" {
			detected, err = translate.GetDetectLanguage(text)
		} else {
			detected, err = translate.DetectLanguage(text)

		}
		if err == nil {
			src = detected
		} else {
			return nil, err
		}
	}

	// 目标语言兜底
	if src == target {
		switch src {
		case "zh":
			target = "en"
		case "en":
			target = "zh"
		default:
			target = "en"
		}
	}

	result := ""
	var err error
	if engine == "aliyun" {
		result, err = translate.TranslateGeneral(text, src, target)
	} else {
		result, err = translate.Translate(text, src, target)
	}

	// result, err := translate.Translate(text, src, target)
	if err != nil {
		return nil, err
	}

	return &TranslateResult{
		Source:  source,
		AutoSrc: src,
		Target:  target,
		Text:    result,
	}, nil
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

func detectLanguageBestEffort(text string, engines []string) (string, error) {
	// Try in given order; prefer aliyun if present because it has dedicated API.
	for _, e := range engines {
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
			return lang, nil
		}
	}
	return "", fmt.Errorf("语言识别失败")
}

// 多引擎并发翻译：同一句话同时走多个引擎，返回并排结果
func (a *App) TranslateMulti(text string, source string, target string, engines []string) (*MultiTranslateResult, error) {
	engines = normalizeEngines(engines)
	src := source

	// 自动识别一次，供所有引擎共用
	if src == "" || src == "auto" {
		detected, err := detectLanguageBestEffort(text, engines)
		if err != nil {
			return nil, err
		}
		src = detected
	}

	// 目标语言兜底
	tgt := target
	if src == tgt {
		switch src {
		case "zh":
			tgt = "en"
		case "en":
			tgt = "zh"
		default:
			tgt = "en"
		}
	}

	results := make(map[string]EngineTranslateResult, len(engines))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, engine := range engines {
		engine := engine
		wg.Add(1)
		go func() {
			defer wg.Done()
			var (
				out string
				err error
			)
			if engine == "aliyun" {
				out, err = translate.TranslateGeneral(text, src, tgt)
			} else {
				out, err = translate.Translate(text, src, tgt)
			}

			r := EngineTranslateResult{Engine: engine, Text: out}
			if err != nil {
				r.Error = err.Error()
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
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".simple_translate")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.MkdirAll(dir, 0755)
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
