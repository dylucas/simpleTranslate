<script lang="ts">
  import { Check, Copy, Settings, Square, Volume2 } from "@lucide/svelte";
  import type { EngineId, EngineTranslateResult } from "./types";

  interface Props {
    engines: EngineId[];
    outputs: Partial<Record<EngineId, EngineTranslateResult>>;
    loading: Partial<Record<EngineId, boolean>>;
    copied: Partial<Record<EngineId, boolean>>;
    speakingText: string | null;
    target: string;
    isConfigured: (engine: EngineId) => boolean;
    onToggle: (engine: EngineId) => void;
    onCopy: (engine: EngineId) => void;
    onSpeak: (text: string, language: string) => void;
    onSettings: () => void;
  }

  let { engines, outputs, loading, copied, speakingText, target, isConfigured, onToggle, onCopy, onSpeak, onSettings }: Props = $props();
  const allEngines: EngineId[] = ["tencent", "aliyun"];
  const label = (engine: EngineId) => engine === "tencent" ? "混元" : "阿里云";
</script>

<div class="compare-toolbar">
  <span>对照引擎</span>
  <div class="segments" role="group" aria-label="参与对照的翻译引擎">
    {#each allEngines as engine}
      <button class:active={engines.includes(engine)} aria-pressed={engines.includes(engine)} disabled={engines.includes(engine) && engines.length === 1} onclick={() => onToggle(engine)}>{label(engine)}</button>
    {/each}
  </div>
</div>

<div class="compare-grid">
  {#each engines as engine}
    {@const result = outputs[engine]}
    <section class="engine-result" class:updating={loading[engine]} aria-busy={loading[engine]}>
      <header>
        <span class="engine-name"><i></i>{label(engine)}</span>
        <div class="result-actions">
          {#if loading[engine]}<span class="state">更新中</span>{/if}
          {#if result?.error}<span class="state error">失败</span>{/if}
          <button disabled={!result?.text || Boolean(result.error)} class:active={speakingText === result?.text} onclick={() => onSpeak(result?.text ?? "", target)} aria-label={`朗读${label(engine)}译文`} title="朗读">
            {#if speakingText === result?.text}<Square size={13} />{:else}<Volume2 size={14} />{/if}
          </button>
          <button disabled={!result?.text || Boolean(result.error)} class:success={copied[engine]} onclick={() => onCopy(engine)} aria-label={`复制${label(engine)}译文`} title="复制">
            {#if copied[engine]}<Check size={14} />{:else}<Copy size={14} />{/if}
          </button>
        </div>
      </header>

      {#if !isConfigured(engine)}
        <div class="engine-empty">
          <Settings size={20} />
          <span>尚未配置{label(engine)}凭据</span>
          <button onclick={onSettings}>前往设置</button>
        </div>
      {:else if loading[engine] && !result?.text}
        <div class="skeleton"><span></span><span></span><span></span></div>
      {:else if result?.error}
        <div class="engine-empty error-copy">{result.error}</div>
      {:else}
        <textarea readonly value={result?.text ?? ""} placeholder="等待翻译结果" aria-label={`${label(engine)}译文`}></textarea>
      {/if}
    </section>
  {/each}
</div>

<style>
  .compare-toolbar { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-3); color: var(--text-muted); font-size: var(--fs-xs); font-weight: var(--fw-semibold); }
  .segments { display: flex; gap: 2px; border: 1px solid var(--border); border-radius: var(--radius-md); padding: 2px; background: var(--bg-input); }
  button { border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text-sec); cursor: pointer; font-family: inherit; }
  .segments button { min-height: 28px; padding: 0 var(--sp-3); font-size: var(--fs-sm); }
  .segments button.active { background: var(--bg-elevated); color: var(--primary); }
  button:disabled { cursor: not-allowed; opacity: .4; }
  .compare-grid { display: grid; min-height: 0; flex: 1; grid-template-columns: 1fr; gap: var(--sp-3); overflow: auto; }
  .engine-result { display: flex; min-height: 150px; flex-direction: column; overflow: hidden; border: 1px solid var(--border); border-radius: var(--radius-md); background: var(--bg-surface); }
  .engine-result.updating { border-color: color-mix(in srgb, var(--primary) 45%, var(--border)); }
  .engine-result header { display: flex; min-height: 38px; flex: 0 0 auto; align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border-soft); padding: 0 var(--sp-3); }
  .engine-name { display: flex; align-items: center; gap: var(--sp-2); color: var(--text-sec); font-size: var(--fs-sm); font-weight: var(--fw-semibold); }
  .engine-name i { width: 7px; height: 7px; border-radius: 50%; background: var(--primary); }
  .result-actions { display: flex; align-items: center; gap: var(--sp-1); }
  .result-actions button { display: grid; width: 28px; height: 28px; place-items: center; }
  .result-actions button:hover:not(:disabled) { background: var(--bg-hover); color: var(--text-main); }
  .result-actions button.success { color: var(--success); }
  .state { color: var(--primary); font-size: var(--fs-xs); }
  .state.error, .error-copy { color: var(--danger); }
  textarea { width: 100%; min-height: 0; flex: 1; resize: none; border: 0; outline: 0; padding: var(--sp-3); background: transparent; color: var(--text-main); font: var(--fw-regular) var(--fs-md)/var(--lh-relaxed) var(--font-sans); }
  .engine-empty { display: flex; min-height: 110px; flex: 1; align-items: center; justify-content: center; gap: var(--sp-2); padding: var(--sp-4); color: var(--text-muted); font-size: var(--fs-sm); }
  .engine-empty button { color: var(--primary); text-decoration: underline; }
  .skeleton { display: grid; gap: var(--sp-3); padding: var(--sp-4); }
  .skeleton span { height: 10px; border-radius: var(--radius-sm); background: var(--bg-hover); animation: shimmer 1.2s infinite alternate; }
  .skeleton span:nth-child(2) { width: 82%; }
  .skeleton span:nth-child(3) { width: 58%; }
  @keyframes shimmer { to { opacity: .45; } }
  @media (min-width: 1200px) { .compare-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } }
</style>
