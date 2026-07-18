<script lang="ts">
  import { ArrowLeftRight, Zap, Clipboard, Columns2, Send } from "lucide-svelte";
  import { createEventDispatcher } from "svelte";
  import { langs } from "./languages";

  // bind:source / bind:target / bind:autoTranslate 支持双向绑定
  export let source = "auto";
  export let target = "zh";
  export let autoTranslate = true;
  export let autoDetectLang = "自动识别";
  export let detectedSource = "";
  export let clipboardWatch = false;
  export let compareMode = false;
  export let status = "准备就绪";
  export let isProcessing = false;
  export let canTranslate = true;

  const dispatch = createEventDispatcher<{
    swap: void;
    toggleClipboard: void;
    toggleCompare: void;
    translate: void;
  }>();

  function swapLangs(): void {
    const resolvedSource = source === "auto" ? detectedSource : source;
    if (!resolvedSource) return;
    const tmp = resolvedSource;
    source = target;
    target = tmp;
  }

  $: translating = isProcessing || status === "翻译中...";
</script>

<header class="workspace-header">
  <div class="lang-bar" role="group" aria-label="翻译语言">
    <div class="language-control">
      <label for="source-language">源语言</label>
      <div class="select-wrapper">
        <select id="source-language" bind:value={source} aria-label="源语言">
          <option value="auto">{autoDetectLang}</option>
          {#each Object.entries(langs) as [code, name]}
            <option value={code}>{name}</option>
          {/each}
        </select>
      </div>
    </div>

    <button
      class="swap-btn"
      on:click={swapLangs}
      title={source === "auto" && !detectedSource ? "识别语言后即可交换" : "交换语言"}
      aria-label={source === "auto" && !detectedSource ? "尚未识别源语言，暂时无法交换" : "交换语言"}
      disabled={source === "auto" && !detectedSource}
    >
      <ArrowLeftRight size={16} />
    </button>

    <div class="language-control">
      <label for="target-language">目标语言</label>
      <div class="select-wrapper">
        <select id="target-language" bind:value={target} aria-label="目标语言">
          {#each Object.entries(langs) as [code, name]}
            <option value={code}>{name}</option>
          {/each}
        </select>
      </div>
    </div>
  </div>

  <div class="right-tools">
    <div class="mode-group" role="group" aria-label="翻译模式">
      <button
        class="mode-btn"
        class:active={autoTranslate}
        on:click={() => (autoTranslate = !autoTranslate)}
        title={autoTranslate ? "关闭自动翻译" : "开启自动翻译"}
        aria-pressed={autoTranslate}
      >
        <Zap size={14} />
        <span class="mode-label">自动</span>
      </button>
      <button
        class="mode-btn"
        class:active={clipboardWatch}
        on:click={() => dispatch("toggleClipboard")}
        title={clipboardWatch ? "关闭剪贴板监听" : "开启剪贴板监听"}
        aria-pressed={clipboardWatch}
      >
        <Clipboard size={14} />
        <span class="mode-label">剪贴板</span>
      </button>
      <button
        class="mode-btn"
        class:active={compareMode}
        on:click={() => dispatch("toggleCompare")}
        title="多引擎对照"
        aria-pressed={compareMode}
      >
        <Columns2 size={14} />
        <span class="mode-label">对照</span>
      </button>
    </div>
    <button
      class="translate-btn"
      on:click={() => dispatch("translate")}
      disabled={translating || !canTranslate}
      aria-busy={translating}
    >
      <Send size={14} />
      <span>{translating ? "翻译中" : "翻译"}</span>
    </button>
  </div>
</header>

<style>
  .workspace-header {
    min-height: var(--header-h);
    padding: var(--sp-2) var(--sp-4);
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
    gap: var(--sp-4);
    background: var(--bg-panel);
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
  }
  .lang-bar {
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    padding: var(--sp-1);
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
  }
  .language-control {
    min-width: 104px;
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-md);
    transition: background var(--t-base) var(--ease-standard);
  }
  .language-control:hover {
    background: var(--bg-hover);
  }
  .language-control label {
    display: block;
    margin-left: var(--sp-2);
    color: var(--text-muted);
    font-size: var(--fs-xs);
    font-weight: var(--fw-medium);
    line-height: var(--lh-tight);
  }
  .select-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }
  .select-wrapper select {
    width: 100%;
    appearance: none;
    -webkit-appearance: none;
    background: transparent;
    border: none;
    color: var(--text-main);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    padding: var(--sp-1) var(--sp-6) 0 var(--sp-2);
    border-radius: var(--radius-sm);
    line-height: var(--lh-snug);
  }
  .select-wrapper select:focus-visible {
    outline: 2px solid var(--primary);
    outline-offset: var(--sp-1);
  }
  .select-wrapper::after {
    content: "";
    position: absolute;
    right: var(--sp-2);
    width: var(--sp-2);
    height: var(--sp-2);
    border-right: 1px solid var(--text-sec);
    border-bottom: 1px solid var(--text-sec);
    transform: rotate(45deg) translateY(-1px);
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
    width: var(--sp-8);
    height: var(--sp-8);
    background: var(--bg-surface);
    border: 1px solid var(--border);
    color: var(--text-sec);
    border-radius: var(--radius-full);
    cursor: pointer;
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard),
      border-color var(--t-base) var(--ease-standard),
      transform var(--t-slow) var(--ease-spring);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }
  .swap-btn:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--primary);
    border-color: var(--primary);
    transform: rotate(180deg);
  }
  .swap-btn:disabled {
    opacity: 0.38;
  }
  .right-tools {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
  }
  .mode-group {
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    padding: var(--sp-1);
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-full);
  }
  .mode-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    min-height: var(--sp-8);
    padding: var(--sp-1) var(--sp-3);
    border-radius: var(--radius-full);
    font-size: var(--fs-sm);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard);
  }
  .mode-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .mode-btn.active {
    color: var(--primary);
    background: var(--primary-soft);
    box-shadow: inset 0 0 0 1px var(--primary-soft);
  }
  .translate-btn {
    min-height: var(--sp-10);
    background: var(--accent-grad);
    color: var(--text-inverse);
    border: none;
    padding: var(--sp-2) var(--sp-5);
    border-radius: var(--radius-lg);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    box-shadow: var(--shadow-glow);
    transition: transform var(--t-fast) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard),
      filter var(--t-base) var(--ease-standard);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--sp-2);
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
    opacity: 0.48;
    box-shadow: none;
  }

  @media (max-width: 980px) {
    .mode-label {
      display: none;
    }
    .mode-btn {
      width: var(--sp-8);
      padding: var(--sp-1);
      justify-content: center;
    }
  }

  @media (max-width: 720px) {
    .workspace-header {
      flex-wrap: wrap;
      padding: var(--sp-2) var(--sp-3);
      gap: var(--sp-2);
    }
    .lang-bar {
      flex: 1;
      min-width: 0;
    }
    .language-control {
      min-width: 0;
      flex: 1;
    }
    .right-tools {
      width: 100%;
      justify-content: flex-end;
      gap: var(--sp-1);
    }
    .mode-group {
      margin-right: auto;
    }
    .translate-btn {
      min-height: var(--sp-8);
      padding: var(--sp-1) var(--sp-4);
    }
  }

  @media (max-width: 520px) {
    .workspace-header {
      padding: var(--sp-2);
    }
    .language-control {
      padding-inline: var(--sp-1);
    }
    .language-control label {
      margin-left: var(--sp-1);
    }
    .select-wrapper select {
      padding-left: var(--sp-1);
    }
  }
</style>
