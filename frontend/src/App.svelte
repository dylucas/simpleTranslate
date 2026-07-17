<script lang="ts">
  import { LoadHistory, SaveHistory, GetConfig } from "../wailsjs/go/main/App";
  import { AlertCircle, Volume2, Square, X, CornerDownLeft, Copy, Check } from "lucide-svelte";
  import { onMount, onDestroy } from "svelte";
  import { fade } from "svelte/transition";
  import { configStore, initConfig, updateAndSaveConfig } from "./lib/store";
  import { ClipboardGetText } from "../wailsjs/runtime/runtime";
  import { langs, getSpeechLang, DEFAULT_SOURCE, DEFAULT_TARGET, DEFAULT_COMPARE_ENGINES } from "./lib/languages";
  import { createClipboardWatcher } from "./lib/clipboard";
  import { createShortcutHandler } from "./lib/shortcuts";
  import { createSpeaker } from "./lib/speech";
  import { createTranslateController } from "./lib/translateController";
  import type { HistoryEntry } from "./lib/types";

  import ErrorToast from "./lib/ErrorToast.svelte";
  import Sidebar from "./lib/Sidebar.svelte";
  import LanguageBar from "./lib/LanguageBar.svelte";
  import ComparePanel from "./lib/ComparePanel.svelte";
  import StatusBar from "./lib/StatusBar.svelte";
  import Config from "./lib/Config.svelte";
  import History from "./lib/History.svelte";

  // --- UI 状态 ---
  let input = "";
  let source = DEFAULT_SOURCE;
  let target = DEFAULT_TARGET;
  let showConfig = false;
  let showHistory = false;
  let history: HistoryEntry[] = [];
  let autoTranslate = true;
  let suppressAuto = false;
  let copied = false;
  let copiedEngines: Record<string, boolean> = {};
  let speakingText: string | null = null;
  let inputEl: HTMLTextAreaElement;
  let historySaveTimer: ReturnType<typeof setTimeout> | null = null;
  let clipboardWatch = false;

  // --- 翻译控制器 ---
  // 封装翻译流程的状态与逻辑，通过依赖注入读取宿主状态
  const ctrl = createTranslateController({
    getInput: () => input,
    getSource: () => source,
    getTarget: () => target,
    getActiveEngine: () => $configStore?.defaultEngine || "tencent",
    getCompareMode: () => !!($configStore?.compareMode ?? false),
    getCompareEngines: () =>
      Array.isArray($configStore?.compareEngines) && $configStore.compareEngines.length
        ? $configStore.compareEngines
        : [...DEFAULT_COMPARE_ENGINES],
    getHistory: () => history,
    setHistory: (updater) => {
      history = updater(history);
    },
    setTarget: (t) => {
      target = t;
    },
  });

  // 响应式读取控制器状态
  const ctrlState = ctrl.state;
  $: ts = $ctrlState;
  $: isProcessing = ts.isProcessing;
  $: output = ts.output;
  $: status = ts.status;
  $: errorToast = ts.errorToast;
  $: compareOutputs = ts.compareOutputs;
  $: compareLoadingEngines = ts.compareLoadingEngines;
  $: autoDetectLang = ts.autoDetectLang;

  // --- 配置 Store 派生 ---
  $: isDark = $configStore?.isDark ?? true;
  $: sidebarCollapsed = $configStore?.sidebarCollapsed ?? false;
  $: activeEngine = $configStore?.defaultEngine || "tencent";
  $: compareMode = !!($configStore?.compareMode ?? false);
  $: compareEngines = Array.isArray($configStore?.compareEngines) && $configStore.compareEngines.length
    ? $configStore.compareEngines
    : [...DEFAULT_COMPARE_ENGINES];

  $: apiKeyMissing = (() => {
    const cfg = $configStore;
    if (!cfg) return false;
    if (activeEngine === "aliyun") {
      return !cfg.aliyun?.secretId || !cfg.aliyun?.secretKey;
    }
    return !cfg.tencent?.secretKey;
  })();

  $: clipboardWatch = !!($configStore?.clipboardWatch ?? false);

  // --- 语音朗读 ---
  const speaker = createSpeaker(getSpeechLang);

  function handleSpeak(text: string, langCode: string): void {
    if (!text) return;
    speaker.speak(text, langCode, {
      onChange: (cur) => (speakingText = cur),
    });
  }

  // --- 剪贴板监听 ---
  const clipboardWatcher = createClipboardWatcher({
    getText: () => ClipboardGetText(),
    onText: (text) => {
      suppressAuto = true;
      input = text;
      setTimeout(() => {
        suppressAuto = false;
        ctrl.translate();
      }, 50);
    },
    isBusy: () => isProcessing,
  });

  $: if (clipboardWatch !== undefined) {
    updateClipboardWatcher(clipboardWatch);
  }

  function updateClipboardWatcher(on: boolean): void {
    if (on) {
      clipboardWatcher.start();
    } else {
      clipboardWatcher.stop();
    }
  }

  function toggleClipboardWatch(): void {
    updateAndSaveConfig("clipboardWatch", !clipboardWatch).catch(() =>
      ctrl.showError("切换剪贴板监听失败")
    );
  }

  // --- 历史记录持久化（防抖写盘）---
  function scheduleHistorySave(): void {
    if (history === undefined) return;
    if (historySaveTimer) clearTimeout(historySaveTimer);
    historySaveTimer = setTimeout(() => {
      SaveHistory(history).catch((e) => {
        console.error("保存历史失败:", e);
        ctrl.showError("保存历史失败");
      });
    }, 500);
  }

  $: if (history !== undefined) {
    scheduleHistorySave();
  }

  // --- 结果再翻译 ---
  function retranslateOutput(): void {
    if (!output || isProcessing) return;
    const origSource = source;
    const origTarget = target;
    let newSource = langs[origTarget] ? origTarget : "auto";
    let newTarget = origSource === "auto" ? (ts.lastDetectedLang || "zh") : origSource;
    if (newTarget === newSource) {
      newTarget = newSource === "zh" ? "en" : "zh";
    }
    ctrl.state.update((s) => ({ ...s, lastDetectedLang: "" }));
    suppressAuto = true;
    input = output;
    ctrl.state.update((s) => ({ ...s, output: "" }));
    source = newSource;
    target = newTarget;
    setTimeout(() => {
      suppressAuto = false;
      ctrl.translate();
    }, 50);
  }

  // --- 对照引擎切换 ---
  function toggleCompareEngine(eng: string): void {
    let next = Array.isArray(compareEngines) ? [...compareEngines] : [];
    if (next.includes(eng)) {
      if (next.length <= 1) return;
      next = next.filter((e) => e !== eng);
    } else {
      next.push(eng);
    }
    updateAndSaveConfig("compareEngines", next);
    ctrl.state.update((s) => ({ ...s, compareOutputs: {} }));
  }

  // --- 复制 ---
  function handleCopy(): void {
    if (!output) return;
    navigator.clipboard.writeText(output);
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }

  function handleCopyEngine(engine: string): void {
    const textToCopy = compareOutputs?.[engine]?.text;
    if (!textToCopy) return;
    navigator.clipboard.writeText(textToCopy);
    copiedEngines[engine] = true;
    setTimeout(() => {
      copiedEngines[engine] = false;
      copiedEngines = { ...copiedEngines };
    }, 2000);
  }

  // --- 历史记录操作 ---
  function clearHistory(): void {
    history = [];
    SaveHistory([]).catch((e) => console.error("清空历史失败:", e));
  }

  function handleHistorySelect(event: CustomEvent<HistoryEntry>): void {
    const item = event.detail;
    suppressAuto = true;
    input = item.input;
    ctrl.state.update((s) => ({ ...s, output: item.output }));
    if (item.source) source = item.source;
    if (item.target) target = item.target;
    showHistory = false;
    setTimeout(() => (suppressAuto = false), 50);
  }

  // --- 快捷键 ---
  const handleGlobalKeydown = createShortcutHandler({
    onTranslate: () => ctrl.translate(),
    onFocusInput: () => inputEl?.focus(),
    onClearInput: () => (input = ""),
    onSwapLangs: () => ([source, target] = [target, source]),
    onToggleHistory: () => (showHistory = !showHistory),
    onToggleTheme: () => updateAndSaveConfig("isDark", !$configStore.isDark),
    onClosePanel: () => {
      if (showHistory) showHistory = false;
    },
  });

  // --- 自动翻译防抖 ---
  $: if (autoTranslate && !suppressAuto) {
    ctrl.handleAutoTranslate(input);
  }

  // --- 生命周期 ---
  onMount(async () => {
    await initConfig();
    try {
      const saved = await LoadHistory();
      if (Array.isArray(saved)) history = saved;
    } catch (e) {
      console.error("加载历史失败:", e);
    }
  });

  onDestroy(() => {
    ctrl.destroy();
    clipboardWatcher.stop();
    speaker.stop();
    if (historySaveTimer) clearTimeout(historySaveTimer);
  });

  // --- 配置面板关闭后刷新 ---
  $: if (!showConfig) {
    refreshConfig();
  }

  async function refreshConfig(): Promise<void> {
    try {
      const cfg = await GetConfig();
      if (cfg) {
        configStore.set(cfg);
      }
    } catch (e) {
      console.error("读取配置失败:", e);
    }
  }
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="app-shell" class:light-mode={!isDark}>
  <ErrorToast
    {errorToast}
    on:retry={() => ctrl.retry()}
    on:settings={() => {
      ctrl.dismissError();
      showConfig = true;
    }}
    on:dismiss={() => ctrl.dismissError()}
  />

  <Sidebar
    {sidebarCollapsed}
    {activeEngine}
    {isDark}
    on:toggleSidebar={() => updateAndSaveConfig("sidebarCollapsed", !sidebarCollapsed)}
    on:selectEngine={(e) => updateAndSaveConfig("defaultEngine", e.detail)}
    on:toggleTheme={() => updateAndSaveConfig("isDark", !$configStore.isDark)}
    on:openSettings={() => (showConfig = true)}
    on:openHistory={() => (showHistory = true)}
  />

  <main class="main-content">
    {#if apiKeyMissing}
      <div class="api-key-banner" transition:fade={{ duration: 200 }}>
        <AlertCircle size={14} />
        <span>
          当前引擎（{activeEngine === "aliyun" ? "阿里云" : "混元"}）未配置凭据，
        </span>
        <button class="banner-link" on:click={() => (showConfig = true)}>前往设置</button>
      </div>
    {/if}

    <LanguageBar
      bind:source
      bind:target
      bind:autoTranslate
      {autoDetectLang}
      {clipboardWatch}
      {compareMode}
      {status}
      on:toggleClipboard={toggleClipboardWatch}
    on:toggleCompare={() => updateAndSaveConfig("compareMode", !compareMode)}
      on:translate={() => ctrl.translate()}
    />

    <div class="editor-container">
      <section class="editor-pane source">
        <textarea
          bind:this={inputEl}
          bind:value={input}
          placeholder="在此输入要翻译的文本..."
          spellcheck="false"
        ></textarea>
        <div class="pane-footer">
          <span class="char-count">{input.length} 字符</span>
          {#if input}
            <button
              class="clear-btn"
              on:click={() => handleSpeak(input, source === "auto" ? "en" : source)}
              title="朗读"
            >
              {#if speakingText === input}
                <Square size={12} fill="currentColor" />
              {:else}
                <Volume2 size={12} />
              {/if}
            </button>
            <button class="clear-btn" on:click={() => (input = "")}>
              <X size={12} /> 清空
            </button>
          {/if}
        </div>
      </section>

      <section class="editor-pane result" class:compare-mode={compareMode}>
        {#if compareMode}
          <ComparePanel
            {compareEngines}
            {compareOutputs}
            {compareLoadingEngines}
            {copiedEngines}
            {speakingText}
            {target}
            on:toggleEngine={(e) => toggleCompareEngine(e.detail)}
            on:copy={(e) => handleCopyEngine(e.detail)}
            on:speak={(e) => handleSpeak(e.detail.text, e.detail.lang)}
          />
        {:else}
          {#if isProcessing && !output}
            <div class="skeleton-block">
              <div class="skeleton-line"></div>
              <div class="skeleton-line short"></div>
              <div class="skeleton-line"></div>
            </div>
          {:else}
            <textarea
              readonly
              value={output}
              placeholder="翻译结果..."
              spellcheck="false"
            ></textarea>
          {/if}
          <div class="pane-footer">
            {#if output}
              <button class="action-btn" on:click={() => handleSpeak(output, target)} title="朗读">
                {#if speakingText === output}
                  <Square size={14} fill="currentColor" />
                {:else}
                  <Volume2 size={14} />
                {/if}
                朗读
              </button>
              <button class="action-btn" on:click={retranslateOutput} title="将翻译结果作为新输入，交换语言后重新翻译">
                <CornerDownLeft size={14} />
                再翻译
              </button>
              <button class="action-btn copy" on:click={handleCopy} class:success={copied}>
                {#if copied}<Check size={14} />{:else}<Copy size={14} />{/if}
                {copied ? "已复制" : "复制"}
              </button>
            {/if}
          </div>
        {/if}
      </section>
    </div>

    <StatusBar {status} />
  </main>

  <History
    bind:show={showHistory}
    {history}
    on:select={handleHistorySelect}
    on:close={() => (showHistory = false)}
    on:clear={clearHistory}
  />

  <Config bind:show={showConfig} {isDark} />
</div>

<style>
  .app-shell {
    display: flex;
    height: 100vh;
    background: var(--bg-base);
    color: var(--text-main);
    font-family: var(--font-sans);
    font-size: var(--fs-base);
    overflow: hidden;
  }

  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: var(--bg-surface);
    position: relative;
    z-index: 1;
  }

  .api-key-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: rgba(251, 191, 36, 0.12);
    border-bottom: 1px solid rgba(251, 191, 36, 0.3);
    color: var(--warning);
    font-size: 12px;
    font-weight: 500;
  }
  .light-mode .api-key-banner {
    background: rgba(251, 191, 36, 0.1);
  }
  .banner-link {
    background: transparent;
    border: none;
    color: var(--warning);
    text-decoration: underline;
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    padding: 0;
  }
  .banner-link:hover {
    color: #d97706;
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
    padding: var(--sp-6);
    position: relative;
    transition: background var(--t-slow) var(--ease-standard);
    min-width: 0;
  }
  .editor-pane.source {
    border-right: 1px solid var(--border);
    flex: 0.9;
  }
  .editor-pane.result {
    background: var(--bg-base);
    padding: var(--sp-3) var(--sp-4);
    flex: 1.1;
  }

  textarea {
    flex: 1;
    background: transparent;
    border: none;
    resize: none;
    outline: none;
    font-size: var(--fs-xl);
    line-height: var(--lh-relaxed);
    color: var(--text-main);
    padding: 0;
    font-family: inherit;
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
    gap: var(--sp-2);
    margin-top: var(--sp-2);
  }
  .editor-pane.result.compare-mode .pane-footer {
    height: 0;
    margin-top: 0;
    overflow: hidden;
  }
  .editor-pane.result.compare-mode {
    padding-bottom: var(--sp-3);
  }

  .char-count {
    font-size: var(--fs-sm);
    color: var(--text-sec);
    margin-right: auto;
    font-variant-numeric: tabular-nums;
  }

  .clear-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    font-size: var(--fs-sm);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    padding: var(--sp-1);
    border-radius: var(--radius-xs);
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard);
  }
  .clear-btn:hover {
    color: var(--text-main);
    background: var(--bg-hover);
  }

  .action-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: var(--sp-1) var(--sp-3);
    border-radius: var(--radius-sm);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    transition: all var(--t-base) var(--ease-standard);
  }
  .action-btn:hover {
    border-color: var(--text-sec);
    background: var(--bg-hover);
  }
  .action-btn:active {
    transform: scale(0.96);
  }
  .action-btn.success {
    border-color: var(--success);
    color: var(--success);
    background: var(--success-soft);
  }

  .skeleton-block {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 4px 0;
  }
  .skeleton-line {
    height: 16px;
    border-radius: 6px;
    background: linear-gradient(
      90deg,
      var(--bg-hover) 25%,
      var(--border) 50%,
      var(--bg-hover) 75%
    );
    background-size: 200% 100%;
    animation: shimmer 1.4s ease-in-out infinite;
  }
  .skeleton-line.short {
    width: 60%;
  }
  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }

  @media (max-width: 720px) {
    .editor-pane {
      padding: var(--sp-3);
    }
    .editor-pane.result {
      padding: var(--sp-2) var(--sp-3);
    }
    textarea {
      font-size: var(--fs-lg);
    }
  }
</style>
