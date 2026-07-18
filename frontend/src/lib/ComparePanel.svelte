<script lang="ts">
  import { Copy, Check, Volume2, Square } from "lucide-svelte";
  import { createEventDispatcher } from "svelte";
  import type { EngineTranslateResult } from "./types";

  export let compareEngines: string[] = [];
  export let compareOutputs: Record<string, EngineTranslateResult> = {};
  export let compareLoadingEngines: Record<string, boolean> = {};
  export let copiedEngines: Record<string, boolean> = {};
  export let speakingText: string | null = null;
  export let target = "zh";

  const dispatch = createEventDispatcher<{
    copy: string;
    speak: { text: string; lang: string };
    toggleEngine: string;
  }>();
</script>

<div class="compare-engine-bar">
  <span class="engine-bar-label">对照引擎</span>
  <div class="engine-toggle-pills" role="group" aria-label="参与对照的翻译引擎">
    {#each ["tencent", "aliyun"] as eng}
      <button
        class="toggle-pill"
        class:active={compareEngines.includes(eng)}
        aria-pressed={compareEngines.includes(eng)}
        disabled={compareEngines.includes(eng) && compareEngines.length <= 1}
        on:click={() => dispatch("toggleEngine", eng)}
      >
        {eng === "tencent" ? "混元" : "阿里"}
      </button>
    {/each}
  </div>
</div>

<div class="compare-grid">
  {#each compareEngines as eng}
    <div class="compare-card" class:refreshing={compareLoadingEngines[eng]} aria-busy={compareLoadingEngines[eng]}>
      <div class="compare-header">
        <span class="compare-title"><span class="engine-dot"></span>{eng === "tencent" ? "混元" : "阿里"}</span>
        <div class="compare-header-right">
          {#if compareLoadingEngines[eng] && compareOutputs?.[eng]?.text && !compareOutputs?.[eng]?.error}
            <span class="compare-refreshing" title="正在更新">刷新中</span>
          {/if}
          {#if compareOutputs?.[eng]?.error}
            <span class="compare-error">失败</span>
          {/if}
          <button
            class="compare-copy-btn"
            class:active={speakingText === compareOutputs?.[eng]?.text}
            disabled={!compareOutputs?.[eng]?.text || !!compareOutputs?.[eng]?.error}
            on:click={() => dispatch("speak", { text: compareOutputs?.[eng]?.text || "", lang: target })}
            title={speakingText === compareOutputs?.[eng]?.text ? "停止朗读" : "朗读此结果"}
            aria-label={speakingText === compareOutputs?.[eng]?.text ? `停止朗读${eng === "tencent" ? "混元" : "阿里"}译文` : `朗读${eng === "tencent" ? "混元" : "阿里"}译文`}
          >
            {#if speakingText === compareOutputs?.[eng]?.text}
              <Square size={12} fill="currentColor" />
            {:else}
              <Volume2 size={12} />
            {/if}
          </button>
          <button
            class="compare-copy-btn"
            class:success={copiedEngines[eng]}
            disabled={!compareOutputs?.[eng]?.text || !!compareOutputs?.[eng]?.error}
            on:click={() => dispatch("copy", eng)}
            title={copiedEngines[eng] ? "已复制" : "复制此结果"}
            aria-label={copiedEngines[eng] ? `${eng === "tencent" ? "混元" : "阿里"}译文已复制` : `复制${eng === "tencent" ? "混元" : "阿里"}译文`}
          >
            {#if copiedEngines[eng]}
              <Check size={12} />
            {:else}
              <Copy size={12} />
            {/if}
          </button>
        </div>
      </div>
      {#if compareLoadingEngines[eng] && !compareOutputs?.[eng]?.text && !compareOutputs?.[eng]?.error}
        <div class="skeleton-line"></div>
      {:else}
        <textarea
          readonly
          value={compareOutputs?.[eng]?.text || (compareOutputs?.[eng]?.error ? `错误：${compareOutputs[eng].error}` : "")}
          placeholder="翻译结果..."
          aria-label={`${eng === "tencent" ? "混元" : "阿里"}译文`}
          spellcheck="false"
        ></textarea>
      {/if}
    </div>
  {:else}
    <div class="empty-compare">请至少选择一个对照引擎</div>
  {/each}
</div>

<style>
  .compare-engine-bar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-3);
    margin-bottom: var(--sp-3);
    flex-shrink: 0;
  }
  .engine-bar-label {
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: var(--tracking-wider);
  }
  .engine-toggle-pills {
    display: flex;
    gap: var(--sp-1);
    padding: var(--sp-1);
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-full);
  }
  .toggle-pill {
    border: none;
    background: transparent;
    color: var(--text-sec);
    padding: var(--sp-1) var(--sp-3);
    border-radius: var(--radius-full);
    font-size: var(--fs-sm);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard);
  }
  .toggle-pill.active {
    color: var(--primary);
    background: var(--primary-soft);
  }
  .toggle-pill:hover {
    color: var(--text-main);
  }
  .toggle-pill:disabled {
    opacity: 0.7;
  }
  .compare-grid {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    height: 100%;
    overflow-y: auto;
  }
  .compare-card {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: var(--sp-3);
    background: var(--bg-surface);
    overflow: hidden;
    box-shadow: var(--shadow-sm);
    transition: opacity var(--t-base) var(--ease-standard),
      border-color var(--t-base) var(--ease-standard);
  }
  .compare-card.refreshing {
    opacity: 0.85;
  }
  .compare-refreshing {
    color: var(--primary);
    font-weight: var(--fw-medium);
    font-size: var(--fs-xs);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-full);
    background: var(--primary-soft);
    animation: pulse 1.2s var(--ease-standard) infinite;
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; }
  }
  .compare-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--sp-2);
    font-size: var(--fs-sm);
    color: var(--text-sec);
  }
  .compare-header-right {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    min-width: var(--sp-10);
    justify-content: flex-end;
  }
  .compare-title {
    font-weight: 700;
    color: var(--text-main);
    display: flex;
    align-items: center;
    gap: var(--sp-2);
  }
  .engine-dot {
    width: var(--sp-2);
    height: var(--sp-2);
    background: var(--primary);
    border-radius: var(--radius-full);
    box-shadow: 0 0 0 var(--sp-1) var(--primary-soft);
  }
  .compare-error {
    color: var(--danger);
    font-weight: var(--fw-bold);
  }
  .compare-copy-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-sec);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-sm);
    font-size: var(--fs-sm);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard),
      border-color var(--t-base) var(--ease-standard);
    min-width: var(--sp-8);
    height: var(--sp-8);
  }
  .compare-copy-btn:disabled {
    opacity: 0.32;
  }
  .compare-copy-btn:hover {
    border-color: var(--text-sec);
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .compare-copy-btn.success {
    border-color: var(--success);
    color: var(--success);
    background: var(--success-soft);
  }
  .compare-copy-btn.active {
    border-color: var(--primary);
    color: var(--primary);
    background: var(--primary-soft);
  }
  .compare-card :global(textarea) {
    flex: 1;
    min-height: 0;
    margin-top: var(--sp-1);
    padding: var(--sp-3) var(--sp-1) 0;
    border-top: 1px solid var(--border-soft);
    font-size: var(--fs-md);
    background: transparent;
    border-left: none;
    border-right: none;
    border-bottom: none;
    resize: none;
    color: var(--text-main);
    font-family: inherit;
    line-height: var(--lh-relaxed);
  }
  .compare-card :global(textarea:focus-visible) {
    outline: 2px solid var(--primary);
    outline-offset: -2px;
  }
  .compare-card :global(textarea::placeholder) {
    color: var(--text-muted);
    opacity: 1;
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
  @keyframes shimmer {
    0% { background-position: 200% 0; }
    100% { background-position: -200% 0; }
  }
  .empty-compare {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-sec);
    font-size: var(--fs-base);
  }
</style>
