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
  <div class="engine-toggle-pills">
    {#each ["tencent", "aliyun"] as eng}
      <button
        class="toggle-pill"
        class:active={compareEngines.includes(eng)}
        on:click={() => dispatch("toggleEngine", eng)}
      >
        {eng === "tencent" ? "混元" : "阿里"}
      </button>
    {/each}
  </div>
</div>

<div class="compare-grid">
  {#each compareEngines as eng}
    <div class="compare-card" class:refreshing={compareLoadingEngines[eng]}>
      <div class="compare-header">
        <span class="compare-title">{eng === "tencent" ? "混元" : "阿里"}</span>
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
            class:disabled={!compareOutputs?.[eng]?.text || compareOutputs?.[eng]?.error}
            on:click={() => dispatch("speak", { text: compareOutputs?.[eng]?.text || "", lang: target })}
            title="朗读"
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
            class:disabled={!compareOutputs?.[eng]?.text || compareOutputs?.[eng]?.error}
            on:click={() => dispatch("copy", eng)}
            title="复制此结果"
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
    gap: 10px;
    margin-bottom: 8px;
    flex-shrink: 0;
  }
  .engine-bar-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-sec);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .engine-toggle-pills {
    display: flex;
    gap: 6px;
  }
  .toggle-pill {
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-sec);
    padding: 4px 12px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }
  .toggle-pill.active {
    border-color: var(--primary);
    color: var(--primary);
    background: var(--primary-soft);
  }
  .toggle-pill:hover {
    color: var(--text-main);
  }
  .compare-grid {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
    overflow-y: auto;
  }
  .compare-card {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 8px 10px;
    background: rgba(0, 0, 0, 0.08);
    overflow: hidden;
    transition: opacity 0.2s;
  }
  .compare-card.refreshing {
    opacity: 0.85;
  }
  .compare-refreshing {
    color: var(--primary);
    font-weight: var(--fw-medium);
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--primary-bg, rgba(99, 102, 241, 0.12));
    animation: pulse 1.2s ease-in-out infinite;
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; }
  }
  .compare-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
    font-size: 12px;
    color: var(--text-sec);
  }
  .compare-header-right {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 40px;
    justify-content: flex-end;
  }
  .compare-title {
    font-weight: 700;
    color: var(--text-main);
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .compare-error {
    color: var(--danger);
    font-weight: var(--fw-bold);
  }
  .compare-copy-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-sec);
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    min-width: 28px;
    height: 24px;
  }
  .compare-copy-btn.disabled {
    visibility: hidden;
    pointer-events: none;
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
    margin-top: 4px;
    padding: 6px 0 0;
    border-top: 1px dashed var(--border);
    font-size: 14px;
    background: transparent;
    border-left: none;
    border-right: none;
    border-bottom: none;
    resize: none;
    outline: none;
    color: var(--text-main);
    font-family: inherit;
    line-height: var(--lh-relaxed);
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
    font-size: 13px;
    opacity: 0.6;
  }
</style>
