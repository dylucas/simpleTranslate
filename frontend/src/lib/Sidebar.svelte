<script lang="ts">
  import { History, Languages, Moon, PanelLeftClose, Settings, Sun } from "@lucide/svelte";
  import type { EngineId } from "./types";

  interface Props {
    collapsed: boolean;
    activeEngine: EngineId;
    isDark: boolean;
    onToggle: () => void;
    onEngine: (engine: EngineId) => void;
    onTheme: () => void;
    onSettings: () => void;
    onHistory: () => void;
  }

  let { collapsed, activeEngine, isDark, onToggle, onEngine, onTheme, onSettings, onHistory }: Props = $props();
</script>

<aside class:collapsed aria-label="应用侧边栏">
  <header>
    <div class="brand" aria-label="SimpleTranslate">
      <span class="brand-mark"><Languages size={20} /></span>
      {#if !collapsed}<span><strong>SimpleTranslate</strong><small>翻译工作台</small></span>{/if}
    </div>
    <button class="icon-btn collapse" onclick={onToggle} aria-label={collapsed ? "展开侧边栏" : "收起侧边栏"} title={collapsed ? "展开" : "收起"}>
      <PanelLeftClose size={17} class={collapsed ? "rotated" : ""} />
    </button>
  </header>

  <nav aria-label="工作区">
    {#if !collapsed}<span class="section-label">工作区</span>{/if}
    <button class="nav-item" onclick={onHistory} aria-label="打开历史记录" title="历史记录">
      <History size={18} />{#if !collapsed}<span>历史记录</span>{/if}
    </button>
  </nav>

  <footer>
    {#if !collapsed}
      <div class="engine-control">
        <span class="section-label">默认引擎</span>
        <div class="segments" role="group" aria-label="默认翻译引擎">
          <button class:active={activeEngine === "tencent"} aria-pressed={activeEngine === "tencent"} onclick={() => onEngine("tencent")}>混元</button>
          <button class:active={activeEngine === "aliyun"} aria-pressed={activeEngine === "aliyun"} onclick={() => onEngine("aliyun")}>阿里云</button>
        </div>
      </div>
    {/if}
    <div class:vertical={collapsed} class="tools">
      <button class="icon-btn" onclick={onTheme} aria-label="切换深浅主题" title="切换主题">
        {#if isDark}<Sun size={18} />{:else}<Moon size={18} />{/if}
      </button>
      <button class="icon-btn" onclick={onSettings} aria-label="打开设置" title="设置"><Settings size={18} /></button>
    </div>
  </footer>
</aside>

<style>
  aside {
    display: flex;
    width: var(--sidebar-w);
    min-width: 0;
    flex: 0 0 auto;
    flex-direction: column;
    border-right: 1px solid var(--border-soft);
    background: var(--bg-sidebar);
    transition: width var(--t-slow) var(--ease-standard);
  }
  aside.collapsed { width: var(--sidebar-collapsed-w); }
  header { display: flex; min-height: var(--header-h); align-items: center; justify-content: space-between; gap: var(--sp-2); padding: 0 var(--sp-3); border-bottom: 1px solid var(--border-soft); }
  .brand { display: flex; min-width: 0; align-items: center; gap: var(--sp-2); overflow: hidden; color: var(--text-main); }
  .brand-mark { display: grid; width: 34px; height: 34px; flex: 0 0 auto; place-items: center; border: 1px solid color-mix(in srgb, var(--primary) 72%, white); border-radius: var(--radius-md); background: var(--primary); color: white; box-shadow: var(--shadow-sm); }
  .brand > span:last-child { display: flex; min-width: 0; flex-direction: column; }
  strong { overflow: hidden; max-width: 124px; font-size: var(--fs-md); font-weight: var(--fw-bold); text-overflow: ellipsis; white-space: nowrap; }
  small { color: var(--text-muted); font-size: var(--fs-xs); }
  nav { display: flex; flex: 1; flex-direction: column; gap: var(--sp-2); padding: var(--sp-4) var(--sp-3); }
  .section-label { color: var(--text-muted); font-size: var(--fs-xs); font-weight: var(--fw-semibold); }
  button { font-family: inherit; }
  .nav-item, .icon-btn { display: flex; align-items: center; justify-content: center; border: 1px solid transparent; background: transparent; color: var(--text-sec); cursor: pointer; }
  .nav-item { position: relative; justify-content: flex-start; gap: var(--sp-3); min-height: 40px; border-radius: var(--radius-md); padding: 0 var(--sp-3); font-size: var(--fs-base); }
  .nav-item:hover, .icon-btn:hover { border-color: var(--border); background: var(--bg-hover); color: var(--text-main); }
  .nav-item:hover::before { position: absolute; top: 10px; bottom: 10px; left: -1px; width: 2px; border-radius: var(--radius-full); background: var(--primary); content: ""; }
  footer { display: flex; flex-direction: column; gap: var(--sp-3); border-top: 1px solid var(--border-soft); padding: var(--sp-3); }
  .engine-control { display: grid; gap: var(--sp-2); }
  .segments { display: grid; grid-template-columns: 1fr 1fr; gap: 2px; border: 1px solid var(--border); border-radius: var(--radius-md); padding: 2px; background: var(--bg-input); }
  .segments button { min-height: 31px; border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text-sec); cursor: pointer; font-size: var(--fs-sm); }
  .segments button.active { background: var(--bg-elevated); color: var(--primary); box-shadow: var(--shadow-sm); font-weight: var(--fw-semibold); }
  .tools { display: flex; justify-content: space-between; }
  .tools.vertical { flex-direction: column; gap: var(--sp-2); }
  .icon-btn { width: 34px; height: 34px; border-radius: var(--radius-md); padding: 0; }
  .icon-btn :global(.rotated) { transform: rotate(180deg); }
  .collapsed header { justify-content: center; padding: 0; }
  .collapsed .brand { display: none; }
  .collapsed nav { align-items: center; padding-inline: var(--sp-2); }
  .collapsed .nav-item { width: 40px; justify-content: center; padding: 0; }
  @media (max-width: 720px) { aside { width: var(--sidebar-collapsed-w); } aside header { justify-content: center; } aside:not(.collapsed) .brand { display: flex; } aside:not(.collapsed) .brand > span:last-child, .collapse, .section-label, .nav-item span, .engine-control { display: none; } nav { align-items: center; padding-inline: var(--sp-2); } .nav-item { width: 40px; justify-content: center; padding: 0; } .tools { flex-direction: column; gap: var(--sp-2); } }
</style>
