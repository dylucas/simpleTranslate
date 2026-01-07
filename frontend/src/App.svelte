<script>
  import { TranslateText, GetConfig } from "../wailsjs/go/main/App";
  import {
    Languages,
    ArrowLeftRight,
    History as HistoryIcon,
    Sun,
    Moon,
    Copy,
    Check,
    Keyboard,
    X,
    Settings,
    PanelLeftClose, // 使用更语义化的图标
  } from "lucide-svelte";
  import Config from "./lib/Config.svelte";
  import History from "./lib/History.svelte";
  import { onMount } from "svelte";
  import { fade } from "svelte/transition";
  import { configStore, initConfig, updateAndSaveConfig } from "./lib/store";

  // --- 状态控制增强 ---
  let isProcessing = false; // 并发锁

  // --- 核心状态 ---
  let input = "";
  let output = "";
  let source = "auto";
  let target = "zh";
  let status = "准备就绪";
  let copied = false;

  // --- 界面控制 ---
  let showConfig = false;
  let showHistory = false;
  let history = [];
  let autoDetectLang = "自动识别";

  const langs = {
    zh: "中文",
    en: "英文",
    jp: "日语",
    kr: "韩语",
    fr: "法语",
    de: "德语",
    ru: "俄语",
    es: "西语",
  };

  // 模拟持久化历史记录
  $: {
    if (history.length > 0) {
      localStorage.setItem(
        "translate_history",
        JSON.stringify(history.slice(0, 50)),
      );
    }
  }

  // 从 Store 响应式获取配置
  let currentConfig;
  configStore.subscribe((value) => {
    currentConfig = value;
  });

  // 这里的 activeEngine 和 isDark 直接引用 currentConfig
  $: isDark = currentConfig?.isDark ?? true;
  $: sidebarCollapsed = currentConfig?.sidebarCollapsed ?? false;
  $: activeEngine = currentConfig?.defaultEngine || "tencent";

  // 切换主题的函数
  function toggleTheme() {
    updateAndSaveConfig("isDark", !currentConfig.isDark);
  }

  // 切换侧边栏并保存状态
  function toggleSidebar() {
    updateAndSaveConfig("sidebarCollapsed", !sidebarCollapsed);
  }

  onMount(async () => {
    await initConfig();
    const savedHistory = localStorage.getItem("translate_history");
    if (savedHistory) history = JSON.parse(savedHistory);
  });

  async function translate() {
    if (!input.trim() || isProcessing) return;
    isProcessing = true;
    status = "翻译中...";
    try {
      const res = await TranslateText(input, source, target, activeEngine);
      output = res.text;
      if (source === "auto") {
        let detected = langs[res.autoSrc] || res.autoSrc;
        autoDetectLang = `自动 (${detected})`;
      }
      status = "完成";
      // 添加到历史记录
      addHistory(input, output, source, target);
    } catch (e) {
      status = "翻译失败";
      console.error(e);
    }
    isProcessing = false;
  }

  function addHistory(input, output, src, tgt) {
    const entry = {
      id: Date.now(),
      input,
      output,
      source: src,
      target: tgt,
      time: new Date().toLocaleTimeString([], {
        hour: "2-digit",
        minute: "2-digit",
      }),
    };
    history = [entry, ...history];
  }

  function clearHistory() {
    history = []; // 清空历史记录
    localStorage.removeItem("translate_history"); // 清空本地存储的历史记录
  }

  function handleCopy() {
    if (!output) return;
    navigator.clipboard.writeText(output);
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }

  /**
   * @param {{ ctrlKey: any; metaKey: any; key: string; preventDefault: () => void; }} e
   */
  function handleGlobalKeydown(e) {
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      translate();
    }
    if (e.key === "Escape" && showHistory) {
      showHistory = false;
    }
  }

  function handleHistorySelect(event) {
    const item = event.detail;
    input = item.input;
    output = item.output;
    showHistory = false;
  }

  function handleHistoryClose() {
    showHistory = false;
  }

  function handleHistoryClear() {
    clearHistory();
  }

  // 监视配置窗口的关闭
  $: if (!showConfig) {
    refreshConfig();
  }

  // 抽离出更新配置的逻辑
  async function refreshConfig() {
    const cfg = await GetConfig();
    if (cfg?.defaultEngine) {
      activeEngine = cfg.defaultEngine;
    }
  }
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="app-shell" class:light-mode={!isDark}>
  <aside class="sidebar" class:collapsed={sidebarCollapsed}>
    <div class="sidebar-header">
      {#if !sidebarCollapsed}
        <div class="brand" transition:fade={{ duration: 150 }}>
          <div class="brand-icon"><Languages size={22} /></div>
          <span>Translate</span>
        </div>
      {/if}

      <button
        class="collapse-toggle"
        class:centered={sidebarCollapsed}
        on:click={toggleSidebar}
        title={sidebarCollapsed ? "展开" : "收起"}
      >
        <div class="icon-wrapper" class:rotated={sidebarCollapsed}>
          <PanelLeftClose size={18} />
        </div>
      </button>
    </div>

    <nav class="side-nav">
      <button
        class="nav-item"
        on:click={() => (showHistory = true)}
        title="历史记录"
      >
        <div class="nav-icon"><HistoryIcon size={20} /></div>
        {#if !sidebarCollapsed}
          <span class="nav-text" transition:fade={{ duration: 100 }}
            >历史记录</span
          >
        {/if}
      </button>
    </nav>

    <div class="sidebar-footer">
      {#if !sidebarCollapsed}
        <div class="engine-box" transition:fade={{ duration: 100 }}>
          <!-- svelte-ignore a11y-label-has-associated-control -->
          <label>翻译引擎</label>
          <div class="engine-pills">
            <button
              class:active={activeEngine === "tencent"}
              on:click={() => updateAndSaveConfig("defaultEngine", "tencent")}
              >腾讯</button
            >
            <button
              class:active={activeEngine === "aliyun"}
              on:click={() => updateAndSaveConfig("defaultEngine", "aliyun")}
              >阿里</button
            >
          </div>
        </div>
      {/if}

      <div class="bottom-tools" class:column-layout={sidebarCollapsed}>
        <button
          class="tool-btn"
          on:click={() => updateAndSaveConfig("isDark", !isDark)}
          title="切换主题"
        >
          {#if isDark}<Sun size={18} />{:else}<Moon size={18} />{/if}
        </button>
        <button
          class="tool-btn"
          on:click={() => (showConfig = true)}
          title="设置"
        >
          <Settings size={18} />
        </button>
      </div>
    </div>
  </aside>

  <main class="main-content">
    <header class="workspace-header">
      <div class="lang-bar">
        <div class="select-wrapper">
          <select
            bind:value={source}
            style={isDark ? "background: #1e1e1e;" : ""}
          >
            <option value="auto">{autoDetectLang}</option>
            {#each Object.entries(langs) as [code, name]}
              <option value={code}>{name}</option>
            {/each}
          </select>
        </div>

        <button
          class="swap-btn"
          on:click={() => ([source, target] = [target, source])}
          title="交换语言"
        >
          <ArrowLeftRight size={16} />
        </button>

        <div class="select-wrapper">
          <select
            bind:value={target}
            style={isDark ? "background: #1e1e1e;" : ""}
          >
            {#each Object.entries(langs) as [code, name]}
              <option value={code}>{name}</option>
            {/each}
          </select>
        </div>
      </div>

      <button
        class="translate-btn"
        on:click={translate}
        disabled={status === "翻译中..."}
      >
        <span>{status === "翻译中..." ? "翻译中" : "翻译"}</span>
        {#if status === "翻译中..."}
          <span class="loading-dots">...</span>
        {/if}
      </button>
    </header>

    <div class="editor-container">
      <section class="editor-pane source">
        <textarea
          bind:value={input}
          placeholder="在此输入要翻译的文本..."
          spellcheck="false"
        ></textarea>
        <div class="pane-footer">
          <span class="char-count">{input.length} 字符</span>
          {#if input}
            <button class="clear-btn" on:click={() => (input = "")}
              ><X size={12} /> 清空</button
            >
          {/if}
        </div>
      </section>

      <section class="editor-pane result">
        <textarea
          readonly
          value={output}
          placeholder="翻译结果..."
          spellcheck="false"
        ></textarea>
        <div class="pane-footer">
          {#if output}
            <button
              class="action-btn copy"
              on:click={handleCopy}
              class:success={copied}
            >
              {#if copied}<Check size={14} />{:else}<Copy size={14} />{/if}
              {copied ? "已复制" : "复制"}
            </button>
          {/if}
        </div>
      </section>
    </div>

    <footer class="app-status-bar">
      <div class="status-item">
        <span
          class="status-dot"
          class:processing={status === "翻译中..."}
          class:done={status === "完成"}
          class:error={status === "翻译失败"}
        ></span>
        {status}
      </div>
      <div class="status-item shortcut-hint">
        <Keyboard size={12} /> <span>Ctrl + Enter 发送</span>
      </div>
    </footer>
  </main>

  <History
    bind:show={showHistory}
    {history}
    on:select={handleHistorySelect}
    on:close={handleHistoryClose}
    on:clear={handleHistoryClear}
  />

  <Config bind:show={showConfig} {isDark} />
</div>

<style>
  :root {
    --bg-base: #121212;
    --bg-sidebar: #181818;
    --bg-surface: #1e1e1e;
    --bg-hover: #2a2a2a;
    --border: #333;
    --primary: #3b82f6;
    --primary-hover: #2563eb;
    --text-main: #e5e5e5;
    --text-sec: #a3a3a3;
    --shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.3),
      0 2px 4px -1px rgba(0, 0, 0, 0.16);
    --sidebar-w: 240px;
    --sidebar-collapsed-w: 72px;
  }

  .light-mode {
    --bg-base: #ffffff;
    --bg-sidebar: #f8f9fa;
    --bg-surface: #ffffff;
    --bg-hover: #e9ecef;
    --border: #e5e7eb;
    --primary: #2563eb;
    --primary-hover: #1d4ed8;
    --text-main: #1f2937;
    --text-sec: #6b7280;
    --shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1),
      0 2px 4px -1px rgba(0, 0, 0, 0.06);
  }

  /* 全局重置 */
  * {
    box-sizing: border-box;
  }

  .app-shell {
    display: flex;
    height: 100vh;
    background: var(--bg-base);
    color: var(--text-main);
    font-family:
      "Inter",
      -apple-system,
      BlinkMacSystemFont,
      "Segoe UI",
      Roboto,
      sans-serif;
    overflow: hidden;
  }

  /* --- 侧边栏 --- */
  .sidebar {
    width: var(--sidebar-w);
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    transition: width 0.3s cubic-bezier(0.2, 0, 0, 1);
    z-index: 10;
    flex-shrink: 0;
    backdrop-filter: blur(10px); /* 增加毛玻璃 */
    -webkit-backdrop-filter: blur(10px);
  }

  .sidebar.collapsed {
    width: var(--sidebar-collapsed-w);
  }

  .sidebar-header {
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: space-between; /* 默认两端对齐 */
    padding: 0 16px;
    position: relative;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 10px;
    font-weight: 700;
    font-size: 16px;
    color: var(--text-main);
    white-space: nowrap;
    overflow: hidden;
  }
  .brand-icon {
    color: var(--primary);
  }

  /* 切换按钮样式 */
  .collapse-toggle {
    background: transparent;
    border: none;
    color: var(--text-sec);
    width: 32px;
    height: 32px;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s;
  }
  .collapse-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  /* 当侧边栏折叠时，按钮居中显示 */
  .collapse-toggle.centered {
    margin: 0 auto;
    width: 100%;
  }

  .icon-wrapper {
    display: flex;
    transition: transform 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  /* 图标旋转动画：利用 CSS Transform 翻转 */
  .icon-wrapper.rotated {
    transform: rotate(180deg);
  }

  .side-nav {
    flex: 1;
    padding: 16px 12px;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    padding: 10px;
    background: transparent;
    border: none;
    border-radius: 8px;
    color: var(--text-sec);
    cursor: pointer;
    transition: all 0.2s;
    height: 44px;
    width: 100%;
    text-align: left;
  }
  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .nav-icon {
    display: flex;
    justify-content: center;
    align-items: center;
    min-width: 24px; /* 确保图标位置固定 */
  }
  .nav-text {
    margin-left: 12px;
    font-size: 14px;
    font-weight: 500;
    white-space: nowrap;
  }

  .sidebar-footer {
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: 16px;
    padding: 16px 12px;
    transition: padding 0.3s;
  }

  .engine-box label {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-sec);
    margin-bottom: 8px;
    display: block;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .engine-pills {
    display: flex;
    background: var(--bg-hover);
    border-radius: 6px;
    padding: 3px;
  }
  .engine-pills button {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--text-sec);
    font-size: 12px;
    padding: 6px;
    border-radius: 4px;
    cursor: pointer;
    transition: 0.2s;
  }
  .engine-pills button.active {
    background: var(--bg-surface);
    color: var(--text-main);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
    font-weight: 500;
  }

  .bottom-tools {
    display: flex;
    gap: 8px;
    justify-content: space-between;
    /* transition: all 0.3s cubic-bezier(0.2, 0, 0, 1); */
    transition: all 0.3s;
  }

  /* 引擎选择框在折叠时的过渡 */
  .engine-box {
    overflow: hidden;
    transition:
      max-height 0.3s,
      opacity 0.2s;
  }

  .sidebar.collapsed .sidebar-footer {
    padding: 16px 0; /* 折叠时取消左右内边距，方便图标居中 */
  }

  /* 当侧边栏折叠时，改变工具栏布局 */
  .sidebar.collapsed .bottom-tools {
    flex-direction: column; /* 改为垂直排列 */
    align-items: center; /* 居中对齐 */
    gap: 12px; /* 增加垂直间距 */
  }

  .tool-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-sec);
    padding: 8px;
    border-radius: 8px;
    cursor: pointer;
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center; /* 确保图标在按钮内部绝对居中 */
    transition: 0.2s;
    min-width: 0; /* 防止折叠时撑开容器 */
  }

  .sidebar.collapsed .tool-btn {
    width: 40px; /* 给一个固定的宽度 */
    height: 40px;
    flex: none; /* 取消 flex: 1 */
  }
  .tool-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
    border-color: var(--border);
  }

  /* --- 主内容 --- */
  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: var(--bg-surface);
    position: relative;
    z-index: 1;
  }

  .workspace-header {
    height: 64px;
    padding: 0 24px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
  }

  .lang-bar {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .select-wrapper {
    position: relative;
  }
  .select-wrapper select {
    appearance: none;
    background: transparent;
    border: none;
    color: var(--text-main);
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    padding: 8px 12px;
    border-radius: 6px;
    outline: none;
  }
  .select-wrapper select:hover {
    background: var(--bg-hover);
  }

  .swap-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    padding: 8px;
    border-radius: 50%;
    cursor: pointer;
    transition: 0.2s;
    display: flex;
  }
  .swap-btn:hover {
    background: var(--bg-hover);
    color: var(--primary);
    transform: rotate(180deg);
  }

  .translate-btn {
    background: linear-gradient(135deg, var(--primary), var(--primary-hover));
    color: white;
    border: none;
    padding: 8px 24px;
    border-radius: 20px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    box-shadow: 0 4px 10px rgba(59, 130, 246, 0.3);
    transition:
      transform 0.1s,
      box-shadow 0.2s;
  }
  .translate-btn:hover {
    box-shadow: 0 6px 14px rgba(59, 130, 246, 0.4);
  }
  .translate-btn:active {
    transform: scale(0.96);
  }
  .translate-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .editor-container {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .editor-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: 24px;
    position: relative;
    transition: background 0.3s;
  }
  .editor-pane.source {
    border-right: 1px solid var(--border);
  }
  .editor-pane.result {
    background: var(--bg-base); /* 结果区稍微深一点/浅一点区分 */
  }

  textarea {
    flex: 1;
    background: transparent;
    border: none;
    resize: none;
    outline: none;
    font-size: 18px;
    line-height: 1.6;
    color: var(--text-main);
    padding: 0;
  }
  textarea::placeholder {
    color: var(--text-sec);
    opacity: 0.5;
  }

  .pane-footer {
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: 10px;
    margin-top: 10px;
  }
  .char-count {
    font-size: 12px;
    color: var(--text-sec);
    margin-right: auto;
  }

  .clear-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .clear-btn:hover {
    color: var(--text-main);
  }

  .action-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: 4px 10px;
    border-radius: 6px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: 6px;
    transition: all 0.2s;
  }
  .action-btn:hover {
    border-color: var(--text-sec);
    background: var(--bg-hover);
  }
  .action-btn.success {
    border-color: #10b981;
    color: #10b981;
    background: rgba(16, 185, 129, 0.1);
  }

  /* 状态栏 */
  .app-status-bar {
    height: 32px;
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    font-size: 11px;
    color: var(--text-sec);
    background: var(--bg-sidebar);
  }
  .status-item {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: #10b981; /* Ready Green */
  }
  .status-dot.processing {
    background: #f59e0b;
    animation: pulse 1s infinite;
  }
  .status-dot.error {
    background: #ef4444;
  }

  @keyframes pulse {
    0% {
      opacity: 1;
    }
    50% {
      opacity: 0.4;
    }
    100% {
      opacity: 1;
    }
  }
</style>
