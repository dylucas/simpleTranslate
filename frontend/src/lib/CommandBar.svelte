<script lang="ts">
  import { ArrowLeftRight, Clipboard, Columns2, LoaderCircle, Send, Zap } from "@lucide/svelte";
  import { langs } from "./languages";
  import type { EngineId } from "./types";

  interface Props {
    source: string;
    target: string;
    activeEngine: EngineId;
    autoTranslate: boolean;
    autoDetectLang: string;
    detectedSource: string;
    clipboardWatch: boolean;
    compareMode: boolean;
    isProcessing: boolean;
    canTranslate: boolean;
    unavailableReason: string;
    onSource: (value: string) => void;
    onTarget: (value: string) => void;
    onEngine: (engine: EngineId) => void;
    onSwap: () => void;
    onAuto: (value: boolean) => void;
    onClipboard: () => void;
    onCompare: () => void;
    onTranslate: () => void;
  }

  let {
    source, target, activeEngine, autoTranslate, autoDetectLang, detectedSource,
    clipboardWatch, compareMode, isProcessing, canTranslate, unavailableReason,
    onSource, onTarget, onEngine, onSwap, onAuto, onClipboard, onCompare, onTranslate,
  }: Props = $props();

  let canSwap = $derived(source !== "auto" || Boolean(detectedSource));
</script>

