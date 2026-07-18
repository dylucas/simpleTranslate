<script lang="ts">
  import { History, Languages, Moon, Settings, Sun } from "@lucide/svelte";

  interface Props {
    activePanel: "config" | "history" | null;
    isDark: boolean;
    onTheme: () => void;
    onSettings: () => void;
    onHistory: () => void;
  }

  let { activePanel, isDark, onTheme, onSettings, onHistory }: Props = $props();
</script>

<aside aria-label="应用侧边栏">
  <div class="brand" aria-label="SimpleTranslate" data-tooltip="SimpleTranslate">
    <span class="brand-mark"><Languages size={20} strokeWidth={2} /></span>
    <span class="sr-only">SimpleTranslate</span>
  </div>

  <nav aria-label="工作区">
    <button
      class:active={activePanel === "history"}
      class="rail-action"
      onclick={onHistory}
      aria-label="打开历史记录"
      aria-pressed={activePanel === "history"}
      data-tooltip="历史记录"
    >
      <History size={18} />
    </button>
  </nav>

  <footer>
    <button class="rail-action" onclick={onTheme} aria-label="切换深浅主题" data-tooltip={isDark ? "切换到浅色" : "切换到深色"}>
      {#if isDark}<Sun size={18} />{:else}<Moon size={18} />{/if}
    </button>
    <button
      class:active={activePanel === "config"}
      class="rail-action"
      onclick={onSettings}
      aria-label="打开设置"
      aria-pressed={activePanel === "config"}
      data-tooltip="偏好设置"
    >
      <Settings size={18} />
    </button>
  </footer>
</aside>

<style>
  aside {
    position: relative;
    z-index: 3;
    display: flex;
    width: var(--rail-w);
    flex: 0 0 var(--rail-w);
    flex-direction: column;
    align-items: center;
    border-right: 1px solid var(--border-soft);
    background: var(--bg-rail);
  }
  .brand {
    display: grid;
    width: 100%;
    height: var(--commandbar-h);
    flex: 0 0 auto;
    place-items: center;
    border-bottom: 1px solid var(--border-soft);
  }
  .brand-mark {
    display: grid;
    width: 32px;
    height: 32px;
    place-items: center;
    border: 1px solid color-mix(in srgb, var(--primary) 68%, var(--border));
    border-radius: var(--radius-md);
    background: var(--primary);
    color: var(--text-inverse);
    box-shadow: var(--shadow-sm);
  }
  nav { display: flex; width: 100%; flex: 1; flex-direction: column; align-items: center; gap: var(--sp-2); padding: var(--sp-3) 0; }
  footer { display: flex; width: 100%; flex-direction: column; align-items: center; gap: var(--sp-2); border-top: 1px solid var(--border-soft); padding: var(--sp-3) 0; }
  .rail-action {
    position: relative;
    display: grid;
    width: 36px;
    height: 36px;
    place-items: center;
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
  }
  .rail-action:hover { border-color: var(--border); background: var(--bg-hover); color: var(--text-main); }
  .rail-action.active { border-color: color-mix(in srgb, var(--primary) 38%, var(--border)); background: var(--primary-soft); color: var(--primary); }
  .rail-action.active::before { position: absolute; top: 8px; bottom: 8px; left: -11px; width: 2px; border-radius: var(--radius-full); background: var(--primary); content: ""; }
  [data-tooltip]::after {
    position: absolute;
    z-index: 20;
    top: 50%;
    left: calc(100% + 10px);
    visibility: hidden;
    width: max-content;
    max-width: 140px;
    padding: 5px 8px;
    transform: translateY(-50%) translateX(-2px);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    background: var(--bg-elevated);
    box-shadow: var(--shadow-md);
    color: var(--text-main);
    content: attr(data-tooltip);
    font-size: var(--fs-xs);
    line-height: 1.3;
    opacity: 0;
    pointer-events: none;
    transition: opacity var(--t-fast) var(--ease-standard), transform var(--t-fast) var(--ease-standard), visibility var(--t-fast);
  }
  [data-tooltip]:hover::after, [data-tooltip]:focus-visible::after { visibility: visible; transform: translateY(-50%) translateX(0); opacity: 1; }
  .brand[data-tooltip] { position: relative; }
  .sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0, 0, 0, 0); white-space: nowrap; }
</style>
