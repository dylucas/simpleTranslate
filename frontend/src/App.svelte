<script lang="ts">
  import { LoadHistory, SaveHistory, GetConfig } from "../wailsjs/go/main/App";
  import {
    AlertCircle,
    Volume2,
    Square,
    X,
    CornerDownLeft,
    Copy,
    Check,
    TextCursorInput,
    Languages as LanguagesIcon,
  } from "lucide-svelte";
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
  async function handleCopy(): Promise<void> {
    if (!output) return;
    try {
      await navigator.clipboard.writeText(output);
      copied = true;
      setTimeout(() => (copied = false), 2000);
    } catch {
      ctrl.showError("复制失败，请重试");
    }
  }

  async function handleCopyEngine(engine: string): Promise<void> {
    const textToCopy = compareOutputs?.[engine]?.text;
    if (!textToCopy) return;
    try {
      await navigator.clipboard.writeText(textToCopy);
      copiedEngines[engine] = true;
      setTimeout(() => {
        copiedEngines[engine] = false;
        copiedEngines = { ...copiedEngines };
      }, 2000);
    } catch {
      ctrl.showError("复制失败，请重试");
    }
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
  const dispatchShortcut = createShortcutHandler({
    onTranslate: () => ctrl.translate(),
    onFocusInput: () => inputEl?.focus(),
    onClearInput: () => (input = ""),
    onSwapLangs: () => {
      const resolvedSource = source === "auto" ? ts.lastDetectedLang : source;
      if (resolvedSource) [source, target] = [target, resolvedSource];
    },
    onToggleHistory: () => (showHistory = !showHistory),
    onToggleTheme: () => updateAndSaveConfig("isDark", !$configStore.isDark),
    onClosePanel: () => {
      if (showConfig) showConfig = false;
      else if (showHistory) showHistory = false;
    },
  });

  function handleGlobalKeydown(event: KeyboardEvent): void {
    if ((showConfig || showHistory) && event.key !== "Escape") return;
    dispatchShortcut(event);
  }

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
      detectedSource={ts.lastDetectedLang}
      {clipboardWatch}
      {compareMode}
      {status}
      {isProcessing}
      canTranslate={!!input.trim()}
      on:toggleClipboard={toggleClipboardWatch}
      on:toggleCompare={() => updateAndSaveConfig("compareMode", !compareMode)}
      on:translate={() => ctrl.translate()}
    />

    <div class="editor-container">
      <section class="editor-pane source" aria-labelledby="source-pane-title">
        <div class="pane-heading">
          <div class="pane-title-wrap">
            <span class="pane-icon"><TextCursorInput size={15} /></span>
            <div>
              <span class="pane-eyebrow">输入</span>
              <h2 id="source-pane-title">原文</h2>
            </div>
          </div>
          <span class="char-count">{input.length} 字符</span>
        </div>
        <textarea
          bind:this={inputEl}
          bind:value={input}
          placeholder="在此输入要翻译的文本..."
          aria-label="原文"
          spellcheck="false"
        ></textarea>
        <div class="pane-footer source-footer">
          <span class="input-hint">{autoTranslate ? "停顿后自动翻译" : "按快捷键或点击翻译"}</span>
          {#if input}
            <button
              class="clear-btn"
              on:click={() => handleSpeak(input, source === "auto" ? "en" : source)}
              title={speakingText === input ? "停止朗读" : "朗读原文"}
              aria-label={speakingText === input ? "停止朗读原文" : "朗读原文"}
            >
              {#if speakingText === input}
                <Square size={12} fill="currentColor" />
              {:else}
                <Volume2 size={12} />
              {/if}
            </button>
            <button class="clear-btn" on:click={() => (input = "")} aria-label="清空原文">
              <X size={12} /> 清空
            </button>
          {/if}
        </div>
      </section>

      <section
        class="editor-pane result"
        class:compare-mode={compareMode}
        aria-labelledby="result-pane-title"
        aria-busy={isProcessing}
      >
        <div class="pane-heading result-heading">
          <div class="pane-title-wrap">
            <span class="pane-icon result-icon"><LanguagesIcon size={15} /></span>
            <div>
              <span class="pane-eyebrow">输出 · {compareMode ? "多引擎" : activeEngine === "aliyun" ? "阿里云" : "混元"}</span>
              <h2 id="result-pane-title">{compareMode ? "对照译文" : "译文"}</h2>
            </div>
          </div>
          {#if isProcessing && output}
            <span class="update-status"><span></span>正在更新</span>
          {/if}
        </div>
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
          {:else if output}
            <textarea
              readonly
              value={output}
              aria-label="译文"
              aria-live="polite"
              spellcheck="false"
            ></textarea>
          {:else}
            <div class="result-empty">
              <div class="empty-symbol"><LanguagesIcon size={22} strokeWidth={1.7} /></div>
              <p>译文会显示在这里</p>
              <span>{autoTranslate ? "输入内容后将自动开始翻译" : "输入内容并点击翻译按钮"}</span>
            </div>
          {/if}
          <div class="pane-footer">
            {#if output}
              <button
                class="action-btn"
                on:click={() => handleSpeak(output, target)}
                title={speakingText === output ? "停止朗读" : "朗读译文"}
              >
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
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    background: var(--bg-workspace);
    position: relative;
    z-index: var(--z-base);
    isolation: isolate;
  }

  .api-key-banner {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: var(--sp-2);
    padding: var(--sp-2) var(--sp-4);
    background: var(--warning-soft);
    border-bottom: 1px solid var(--warning-border);
    color: var(--warning);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    z-index: var(--z-banner);
  }
  .banner-link {
    background: transparent;
    border: none;
    color: var(--warning);
    cursor: pointer;
    font-size: var(--fs-sm);
    font-weight: var(--fw-semibold);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-full);
    transition: background var(--t-base) var(--ease-standard),
      color var(--t-base) var(--ease-standard);
  }
  .banner-link:hover {
    background: var(--warning-soft);
    color: var(--text-main);
  }

  .editor-container {
    flex: 1;
    min-height: 0;
    display: flex;
    gap: var(--sp-3);
    padding: var(--sp-4);
    overflow: hidden;
  }

  .editor-pane {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: var(--sp-5);
    position: relative;
    overflow: hidden;
    background: var(--bg-panel);
    border: 1px solid var(--border);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-card);
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
    transition: border-color var(--t-base) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard),
      background var(--t-slow) var(--ease-standard);
  }
  .editor-pane:focus-within {
    border-color: var(--primary);
    box-shadow: var(--shadow-card), 0 0 0 3px var(--primary-soft);
  }
  .editor-pane.source {
    flex: 0.96;
  }
  .editor-pane.result {
    background: var(--bg-panel-muted);
    flex: 1.04;
  }

  .pane-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-3);
    min-height: var(--sp-10);
    margin-bottom: var(--sp-4);
  }
  .pane-title-wrap {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    min-width: 0;
  }
  .pane-icon {
    width: var(--sp-8);
    height: var(--sp-8);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    color: var(--primary);
    background: var(--primary-soft);
    border-radius: var(--radius-md);
  }
  .pane-icon.result-icon {
    color: var(--info);
    background: var(--info-soft);
  }
  .pane-eyebrow {
    display: block;
    color: var(--text-muted);
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    line-height: var(--lh-tight);
    letter-spacing: var(--tracking-wider);
    text-transform: uppercase;
  }
  .pane-heading h2 {
    margin: var(--sp-1) 0 0;
    color: var(--text-main);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    line-height: var(--lh-tight);
  }

  .char-count {
    flex-shrink: 0;
    padding: var(--sp-1) var(--sp-2);
    color: var(--text-muted);
    background: var(--bg-hover);
    border-radius: var(--radius-full);
    font-size: var(--fs-xs);
    font-variant-numeric: tabular-nums;
  }

  textarea {
    flex: 1;
    min-height: 0;
    background: transparent;
    border: none;
    resize: none;
    font-size: var(--fs-xl);
    line-height: var(--lh-relaxed);
    color: var(--text-main);
    padding: var(--sp-1);
    font-family: inherit;
    letter-spacing: var(--tracking-normal);
  }
  textarea:focus-visible {
    outline: none;
  }
  textarea::placeholder {
    color: var(--text-muted);
    opacity: 1;
  }

  .pane-footer {
    min-height: var(--sp-8);
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--sp-2);
    margin-top: var(--sp-3);
  }
  .source-footer {
    justify-content: space-between;
  }
  .input-hint {
    margin-right: auto;
    color: var(--text-muted);
    font-size: var(--fs-xs);
  }
  .editor-pane.result.compare-mode .pane-footer {
    height: 0;
    margin-top: 0;
    overflow: hidden;
  }
  .editor-pane.result.compare-mode {
    padding-bottom: var(--sp-4);
  }

  .clear-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-sec);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    min-height: var(--sp-8);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-md);
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard),
      border-color var(--t-base) var(--ease-standard);
  }
  .clear-btn:hover {
    color: var(--text-main);
    background: var(--bg-hover);
    border-color: var(--border);
  }

  .action-btn {
    min-height: var(--sp-8);
    background: var(--bg-surface);
    border: 1px solid var(--border);
    color: var(--text-sec);
    padding: var(--sp-1) var(--sp-3);
    border-radius: var(--radius-md);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    transition: all var(--t-base) var(--ease-standard);
  }
  .action-btn:hover {
    border-color: var(--border-strong);
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .action-btn:active {
    transform: scale(0.96);
  }
  .action-btn.success {
    border-color: var(--success);
    color: var(--success);
    background: var(--success-soft);
  }

  .result-empty {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: var(--text-muted);
    text-align: center;
    padding: var(--sp-8);
  }
  .empty-symbol {
    width: var(--sp-12);
    height: var(--sp-12);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    margin-bottom: var(--sp-4);
    color: var(--primary);
    background: var(--primary-soft);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
  }
  .result-empty p {
    margin: 0;
    color: var(--text-sec);
    font-size: var(--fs-md);
    font-weight: var(--fw-medium);
  }
  .result-empty span {
    margin-top: var(--sp-1);
    font-size: var(--fs-sm);
  }

  .update-status {
    display: inline-flex;
    align-items: center;
    gap: var(--sp-2);
    color: var(--primary);
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
  }
  .update-status > span {
    width: var(--sp-2);
    height: var(--sp-2);
    border-radius: var(--radius-full);
    background: var(--primary);
    animation: statusPulse 1.2s var(--ease-standard) infinite;
  }

  .skeleton-block {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    padding: var(--sp-2) var(--sp-1);
  }
  .skeleton-line {
    height: var(--sp-4);
    border-radius: var(--radius-sm);
    background: linear-gradient(
      90deg,
      var(--bg-hover) 25%,
      var(--border) 50%,
      var(--bg-hover) 75%
    );
    background-size: 200% 100%;
    animation: shimmer 1.4s var(--ease-standard) infinite;
  }
  .skeleton-line.short {
    width: 60%;
  }
  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }
  @keyframes statusPulse {
    0%, 100% { opacity: 0.4; transform: scale(0.84); }
    50% { opacity: 1; transform: scale(1); }
  }

  @media (max-width: 900px) {
    .editor-container {
      gap: var(--sp-2);
      padding: var(--sp-3);
    }
    .editor-pane {
      padding: var(--sp-4);
    }
  }

  @media (max-width: 720px) {
    .editor-container {
      flex-direction: column;
      overflow-y: auto;
    }
    .editor-pane,
    .editor-pane.source,
    .editor-pane.result {
      flex: 1 0 240px;
    }
    .pane-heading {
      margin-bottom: var(--sp-3);
    }
    textarea {
      font-size: var(--fs-lg);
    }
  }

  @media (max-width: 520px) {
    .editor-container {
      padding: var(--sp-2);
    }
    .editor-pane {
      padding: var(--sp-3);
      border-radius: var(--radius-lg);
    }
    .input-hint {
      display: none;
    }
  }
</style>