<header class="command-bar">
  <div class="route-control" role="group" aria-label="翻译语言">
    <label class="language-select">
      <span>原文</span>
      <select value={source} onchange={(event) => onSource(event.currentTarget.value)} aria-label="源语言">
        <option value="auto">{autoDetectLang}</option>
        {#each Object.entries(langs) as [code, name]}<option value={code}>{name}</option>{/each}
      </select>
    </label>
    <button class="swap" onclick={onSwap} disabled={!canSwap} aria-label={canSwap ? "交换语言" : "识别语言后即可交换"} title={canSwap ? "交换语言" : "识别语言后即可交换"}>
      <ArrowLeftRight size={15} />
    </button>
    <label class="language-select">
      <span>译文</span>
      <select value={target} onchange={(event) => onTarget(event.currentTarget.value)} aria-label="目标语言">
        {#each Object.entries(langs) as [code, name]}<option value={code}>{name}</option>{/each}
      </select>
    </label>
  </div>

  <div class="command-actions">
    <div class="engine-switch" role="group" aria-label="默认翻译引擎">
      <button class:active={activeEngine === "tencent"} aria-pressed={activeEngine === "tencent"} onclick={() => onEngine("tencent")}>混元</button>
      <button class:active={activeEngine === "aliyun"} aria-pressed={activeEngine === "aliyun"} onclick={() => onEngine("aliyun")}>阿里云</button>
    </div>

    <div class="mode-toggles" role="group" aria-label="翻译模式">
      <button class:active={autoTranslate} aria-pressed={autoTranslate} aria-label="自动翻译" onclick={() => onAuto(!autoTranslate)} title="自动翻译"><Zap size={14} /><span>自动</span></button>
      <button class:active={clipboardWatch} aria-pressed={clipboardWatch} aria-label="剪贴板监听" onclick={onClipboard} title="剪贴板监听"><Clipboard size={14} /><span>剪贴板</span></button>
      <button class:active={compareMode} aria-pressed={compareMode} aria-label="多引擎对照" onclick={onCompare} title="多引擎对照"><Columns2 size={14} /><span>对照</span></button>
    </div>

    <button class="translate" onclick={onTranslate} disabled={!canTranslate || isProcessing} aria-busy={isProcessing} aria-label="翻译" title={unavailableReason || "翻译"}>
      {#if isProcessing}<LoaderCircle size={15} class="spin" />{:else}<Send size={15} />{/if}<span>{isProcessing ? "翻译中" : "翻译"}</span>
    </button>
  </div>
</header>

<style>
  .command-bar {
    display: flex;
    min-height: var(--commandbar-h);
    flex: 0 0 auto;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-3);
    border-bottom: 1px solid var(--border-soft);
    padding: 6px var(--sp-3);
    background: var(--bg-panel);
  }
  .route-control, .engine-switch, .mode-toggles { display: flex; align-items: center; }
  .route-control { min-width: 292px; gap: 2px; }
  .language-select {
    display: grid;
    min-width: 112px;
    flex: 1;
    grid-template-columns: auto minmax(0, 1fr);
    align-items: center;
    gap: var(--sp-2);
    height: 34px;
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    padding: 0 var(--sp-2);
    background: var(--bg-input);
  }
  .language-select:hover { border-color: var(--border-strong); }
  .language-select:focus-within { border-color: var(--primary); box-shadow: 0 0 0 2px var(--primary-soft); }
  .language-select span { color: var(--text-muted); font-size: var(--fs-xs); }
  select { min-width: 0; width: 100%; border: 0; outline: 0; background: transparent; color: var(--text-main); cursor: pointer; font: var(--fw-semibold) var(--fs-base)/1.4 var(--font-sans); }
  select option { background: var(--bg-elevated); }
  button { display: inline-flex; align-items: center; justify-content: center; border: 0; font-family: inherit; cursor: pointer; }
  .swap { width: 30px; height: 30px; flex: 0 0 auto; border-radius: var(--radius-sm); background: transparent; color: var(--text-muted); }
  .swap:hover:not(:disabled) { background: var(--bg-hover); color: var(--primary); }
  .swap:disabled { cursor: not-allowed; opacity: .32; }
  .command-actions { display: flex; min-width: 0; align-items: center; justify-content: flex-end; gap: var(--sp-2); }
  .engine-switch, .mode-toggles { gap: 2px; border: 1px solid var(--border); border-radius: var(--radius-md); padding: 2px; background: var(--bg-input); }
  .engine-switch button { min-width: 54px; height: 28px; border-radius: var(--radius-sm); padding: 0 var(--sp-2); background: transparent; color: var(--text-sec); font-size: var(--fs-xs); }
  .engine-switch button.active { background: var(--bg-elevated); color: var(--primary); box-shadow: var(--shadow-sm); font-weight: var(--fw-semibold); }
  .mode-toggles button { height: 28px; gap: 5px; border-radius: var(--radius-sm); padding: 0 7px; background: transparent; color: var(--text-muted); font-size: var(--fs-xs); }
  .mode-toggles button:hover { background: var(--bg-hover); color: var(--text-main); }
  .mode-toggles button.active { background: var(--primary-soft); color: var(--primary); }
  .translate { min-width: 78px; height: 34px; gap: var(--sp-2); border-radius: var(--radius-md); padding: 0 var(--sp-3); background: var(--primary); color: var(--text-inverse); font-size: var(--fs-sm); font-weight: var(--fw-semibold); box-shadow: var(--shadow-sm); }
  .translate:hover:not(:disabled) { background: var(--primary-hover); }
  .translate:active:not(:disabled) { background: var(--primary-active); }
  .translate:disabled { cursor: not-allowed; opacity: .42; }
  .translate :global(.spin) { animation: spin .9s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 1100px) {
    .mode-toggles button { width: 30px; padding: 0; }
    .mode-toggles span { display: none; }
    .engine-switch button { min-width: 48px; }
  }
  @media (max-width: 720px) {
    .command-bar { min-height: 88px; align-items: stretch; flex-direction: column; gap: 6px; padding: 6px var(--sp-2); }
    .route-control { width: 100%; min-width: 0; }
    .language-select { min-width: 0; }
    .command-actions { width: 100%; justify-content: space-between; }
    .translate { flex: 1; max-width: 88px; }
  }
  @media (max-width: 460px) {
    .language-select { grid-template-columns: 1fr; gap: 0; }
    .language-select span { display: none; }
    .engine-switch button { min-width: 42px; padding-inline: 5px; }
    .translate { width: 34px; min-width: 34px; max-width: 34px; padding: 0; }
    .translate span { display: none; }
  }
</style>
