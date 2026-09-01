<script lang="ts">
  import { AlertCircle, Check, Copy, CornerDownLeft, Languages, Square, TextCursorInput, Volume2, X } from "@lucide/svelte";
  import { onDestroy, onMount } from "svelte";
  import { desktopBridge } from "./lib/bridge";
  import { createAppOrchestrator } from "./lib/appOrchestrator.svelte";
  import { engineLabel } from "./lib/engines";
  import { langs } from "./lib/languages";
  import { ARIA_SHORTCUTS } from "./lib/shortcuts";
  import { MAX_INPUT_BYTES } from "./lib/textLimits";
  import ComparePanel from "./lib/ComparePanel.svelte";
  import Config from "./lib/Config.svelte";
  import ErrorToast from "./lib/ErrorToast.svelte";
  import History from "./lib/History.svelte";
  import CommandBar from "./lib/CommandBar.svelte";
  import StatusBar from "./lib/StatusBar.svelte";
  import UtilityRail from "./lib/UtilityRail.svelte";

  let inputElement: HTMLTextAreaElement;
  let lastPanelTrigger: HTMLElement | null = null;

  const app = createAppOrchestrator({
    bridge: desktopBridge,
    getInputElement: () => inputElement,
    getLastPanelTrigger: () => lastPanelTrigger,
    setLastPanelTrigger: (value) => { lastPanelTrigger = value; },
  });

  onMount(() => app.init());
  onDestroy(() => app.destroy());
</script>

<svelte:window onfocus={app.handleWindowFocus} onkeydown={app.handleGlobalKeydown} />

