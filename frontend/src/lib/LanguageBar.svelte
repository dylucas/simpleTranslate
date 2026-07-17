<script lang="ts">
  import { ArrowLeftRight, Zap, Clipboard } from "lucide-svelte";
  import { createEventDispatcher } from "svelte";
  import { langs } from "./languages";

  // bind:source / bind:target / bind:autoTranslate 支持双向绑定
  export let source = "auto";
  export let target = "zh";
  export let autoTranslate = true;
  export let autoDetectLang = "自动识别";
  export let clipboardWatch = false;
  export let compareMode = false;
  export let status = "准备就绪";

  const dispatch = createEventDispatcher<{
    swap: void;
    toggleClipboard: void;
    toggleCompare: void;
    translate: void;
  }>();

  function swapLangs(): void {
    const tmp = source;
    source = target;
    target = tmp;
  }
</script>

<header class="workspace-header">
  <div class="lang-bar">
    <div class="select-wrapper">
      <select bind:value={source}>
        <option value="auto">{autoDetectLang}</option>
        {#each Object.entries(langs) as [code, name]}
          <option value={code}>{name}</option>
        {/each}
      </select>
    </div>

    <button class="swap-btn" on:click={swapLangs} title="交换语言">
      <ArrowLeftRight size={16} />
    </button>

    <div class="select-wrapper">
      <select bind:value={target}>
        {#each Object.entries(langs) as [code, name]}
          <option value={code}>{name}</option>
        {/each}
      </select>
    </div>
  </div>

  <div class="right-tools">
    <button
      class="mode-btn"
      class:active={autoTranslate}
      on:click={() => (autoTranslate = !autoTranslate)}
      title={autoTranslate ? "关闭自动翻译" : "开启自动翻译"}
    >
      <Zap size={13} />
      自动
    </button>
    <button
      class="mode-btn"
      class:active={clipboardWatch}
      on:click={() => dispatch("toggleClipboard")}
      title={clipboardWatch ? "关闭剪贴板监听" : "开启剪贴板监听"}
    >
      <Clipboard size={13} />
      剪贴板
    </button>
    <button
      class="mode-btn"
      class:active={compareMode}
      on:click={() => dispatch("toggleCompare")}
      title="多引擎对照"
    >
      对照
    </button>
    <button
      class="translate-btn"
      on:click={() => dispatch("translate")}
      disabled={status === "翻译中..."}
    >
      <span>{status === "翻译中..." ? "翻译中" : "翻译"}</span>
      {#if status === "翻译中..."}
        <span class="loading-dots">...</span>
      {/if}
    </button>
  </div>
</header>

<style>
  .workspace-header {
    height: var(--header-h);
    padding: 0 var(--sp-6);
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
    gap: var(--sp-4);
  }
  .lang-bar {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
  }
  .select-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
  }
  .select-wrapper select {
    appearance: none;
    -webkit-appearance: none;
    background: transparent;
    border: none;
    color: var(--text-main);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    padding: var(--sp-2) 26px var(--sp-2) var(--sp-3);
    border-radius: var(--radius-sm);
    outline: none;
    transition: background var(--t-base) var(--ease-standard),
      color var(--t-base) var(--ease-standard);
  }
  .select-wrapper select:hover {
    background: var(--bg-hover);
  }
  .select-wrapper::after {
    content: "";
    position: absolute;
    right: 10px;
    width: 7px;
    height: 7px;
    border-right: 2px solid var(--text-sec);
    border-bottom: 2px solid var(--text-sec);
    transform: rotate(45deg) translateY(-2px);
    pointer-events: none;
    transition: border-color var(--t-base) var(--ease-standard);
  }
  .select-wrapper:hover::after {
    border-color: var(--primary);
  }
  .select-wrapper select option {
    background: var(--bg-elevated);
    color: var(--text-main);
  }
  .swap-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    padding: var(--sp-2);
    border-radius: var(--radius-full);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
    display: flex;
  }
  .swap-btn:hover {
    background: var(--bg-hover);
    color: var(--primary);
    transform: rotate(180deg);
  }
  .right-tools {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .mode-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-sec);
    padding: 8px 12px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }
  .mode-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .mode-btn.active {
    border-color: var(--primary);
    color: var(--primary);
    background: var(--primary-soft);
  }
  .translate-btn {
    background: var(--accent-grad);
    color: var(--text-inverse);
    border: none;
    padding: var(--sp-2) var(--sp-6);
    border-radius: var(--radius-full);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    box-shadow: var(--shadow-glow);
    transition: transform var(--t-fast) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard),
      filter var(--t-base) var(--ease-standard);
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .translate-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    filter: brightness(1.08);
    box-shadow: 0 10px 28px -4px var(--accent-glow);
  }
  .translate-btn:active {
    transform: scale(0.96);
  }
  .translate-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
  .loading-dots {
    display: inline-block;
    letter-spacing: 2px;
    animation: dotsPulse 1.2s var(--ease-standard) infinite;
  }
  @keyframes dotsPulse {
    0%, 100% { opacity: 0.35; }
    50% { opacity: 1; }
  }
  @media (max-width: 720px) {
    .workspace-header {
      padding: 0 var(--sp-3);
    }
    .right-tools {
      gap: var(--sp-1);
    }
    .mode-btn {
      padding: var(--sp-1) var(--sp-2);
    }
    .mode-btn :global(svg) {
      display: none;
    }
  }
</style>
