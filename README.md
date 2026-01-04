## 简介

SimpleTranslate 是一个基于 Wails 框架开发的桌面翻译应用。后端使用 Go 语言，前端使用 Svelte 框架。支持腾讯云和阿里云的翻译服务，提供简洁的用户界面和高效的翻译体验。

![alt text](./.github/screenshots/image.png)
![alt text](./.github/screenshots/image-1.png)

## 特性

- 支持多种语言翻译（中文、英文、日语、韩语、法语、德语、俄语、西班牙语等）
- 自动语言检测
- 支持腾讯云和阿里云翻译服务
- 桌面应用，无需浏览器
- 简洁美观的用户界面
- 历史记录功能
- 深色模式支持
- 自定义配置

## 安装

### 前置要求

- Go 1.23 或更高版本
- Node.js 和 npm
- Wails CLI

### 克隆项目

```bash
git clone https://github.com/yourusername/simpleTranslate.git
cd simpleTranslate
```

### 安装依赖

```bash
# 安装 Go 依赖
go mod tidy

# 安装前端依赖
cd frontend
npm install
cd ..
```

## 使用

### 开发模式

运行开发模式，支持热重载：

```bash
wails dev
```

### 生产构建

构建不同平台的二进制文件：

```bash
# Windows AMD64
wails build -clean -o simpleTranslate-win-amd64.exe -platform windows/amd64

# Windows ARM64
wails build -clean -o simpleTranslate-win-arm64.exe -platform windows/arm64

# macOS AMD64
wails build -clean -o simpleTranslate-mac-amd64 -platform darwin/amd64

# macOS ARM64
wails build -clean -o simpleTranslate-mac-arm64 -platform darwin/arm64
```

### 配置

首次运行时，需要配置云服务 API 密钥：

1. 打开应用设置
2. 选择翻译服务（腾讯云或阿里云）
3. 输入相应的 SecretId 和 SecretKey
4. 可选：设置区域（Region）

配置存储在 `~/.simple_translate/config.json`

## 项目结构

```
simpleTranslate/
├── app.go              # 主应用逻辑
├── config.go           # 配置相关方法
├── main.go             # 应用入口
├── translate/          # 翻译服务包
│   ├── aliyun.go       # 阿里云翻译
│   └── tencent.go      # 腾讯云翻译
├── config/             # 配置工具包
├── frontend/           # 前端代码
│   ├── src/
│   │   ├── App.svelte  # 主界面
│   │   └── lib/
│   └── package.json
├── build.md            # 构建说明
└── wails.json          # Wails 配置
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License