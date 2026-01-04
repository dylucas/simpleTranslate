package main

import (
	"context"
	"simpleTranslate/translate"
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
