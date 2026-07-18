<script lang="ts">
  import { AlertCircle, Check, Copy, CornerDownLeft, Languages, Square, TextCursorInput, Volume2, X } from "@lucide/svelte";
  import { onDestroy, onMount, tick } from "svelte";
  import { desktopBridge } from "./lib/bridge";
  import { createClipboardWatcher } from "./lib/clipboard";
  import { createConfigController } from "./lib/configController";
  import { langs, getSpeechLang } from "./lib/languages";
  import { ARIA_SHORTCUTS, createShortcutHandler } from "./lib/shortcuts";
  import { createSpeaker } from "./lib/speech";
  import { createTranslateController, type TranslateController } from "./lib/translateController";
  import type { EngineId, HistoryEntry } from "./lib/types";
  import ComparePanel from "./lib/ComparePanel.svelte";
  import Config from "./lib/Config.svelte";
  import ErrorToast from "./lib/ErrorToast.svelte";
  import History from "./lib/History.svelte";
  import CommandBar from "./lib/CommandBar.svelte";
  import StatusBar from "./lib/StatusBar.svelte";
  import UtilityRail from "./lib/UtilityRail.svelte";

  let input = $state("");
  let source = $state("auto");
  let target = $state("zh");
  let showConfig = $state(false);
  let showHistory = $state(false);
  let history = $state<HistoryEntry[]>([]);
  let suppressAuto = $state(false);
  let copied = $state(false);
  let copiedEngines = $state<Partial<Record<EngineId, boolean>>>({});
  let speakingText = $state<string | null>(null);
  let inputElement: HTMLTextAreaElement;
  let historySaveTimer: ReturnType<typeof setTimeout> | null = null;
  let lastPanelTrigger: HTMLElement | null = null;
  let queuedErrors: string[] = [];
  let controller: TranslateController;

  const configController = createConfigController(desktopBridge, (message) => {
    if (controller) controller.showError(message);
    else queuedErrors.push(message);
  });
  let config = $derived($configController.value);

  function scheduleHistorySave(): void {
    if (historySaveTimer) clearTimeout(historySaveTimer);
    historySaveTimer = setTimeout(() => {
      void desktopBridge.saveHistory(history).catch(() => controller.showError("历史记录保存失败"));
    }, 400);
  }

  controller = createTranslateController({
    bridge: desktopBridge,
    getInput: () => input,
    getSource: () => source,
    getTarget: () => target,
    getActiveEngine: () => config.defaultEngine,
    getCompareMode: () => config.compareMode,
    getCompareEngines: () => availableCompareEngines,
    getHistory: () => history,
    setHistory: (update) => {
      history = update(history);
      scheduleHistorySave();
    },
    setTarget: (next) => {
      if (target === next) return;
      target = next;
      void configController.patch("targetLanguage", next);
    },
  });
  const translationState = controller.state;
  let translation = $derived($translationState);

  function isConfigured(engine: EngineId): boolean {
    return engine === "tencent"
      ? Boolean(config.tencent.secretKey.trim())
      : Boolean(config.aliyun.secretId.trim() && config.aliyun.secretKey.trim());
  }

  let availableCompareEngines = $derived(config.compareEngines.filter(isConfigured));
  let missingCompareEngines = $derived(config.compareEngines.filter((engine) => !isConfigured(engine)));
  let credentialsReady = $derived(config.compareMode
    ? availableCompareEngines.length > 0
    : isConfigured(config.defaultEngine));
  let canTranslate = $derived(Boolean(input.trim()) && credentialsReady);
  let unavailableReason = $derived(!input.trim()
    ? "请输入文本"
    : !credentialsReady
      ? "请先配置翻译凭据"
      : "");
  let credentialMessage = $derived.by(() => {
    if (!config.compareMode && !isConfigured(config.defaultEngine)) {
      return `${config.defaultEngine === "tencent" ? "腾讯混元" : "阿里云"}尚未配置凭据`;
    }
    if (config.compareMode && missingCompareEngines.length) {
      const names = missingCompareEngines.map((engine) => engine === "tencent" ? "混元" : "阿里云").join("、");
      return availableCompareEngines.length ? `${names}未配置，将仅使用可用引擎` : "所选对照引擎均未配置凭据";
    }
    return "";
  });

  const speaker = createSpeaker(getSpeechLang);
  const clipboardWatcher = createClipboardWatcher({
    getText: () => desktopBridge.getClipboardText(),
    isBusy: () => translation.isProcessing,
    onText: (text) => {
      suppressAuto = true;
      input = text;
      setTimeout(() => {
        suppressAuto = false;
        requestTranslation();
      }, 0);
    },
  });

  $effect(() => {
    if (config.clipboardWatch) clipboardWatcher.start();
    else clipboardWatcher.stop();
  });

  $effect(() => {
    const value = input;
    if (config.autoTranslate && !suppressAuto && credentialsReady) {
      controller.handleAutoTranslate(value);
    }
  });

  function openPanel(panel: "config" | "history"): void {
    lastPanelTrigger = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    showConfig = panel === "config";
    showHistory = panel === "history";
  }

  function closePanels(): void {
    showConfig = false;
    showHistory = false;
    void tick().then(() => lastPanelTrigger?.focus());
  }

  function requestTranslation(): void {
    if (!credentialsReady) {
      controller.showError({ errorCode: "credentials", error: "请先配置可用的翻译凭据" });
      return;
    }
    void controller.translate();
  }

  function setLanguage(kind: "source" | "target", value: string): void {
    if (kind === "source") source = value;
    else target = value;
    void configController.patch(kind === "source" ? "sourceLanguage" : "targetLanguage", value);
  }

  function swapLanguages(): void {
    const resolved = source === "auto" ? translation.lastDetectedLang : source;
    if (!resolved) return;
    const nextSource = target;
    const nextTarget = resolved;
    source = nextSource;
    target = nextTarget;
    void configController.save({ ...configController.snapshot(), sourceLanguage: nextSource, targetLanguage: nextTarget });
  }

  function toggleCompareEngine(engine: EngineId): void {
    const selected = [...config.compareEngines];
    const next = selected.includes(engine)
      ? selected.length === 1 ? selected : selected.filter((item) => item !== engine)
      : [...selected, engine];
    void configController.patch("compareEngines", next);
  }

  function speak(text: string, language: string): void {
    speaker.speak(text, language, { onChange: (value) => (speakingText = value) });
  }

  async function copyText(text: string, engine?: EngineId): Promise<void> {
    if (!text) return;
    try {
      await navigator.clipboard.writeText(text);
      if (engine) copiedEngines[engine] = true;
      else copied = true;
      setTimeout(() => {
        if (engine) copiedEngines[engine] = false;
        else copied = false;
      }, 1600);
    } catch {
      controller.showError("复制失败，请重试");
    }
  }

  function retranslate(): void {
    if (!translation.output || translation.isProcessing) return;
    const nextSource = target;
    let nextTarget = source === "auto" ? (translation.lastDetectedLang || "zh") : source;
    if (nextSource === nextTarget) nextTarget = nextSource === "zh" ? "en" : "zh";
    suppressAuto = true;
    input = translation.output;
    controller.setOutput("");
    source = nextSource;
    target = nextTarget;
    void configController.save({ ...configController.snapshot(), sourceLanguage: nextSource, targetLanguage: nextTarget });
    setTimeout(() => {
      suppressAuto = false;
      requestTranslation();
    }, 0);
  }

  function focusInputFromShortcut(): void {
    showConfig = false;
    showHistory = false;
    void tick().then(() => inputElement?.focus());
  }

  const shortcuts = createShortcutHandler({
    onTranslate: requestTranslation,
    onFocusInput: focusInputFromShortcut,
    onClearInput: () => (input = ""),
    onSwapLangs: swapLanguages,
    onToggleHistory: () => showHistory ? closePanels() : openPanel("history"),
    onToggleSettings: () => showConfig ? closePanels() : openPanel("config"),
    onToggleTheme: () => void configController.patch("isDark", !config.isDark),
    onClosePanel: closePanels,
  }, {
    isPanelOpen: () => showConfig || showHistory,
  });

  function handleGlobalKeydown(event: KeyboardEvent): void {
    shortcuts(event);
  }

  onMount(async () => {
    await configController.load();
    const loaded = configController.snapshot();
    source = loaded.sourceLanguage;
    target = loaded.targetLanguage;
    try {
      history = await desktopBridge.loadHistory();
    } catch {
      controller.showError("历史记录加载失败");
    }
    for (const message of queuedErrors) controller.showError(message);
    queuedErrors = [];
  });

  onDestroy(() => {
    controller.destroy();
    clipboardWatcher.stop();
    speaker.stop();
    if (historySaveTimer) clearTimeout(historySaveTimer);
  });
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<div class="app-shell" class:light-mode={!config.isDark} class:history-open={showHistory}>
  <ErrorToast errorToast={translation.errorToast} onRetry={() => controller.retry()} onSettings={() => { controller.dismissError(); openPanel("config"); }} onDismiss={() => controller.dismissError()} />
  <UtilityRail
    activePanel={showConfig ? "config" : showHistory ? "history" : null}
    isDark={config.isDark}
    onTheme={() => void configController.patch("isDark", !config.isDark)}
    onSettings={() => openPanel("config")}
    onHistory={() => openPanel("history")}
  />

  <main class="main-content">
    <CommandBar
      {source} {target} activeEngine={config.defaultEngine} autoTranslate={config.autoTranslate} autoDetectLang={translation.autoDetectLang}
      detectedSource={translation.lastDetectedLang} clipboardWatch={config.clipboardWatch}
      compareMode={config.compareMode} isProcessing={translation.isProcessing} {canTranslate} {unavailableReason}
      onSource={(value) => setLanguage("source", value)} onTarget={(value) => setLanguage("target", value)}
      onEngine={(engine) => void configController.patch("defaultEngine", engine)}
      onSwap={swapLanguages} onAuto={(value) => void configController.patch("autoTranslate", value)}
      onClipboard={() => void configController.patch("clipboardWatch", !config.clipboardWatch)}
      onCompare={() => void configController.patch("compareMode", !config.compareMode)} onTranslate={requestTranslation}
    />

    {#if credentialMessage}
      <div class="credential-banner" role="status">
        <AlertCircle size={14} />
        <span>{credentialMessage}</span>
        <button onclick={() => openPanel("config")} aria-label="配置翻译服务">前往设置</button>
      </div>
    {/if}

    <div class="workspace">
      <section class="editor-pane" aria-labelledby="source-title">
        <header class="pane-header">
          <div class="pane-title"><span class="pane-icon"><TextCursorInput size={14} /></span><span class="pane-label">输入</span><h2 id="source-title">原文</h2></div>
          <span class="char-count">{input.length} 字符</span>
        </header>
        <textarea class="editor" bind:this={inputElement} bind:value={input} placeholder="输入要翻译的文本" aria-label="原文" aria-keyshortcuts={ARIA_SHORTCUTS.focusInput} spellcheck="false"></textarea>
        <footer class="pane-footer">
          <span class="pane-note">{config.autoTranslate ? "自动翻译" : "手动翻译"}</span>
          <div class="pane-actions">
            {#if input}
              <button class="icon-action" onclick={() => speak(input, source === "auto" ? (translation.lastDetectedLang || "en") : source)} aria-label="朗读原文" title="朗读原文">{#if speakingText === input}<Square size={13} />{:else}<Volume2 size={14} />{/if}</button>
              <button class="icon-action" onclick={() => (input = "")} aria-label="清空原文" aria-keyshortcuts={ARIA_SHORTCUTS.clearInput} title="清空原文"><X size={14} /></button>
            {/if}
          </div>
        </footer>
      </section>

      <section class="editor-pane result-pane" aria-labelledby="result-title" aria-busy={translation.isProcessing}>
        <header class="pane-header">
          <div class="pane-title"><span class="pane-icon result"><Languages size={14} /></span><span class="pane-label">输出</span><h2 id="result-title">{config.compareMode ? "对照译文" : "译文"}</h2><span class="pane-context">{config.compareMode ? "多引擎" : config.defaultEngine === "tencent" ? "混元" : "阿里云"}</span></div>
          {#if translation.isProcessing && translation.output}<span class="updating"><i></i>正在更新</span>{/if}
        </header>
        {#if config.compareMode}
          <div class="compare-wrap"><ComparePanel engines={config.compareEngines} outputs={translation.compareOutputs} loading={translation.compareLoadingEngines} copied={copiedEngines} {speakingText} {target} {isConfigured} onToggle={toggleCompareEngine} onCopy={(engine) => void copyText(translation.compareOutputs[engine]?.text ?? "", engine)} onSpeak={speak} onSettings={() => openPanel("config")} /></div>
        {:else if translation.output}
          <textarea class="editor" readonly value={translation.output} aria-label="译文"></textarea>
          <footer class="pane-footer">
            <span class="pane-note">{langs[target] ?? target}</span>
            <div class="pane-actions">
              <button class="icon-action" onclick={() => speak(translation.output, target)} aria-label="朗读译文" title="朗读译文">{#if speakingText === translation.output}<Square size={13} />{:else}<Volume2 size={14} />{/if}</button>
              <button onclick={retranslate} aria-label="重新翻译译文" title="重新翻译"><CornerDownLeft size={14} /><span class="action-label">再翻译</span></button>
              <button class:success={copied} onclick={() => void copyText(translation.output)} aria-label={copied ? "译文已复制" : "复制译文"} title="复制译文">{#if copied}<Check size={14} /><span class="action-label">已复制</span>{:else}<Copy size={14} /><span class="action-label">复制</span>{/if}</button>
            </div>
          </footer>
        {:else}
          <div class="result-empty"><span class="empty-icon"><Languages size={23} strokeWidth={1.4} /></span><strong>等待翻译</strong><span>译文将在这里显示</span></div>
        {/if}
      </section>
    </div>
    <StatusBar status={translation.status} bridgeKind={desktopBridge.kind} {source} {target} activeEngine={config.defaultEngine} compareMode={config.compareMode} />
  </main>

  <Config open={showConfig} {config} onClose={closePanels} onSave={(next) => configController.save(next)} onTest={(engine, service) => desktopBridge.testConnection(engine, service)} />
  <History open={showHistory} {history} onClose={closePanels} onClear={() => { history = []; scheduleHistorySave(); }} onSelect={(entry) => { suppressAuto = true; input = entry.input; source = entry.source; target = entry.target; controller.restore(entry); closePanels(); setTimeout(() => (suppressAuto = false), 0); }} />
</div>

<style>
  .app-shell { --history-drawer-w: 440px; display: flex; width: 100vw; height: 100vh; overflow: hidden; background: var(--bg-base); color: var(--text-main); }
  .main-content { display: flex; min-width: 0; flex: 1; flex-direction: column; background: var(--bg-workspace); transition: margin-right var(--t-base) var(--ease-standard); }
  .credential-banner { display: flex; min-height: 30px; flex: 0 0 auto; align-items: center; gap: var(--sp-2); border-bottom: 1px solid var(--warning-border); padding: 0 var(--sp-3); background: var(--warning-soft); color: var(--text-sec); font-size: var(--fs-xs); }
  .credential-banner span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .credential-banner :global(svg) { color: var(--warning); }
  .credential-banner button { flex: 0 0 auto; border: 0; padding: 0; background: transparent; color: var(--warning); cursor: pointer; font: inherit; font-weight: var(--fw-semibold); }
  .credential-banner button:hover { color: var(--text-main); }
  .workspace { display: grid; min-height: 0; flex: 1; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 1px; background: var(--border-soft); }
  .editor-pane { position: relative; display: flex; min-width: 0; min-height: 0; flex-direction: column; background: var(--bg-panel); transition: box-shadow var(--t-base) var(--ease-standard); }
  .editor-pane:focus-within { z-index: 1; box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--primary) 45%, transparent); }
  .result-pane { background: var(--bg-panel-muted); }
  .pane-header { display: flex; min-height: 42px; flex: 0 0 auto; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border-soft); padding: 0 var(--sp-3); }
  .pane-title { display: flex; min-width: 0; align-items: center; gap: var(--sp-2); }
  .pane-icon { display: grid; width: 26px; height: 26px; flex: 0 0 auto; place-items: center; border-radius: var(--radius-sm); background: var(--bg-hover); color: var(--text-sec); }
  .pane-icon.result { background: var(--primary-soft); color: var(--primary); }
  .pane-label { color: var(--text-muted); font-size: var(--fs-xs); }
  h2 { margin: 0; color: var(--text-main); font-size: var(--fs-base); font-weight: var(--fw-semibold); }
  .pane-context { overflow: hidden; padding-left: var(--sp-2); border-left: 1px solid var(--border); color: var(--text-muted); font-size: var(--fs-xs); text-overflow: ellipsis; white-space: nowrap; }
  .char-count { color: var(--text-muted); font-size: var(--fs-xs); font-variant-numeric: tabular-nums; }
  .editor { width: 100%; min-height: 0; flex: 1; resize: none; border: 0; outline: 0; padding: 18px; background: transparent; color: var(--text-main); font: var(--fw-regular) var(--fs-lg)/var(--lh-relaxed) var(--font-sans); }
  .editor::placeholder { color: color-mix(in srgb, var(--text-muted) 84%, transparent); }
  .pane-footer { display: flex; min-height: 38px; flex: 0 0 auto; align-items: center; justify-content: space-between; border-top: 1px solid var(--border-soft); padding: 0 var(--sp-3); color: var(--text-muted); font-size: var(--fs-xs); }
  .pane-note { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .pane-actions { display: flex; min-width: 0; align-items: center; gap: 2px; }
  .pane-footer button { display: inline-flex; height: 28px; align-items: center; justify-content: center; gap: var(--sp-1); border: 1px solid transparent; border-radius: var(--radius-sm); padding: 0 var(--sp-2); background: transparent; color: var(--text-sec); cursor: pointer; font: inherit; }
  .pane-footer button:hover { border-color: var(--border); background: var(--bg-hover); color: var(--text-main); }
  .pane-footer button.icon-action { width: 28px; padding: 0; }
  .pane-footer button.success { color: var(--success); }
  .result-empty { display: grid; flex: 1; place-content: center; place-items: center; gap: var(--sp-2); color: var(--text-muted); font-size: var(--fs-xs); }
  .result-empty strong { color: var(--text-sec); font-size: var(--fs-sm); font-weight: var(--fw-medium); }
  .empty-icon { display: grid; width: 44px; height: 44px; margin-bottom: var(--sp-1); place-items: center; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--bg-surface); color: var(--text-sec); }
  .updating { display: flex; align-items: center; gap: var(--sp-2); color: var(--primary); font-size: var(--fs-xs); }
  .updating i { width: 7px; height: 7px; border-radius: 50%; background: var(--primary); animation: pulse 1s infinite; }
  .compare-wrap { display: flex; min-height: 0; flex: 1; flex-direction: column; }
  @keyframes pulse { 50% { opacity: .3; } }
  @media (min-width: 1180px) {
    .app-shell.history-open .main-content { margin-right: var(--history-drawer-w); }
    .app-shell.history-open :global(.mode-toggles button) { width: 30px; padding-inline: 0; }
    .app-shell.history-open :global(.mode-toggles span) { display: none; }
    .app-shell.history-open :global(.engine-switch button) { min-width: 48px; }
  }
  @media (max-width: 720px) {
    .workspace { grid-template-columns: 1fr; grid-template-rows: minmax(220px, 1fr) minmax(220px, 1fr); overflow: auto; }
    .credential-banner { min-height: 30px; }
    .editor { padding: var(--sp-4); }
  }
  @media (max-width: 460px) {
    .credential-banner { padding-inline: var(--sp-2); }
    .credential-banner span { white-space: nowrap; }
    .pane-header, .pane-footer { padding-inline: var(--sp-2); }
    .pane-label { display: none; }
    .action-label { display: none; }
    .pane-footer button:not(.icon-action) { width: 28px; padding: 0; }
  }
</style>