<div class="app-shell" class:light-mode={!app.config.isDark} class:history-open={app.showHistory}>
  <ErrorToast errorToast={app.translation.errorToast} onRetry={() => app.translateController.retry()} onSettings={() => { app.translateController.dismissError(); app.openPanel("config"); }} onDismiss={() => app.translateController.dismissError()} />
  <UtilityRail
    activePanel={app.showConfig ? "config" : app.showHistory ? "history" : null}
    isDark={app.config.isDark}
    onTheme={app.toggleTheme}
    onSettings={() => app.openPanel("config")}
    onHistory={() => app.openPanel("history")}
  />

  <main class="main-content">
    <CommandBar
      source={app.source} target={app.target} activeEngine={app.config.defaultEngine} baiduDomain={app.config.baidu.domain}
      showBaiduDomain={app.config.defaultEngine === "baidu" || (app.config.compareMode && app.config.compareEngines.includes("baidu"))}
      autoTranslate={app.config.autoTranslate} autoDetectLang={app.translation.autoDetectLang}
      detectedSource={app.translation.lastDetectedLang} clipboardWatch={app.config.clipboardWatch}
      compareMode={app.config.compareMode} isProcessing={app.translation.isProcessing} canTranslate={app.canTranslate} unavailableReason={app.unavailableReason}
      onSource={(value) => app.setLanguage("source", value)} onTarget={(value) => app.setLanguage("target", value)}
      onEngine={(engine) => app.patchConfig("defaultEngine", engine)}
      onBaiduDomain={app.setBaiduDomain}
      onSwap={app.swapLanguages} onAuto={(value) => app.patchConfig("autoTranslate", value)}
      onClipboard={() => app.patchConfig("clipboardWatch", !app.config.clipboardWatch)}
      onCompare={() => app.patchConfig("compareMode", !app.config.compareMode)} onTranslate={app.requestTranslation} onCancel={() => app.translateController.cancel()}
    />

    {#if app.credentialMessage}
      <div class="credential-banner" role="status">
        <AlertCircle size={14} />
        <span>{app.credentialMessage}</span>
        <button onclick={() => app.openPanel("config")} aria-label="配置翻译服务">前往设置</button>
      </div>
    {/if}

    <div class="workspace">
      <section class="editor-pane" aria-labelledby="source-title">
        <header class="pane-header">
          <div class="pane-title"><span class="pane-icon"><TextCursorInput size={14} /></span><span class="pane-label">输入</span><h2 id="source-title">原文</h2></div>
          <span class="char-count">{app.inputBytes} / {MAX_INPUT_BYTES} 字节</span>
        </header>
        <textarea class="editor" bind:this={inputElement} value={app.input} oninput={(event) => app.updateInput(event.currentTarget.value)} placeholder="输入要翻译的文本" aria-label="原文" aria-keyshortcuts={ARIA_SHORTCUTS.focusInput} autocomplete="off" spellcheck="false"></textarea>
        <footer class="pane-footer">
          <span class="pane-note">{app.config.autoTranslate ? "自动翻译" : "手动翻译"}</span>
          <div class="pane-actions">
            {#if app.input}
              <button class="icon-action" onclick={() => app.speak(app.input, app.source === "auto" ? (app.translation.lastDetectedLang || "en") : app.source)} aria-label="朗读原文" title="朗读原文">{#if app.speakingText === app.input}<Square size={13} />{:else}<Volume2 size={14} />{/if}</button>
              <button class="icon-action" onclick={app.clearInput} aria-label="清空原文" aria-keyshortcuts={ARIA_SHORTCUTS.clearInput} title="清空原文"><X size={14} /></button>
            {/if}
          </div>
        </footer>
      </section>

      <section class="editor-pane result-pane" aria-labelledby="result-title" aria-busy={app.translation.isProcessing}>
        <header class="pane-header">
          <div class="pane-title"><span class="pane-icon result"><Languages size={14} /></span><span class="pane-label">输出</span><h2 id="result-title">{app.config.compareMode ? "对照译文" : "译文"}</h2><span class="pane-context">{app.config.compareMode ? "多引擎" : engineLabel(app.config.defaultEngine)}</span></div>
          {#if app.translation.isProcessing && app.translation.output}<span class="updating"><i></i>正在更新</span>{/if}
        </header>
        {#if app.config.compareMode}
          <div class="compare-wrap"><ComparePanel engines={app.config.compareEngines} outputs={app.translation.compareOutputs} loading={app.translation.compareLoadingEngines} copied={app.copiedEngines} speakingText={app.speakingText} target={app.target} isConfigured={app.isConfigured} onToggle={app.toggleCompareEngine} onCopy={(engine) => void app.copyText(app.translation.compareOutputs[engine]?.text ?? "", engine)} onSpeak={app.speak} onSettings={() => app.openPanel("config")} /></div>
        {:else if app.translation.output}
          <textarea class="editor" readonly value={app.translation.output} aria-label="译文"></textarea>
          {#if app.translation.notice}<div class="result-notice" role="status"><AlertCircle size={13} />{app.translation.notice}</div>{/if}
          <footer class="pane-footer">
            <span class="pane-note">{langs[app.target] ?? app.target}</span>
            <div class="pane-actions">
              <button class="icon-action" onclick={() => app.speak(app.translation.output, app.target)} aria-label="朗读译文" title="朗读译文">{#if app.speakingText === app.translation.output}<Square size={13} />{:else}<Volume2 size={14} />{/if}</button>
              <button onclick={app.retranslate} aria-label="重新翻译译文" title="重新翻译"><CornerDownLeft size={14} /><span class="action-label">再翻译</span></button>
              <button class:success={app.copied} onclick={() => void app.copyText(app.translation.output)} aria-label={app.copied ? "译文已复制" : "复制译文"} title="复制译文">{#if app.copied}<Check size={14} /><span class="action-label">已复制</span>{:else}<Copy size={14} /><span class="action-label">复制</span>{/if}</button>
            </div>
          </footer>
        {:else}
          <div class="result-empty"><span class="empty-icon"><Languages size={23} strokeWidth={1.4} /></span><strong>等待翻译</strong><span>译文将在这里显示</span></div>
        {/if}
      </section>
    </div>
    <StatusBar status={app.translation.status} bridgeKind={desktopBridge.kind} source={app.source} target={app.target} activeEngine={app.config.defaultEngine} compareMode={app.config.compareMode} />
  </main>

  <Config open={app.showConfig} config={app.config} onClose={app.closePanels} onSave={(next) => app.configController.save(next)} onTest={(engine, service) => desktopBridge.testConnection(engine, service)} />
  <History open={app.showHistory} queryHistory={(query) => desktopBridge.queryHistory(query)} onClear={() => desktopBridge.clearHistory()} onExport={() => desktopBridge.exportHistory()} onError={(message) => app.translateController.showError(message)} onClose={app.closePanels} onSelect={app.restoreHistoryEntry} />
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
  .result-notice { display: flex; min-height: 30px; flex: 0 0 auto; align-items: center; gap: var(--sp-2); border-top: 1px solid var(--warning-border); padding: 0 var(--sp-3); background: var(--warning-soft); color: var(--text-sec); font-size: var(--fs-xs); }
  .result-notice :global(svg) { flex: 0 0 auto; color: var(--warning); }
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
