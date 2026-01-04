# Copilot Instructions for SimpleTranslate

## Architecture Overview
This is a Wails-based desktop translation app with Go backend and Svelte frontend. The app integrates multiple cloud translation services (Tencent Cloud and Alibaba Cloud) for text translation.

- **Backend (Go)**: Handles translation logic, config management, and API calls to cloud services
- **Frontend (Svelte)**: Provides UI for text input/output, language selection, and settings
- **Communication**: Wails binds Go methods to frontend via generated `wailsjs/go/main/App.js`

## Key Components
- `app.go`: Main app struct with `TranslateText()` method - unified entry for translations
- `translate/`: Package with `aliyun.go` and `tencent.go` for different providers
- `config/config.go`: Utility functions for reading/writing user config from `~/.simple_translate/config.json`
- `frontend/src/App.svelte`: Main UI component handling translation workflow and settings
- `frontend/src/lib/store.js`: Svelte store for reactive config state

## Development Workflow
- **Dev Mode**: Run `wails dev -tags webkit2_41` for hot-reload development
- **Build**: Use `wails build -clean -platform <target>` with webkit tags for production
- **Frontend**: `npm run build` in `frontend/` generates assets embedded in Go binary
- **Config**: User settings stored in `~/.simple_translate/config.json` with `CloudConfig` struct

## Patterns & Conventions
- **Translation Engines**: Pass `"aliyun"` or default (Tencent) to `TranslateText()` for provider selection
- **Language Detection**: Automatic if `source="auto"`; falls back to detected language
- **Config Access**: Use `config.GetConfig(config.GetConfigPath())` for cloud credentials
- **Error Handling**: Return errors from translation calls; frontend shows status messages
- **UI State**: Use `configStore` for reactive updates; persist via `updateAndSaveConfig()`
- **Build Tags**: Include `-tags webkit2_41` for compatibility with older WebKit versions

## Integration Points
- **Cloud APIs**: Tencent TMT and Alibaba Cloud MT services require API keys in config
- **Wails Binding**: Methods on `App` struct are auto-exposed to frontend
- **Asset Embedding**: `//go:embed frontend/dist` includes built frontend in binary

## Common Tasks
- Adding new translation provider: Create new file in `translate/`, implement `Translate()` and `DetectLanguage()`, update `app.go` logic
- UI changes: Modify `App.svelte`, use `wailsjs` for backend calls
- Config updates: Edit `CloudConfig` struct, update both `config.go` and `config/config.go` consistently