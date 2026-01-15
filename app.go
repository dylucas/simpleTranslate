package main

import (
	"context"
	"fmt"
	"simpleTranslate/translate"
	"sort"
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
	Source     string                           `json:"source"`
	AutoSrc    string                           `json:"autoSrc"`
	Target     string                           `json:"target"`
	BestEngine string                           `json:"bestEngine,omitempty"`
	BestText   string                           `json:"bestText,omitempty"`
	Results    map[string]EngineTranslateResult `json:"results"`
}

// 给前端调用的统一入口
func (a *App) TranslateText(text string, source string, target string, engine string) (*TranslateResult, error) {
	src := source

	// 自动识别
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

func pickBestByMedianLength(engineOrder []string, results map[string]EngineTranslateResult) (bestEngine, bestText string) {
	type item struct {
		engine string
		n      int
		text   string
	}
	items := make([]item, 0, len(engineOrder))
	for _, e := range engineOrder {
		r, ok := results[e]
		if !ok || r.Error != "" || strings.TrimSpace(r.Text) == "" {
			continue
		}
		items = append(items, item{engine: e, n: len([]rune(r.Text)), text: r.Text})
	}
	if len(items) == 0 {
		return "", ""
	}
	sort.Slice(items, func(i, j int) bool { return items[i].n < items[j].n })
	medianN := items[len(items)/2].n

	// Tie-break by original engine order (stable preference).
	for _, e := range engineOrder {
		for _, it := range items {
			if it.engine == e && it.n == medianN {
				return it.engine, it.text
			}
		}
	}
	return items[0].engine, items[0].text
}

// 多引擎并发翻译：同一句话同时走多个引擎，返回并排结果；可选“最佳结果”(简单融合/选择)。
func (a *App) TranslateMulti(text string, source string, target string, engines []string, pickBest bool) (*MultiTranslateResult, error) {
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
	if pickBest {
		res.BestEngine, res.BestText = pickBestByMedianLength(engines, results)
	}
	return res, nil
}
