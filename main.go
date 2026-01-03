package main

import (
	"embed"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
)

//go:embed frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:  "简单翻译",
		Width:  1000,
		Height: 600,
		Assets: assets,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}
