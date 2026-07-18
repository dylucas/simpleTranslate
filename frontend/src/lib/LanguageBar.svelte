<script lang="ts">
  import { ArrowLeftRight, Clipboard, Columns2, LoaderCircle, Send, Zap } from "@lucide/svelte";
  import { langs } from "./languages";

  interface Props {
    source: string;
    target: string;
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
    onSwap: () => void;
    onAuto: (value: boolean) => void;
    onClipboard: () => void;
    onCompare: () => void;
    onTranslate: () => void;
  }

  let {
    source, target, autoTranslate, autoDetectLang, detectedSource, clipboardWatch,
    compareMode, isProcessing, canTranslate, unavailableReason, onSource, onTarget,
    onSwap, onAuto, onClipboard, onCompare, onTranslate,
  }: Props = $props();

  let canSwap = $derived(source !== "auto" || Boolean(detectedSource));
</script>

<header class="workspace-header">
  <div class="language-pair" role="group" aria-label="翻译语言">
    <label>
      <span>源语言</span>
      <select value={source} onchange={(event) => onSource(event.currentTarget.value)} aria-label="源语言">
        <option value="auto">{autoDetectLang}</option>
        {#each Object.entries(langs) as [code, name]}<option value={code}>{name}</option>{/each}
      </select>
    </label>
    <button class="swap" onclick={onSwap} disabled={!canSwap} aria-label={canSwap ? "交换语言" : "识别语言后即可交换"} title={canSwap ? "交换语言" : "识别语言后即可交换"}>
      <ArrowLeftRight size={16} />
    </button>
    <label>
      <span>目标语言</span>
      <select value={target} onchange={(event) => onTarget(event.currentTarget.value)} aria-label="目标语言">
        {#each Object.entries(langs) as [code, name]}<option value={code}>{name}</option>{/each}
      </select>
    </label>
  </div>

  <div class="actions">
    <div class="mode-segments" role="group" aria-label="翻译模式">
      <button class:active={autoTranslate} aria-pressed={autoTranslate} onclick={() => onAuto(!autoTranslate)} title="自动翻译"><Zap size={14} /><span>自动</span></button>
      <button class:active={clipboardWatch} aria-pressed={clipboardWatch} onclick={onClipboard} title="剪贴板监听"><Clipboard size={14} /><span>剪贴板</span></button>
      <button class:active={compareMode} aria-pressed={compareMode} onclick={onCompare} title="多引擎对照"><Columns2 size={14} /><span>对照</span></button>
    </div>
    <button class="translate" onclick={onTranslate} disabled={!canTranslate || isProcessing} aria-busy={isProcessing} title={unavailableReason || "翻译"}>
      {#if isProcessing}<LoaderCircle size={15} class="spin" />{:else}<Send size={15} />{/if}<span>{isProcessing ? "翻译中" : "翻译"}</span>
    </button>
  </div>
</header>

<style>
  .workspace-header { display: flex; min-height: var(--header-h); flex: 0 0 auto; align-items: center; justify-content: space-between; gap: var(--sp-3); border-bottom: 1px solid var(--border-soft); padding: var(--sp-2) var(--sp-4); background: var(--bg-panel); }
  .language-pair, .mode-segments { display: flex; align-items: center; gap: 2px; border: 1px solid var(--border); border-radius: var(--radius-md); padding: 2px; background: var(--bg-input); box-shadow: inset 0 1px 0 color-mix(in srgb, var(--text-main) 3%, transparent); }
  label { display: grid; min-width: 116px; gap: 1px; padding: var(--sp-1) var(--sp-2); border-radius: var(--radius-sm); }
  label:hover { background: var(--bg-hover); }
  label span { color: var(--text-muted); font-size: var(--fs-xs); }
  select { width: 100%; border: 0; outline: 0; background: transparent; color: var(--text-main); cursor: pointer; font: 600 var(--fs-base)/1.4 var(--font-sans); }
  select option { background: var(--bg-elevated); }
  button { display: inline-flex; align-items: center; justify-content: center; border: 0; font-family: inherit; cursor: pointer; }
  .swap { width: 32px; height: 32px; flex: 0 0 auto; border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--bg-surface); color: var(--text-sec); }
  .swap:hover:not(:disabled) { border-color: var(--primary); color: var(--primary); }
  .swap:disabled { cursor: not-allowed; opacity: .35; }
  .actions { display: flex; align-items: center; gap: var(--sp-2); }
  .mode-segments button { min-height: 32px; gap: var(--sp-1); border-radius: var(--radius-sm); padding: 0 var(--sp-2); background: transparent; color: var(--text-sec); font-size: var(--fs-sm); }
  .mode-segments button:hover { color: var(--text-main); }
  .mode-segments button.active { background: var(--bg-elevated); color: var(--primary); box-shadow: var(--shadow-sm); font-weight: var(--fw-semibold); }
  .translate { min-width: 84px; min-height: 38px; gap: var(--sp-2); border-radius: var(--radius-md); padding: 0 var(--sp-4); background: var(--primary); color: white; font-weight: var(--fw-semibold); box-shadow: var(--shadow-sm); }
  .translate:hover:not(:disabled) { background: var(--primary-hover); }
  .translate:disabled { cursor: not-allowed; opacity: .45; }
  .translate :global(.spin) { animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 980px) { .mode-segments span { display: none; } .mode-segments button { width: 32px; padding: 0; } }
  @media (max-width: 720px) { .workspace-header { align-items: stretch; flex-direction: column; padding: var(--sp-2); } .language-pair { width: 100%; } label { min-width: 0; flex: 1; } .actions { justify-content: space-between; } .translate { min-width: 0; flex: 1; } }
  @media (max-width: 420px) { .translate span { display: none; } .translate { max-width: 46px; flex-basis: 46px; padding: 0; } }
</style>
