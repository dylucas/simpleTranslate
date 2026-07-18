<script lang="ts">
  import { Languages, Sun, Moon, Settings, PanelLeftClose, History as HistoryIcon } from "lucide-svelte";
  import { fade } from "svelte/transition";
  import { createEventDispatcher } from "svelte";

  export let sidebarCollapsed = false;
  export let activeEngine = "tencent";
  export let isDark = true;

  const dispatch = createEventDispatcher<{
    toggleSidebar: void;
    selectEngine: string;
    toggleTheme: void;
    openSettings: void;
    openHistory: void;
  }>();
</script>

<aside class="sidebar" class:collapsed={sidebarCollapsed} aria-label="应用侧边栏">
  <div class="sidebar-header">
    {#if !sidebarCollapsed}
      <div class="brand" transition:fade={{ duration: 150 }}>
        <div class="brand-icon"><Languages size={22} /></div>
        <div class="brand-copy">
          <span>SimpleTranslate</span>
          <small>智能翻译工作台</small>
        </div>
      </div>
    {/if}
    <button
      class="collapse-toggle"
      class:centered={sidebarCollapsed}
      on:click={() => dispatch("toggleSidebar")}
      title={sidebarCollapsed ? "展开" : "收起"}
      aria-label={sidebarCollapsed ? "展开侧边栏" : "收起侧边栏"}
    >
      <div class="icon-wrapper" class:rotated={sidebarCollapsed}>
        <PanelLeftClose size={18} />
      </div>
    </button>
  </div>

  <nav class="side-nav">
    {#if !sidebarCollapsed}
      <span class="nav-label" transition:fade={{ duration: 100 }}>工作区</span>
    {/if}
    <button class="nav-item" on:click={() => dispatch("openHistory")} title="历史记录" aria-label="打开历史记录">
      <div class="nav-icon"><HistoryIcon size={20} /></div>
      {#if !sidebarCollapsed}
        <span class="nav-text" transition:fade={{ duration: 100 }}>历史记录</span>
      {/if}
    </button>
  </nav>

  <div class="sidebar-footer">
    {#if !sidebarCollapsed}
      <div class="engine-box" transition:fade={{ duration: 100 }}>
        <span class="engine-label" id="engine-label">默认引擎</span>
        <div class="engine-pills" role="group" aria-labelledby="engine-label">
          <button
            class:active={activeEngine === "tencent"}
            aria-pressed={activeEngine === "tencent"}
            on:click={() => dispatch("selectEngine", "tencent")}>混元</button
          >
          <button
            class:active={activeEngine === "aliyun"}
            aria-pressed={activeEngine === "aliyun"}
            on:click={() => dispatch("selectEngine", "aliyun")}>阿里</button
          >
        </div>
      </div>
    {/if}

    <div class="bottom-tools" class:column-layout={sidebarCollapsed}>
      <button class="tool-btn" on:click={() => dispatch("toggleTheme")} title="切换主题" aria-label="切换深浅主题">
        {#if isDark}<Sun size={18} />{:else}<Moon size={18} />{/if}
      </button>
      <button class="tool-btn" on:click={() => dispatch("openSettings")} title="设置" aria-label="打开设置">
        <Settings size={18} />
      </button>
    </div>
  </div>
</aside>

<style>
  .sidebar {
    width: var(--sidebar-w);
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    transition: width var(--t-slow) var(--ease-standard);
    z-index: var(--z-sidebar);
    flex-shrink: 0;
    backdrop-filter: blur(18px);
    -webkit-backdrop-filter: blur(18px);
    overflow: hidden;
  }
  .sidebar.collapsed {
    width: var(--sidebar-collapsed-w);
  }
  .sidebar-header {
    min-height: var(--header-h);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--sp-2);
    padding: 0 var(--sp-3);
    position: relative;
    border-bottom: 1px solid var(--border-soft);
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
    color: var(--text-main);
    white-space: nowrap;
    overflow: hidden;
    min-width: 0;
  }
  .brand-icon {
    width: var(--sp-10);
    height: var(--sp-10);
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--accent-grad);
    color: var(--text-inverse);
    box-shadow: var(--shadow-glow);
    flex-shrink: 0;
  }
  .brand-copy {
    display: flex;
    min-width: 0;
    flex-direction: column;
    line-height: var(--lh-tight);
  }
  .brand-copy span {
    max-width: 122px;
    overflow: hidden;
    color: var(--text-main);
    font-size: var(--fs-md);
    font-weight: var(--fw-bold);
    letter-spacing: var(--tracking-tight);
    text-overflow: ellipsis;
  }
  .brand-copy small {
    margin-top: var(--sp-1);
    color: var(--text-muted);
    font-size: var(--fs-xs);
    font-weight: var(--fw-medium);
  }
  .collapse-toggle {
    background: transparent;
    border: none;
    color: var(--text-sec);
    width: var(--sp-8);
    height: var(--sp-8);
    padding: 0;
    border-radius: var(--radius-md);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
  }
  .collapse-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .collapse-toggle.centered {
    margin: 0 auto;
    width: 100%;
  }
  .icon-wrapper {
    display: flex;
    transition: transform var(--t-slow) var(--ease-spring);
  }
  .icon-wrapper.rotated {
    transform: rotate(180deg);
  }
  .side-nav {
    flex: 1;
    padding: var(--sp-5) var(--sp-3);
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }
  .nav-label {
    padding: 0 var(--sp-3);
    color: var(--text-muted);
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    letter-spacing: var(--tracking-widest);
    text-transform: uppercase;
  }
  .nav-item {
    display: flex;
    align-items: center;
    padding: var(--sp-2) var(--sp-3);
    background: transparent;
    border: none;
    border-radius: var(--radius-md);
    color: var(--text-sec);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
    min-height: var(--sp-10);
    width: 100%;
    text-align: left;
  }
  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .nav-item:active {
    background: var(--primary-soft);
    color: var(--primary);
  }
  .nav-icon {
    display: flex;
    justify-content: center;
    align-items: center;
    min-width: var(--sp-6);
  }
  .nav-text {
    margin-left: var(--sp-3);
    font-size: var(--fs-md);
    font-weight: var(--fw-medium);
    white-space: nowrap;
  }
  .sidebar-footer {
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: var(--sp-3);
    padding: var(--sp-3);
    transition: padding var(--t-slow) var(--ease-standard);
  }
  .engine-box {
    overflow: hidden;
    padding: var(--sp-3);
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    transition:
      max-height var(--t-slow) var(--ease-standard),
      opacity var(--t-base) var(--ease-standard);
  }
  .engine-label {
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    color: var(--text-muted);
    margin-bottom: var(--sp-2);
    display: block;
    text-transform: uppercase;
    letter-spacing: var(--tracking-wider);
  }
  .engine-pills {
    display: flex;
    gap: var(--sp-1);
    background: var(--bg-sidebar);
    border-radius: var(--radius-md);
    padding: var(--sp-1);
  }
  .engine-pills button {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--text-sec);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
  }
  .engine-pills button:hover {
    color: var(--text-main);
  }
  .engine-pills button.active {
    background: var(--primary-soft);
    color: var(--primary);
    box-shadow: var(--shadow-sm);
    font-weight: var(--fw-semibold);
  }
  .bottom-tools {
    display: flex;
    gap: var(--sp-2);
    justify-content: space-between;
    transition: all var(--t-slow) var(--ease-standard);
  }
  .sidebar.collapsed .sidebar-footer {
    padding: var(--sp-4) 0;
  }
  .sidebar.collapsed .bottom-tools {
    flex-direction: column;
    align-items: center;
    gap: var(--sp-3);
  }
  .sidebar.collapsed .side-nav {
    padding-inline: var(--sp-3);
  }
  .sidebar.collapsed .nav-item {
    justify-content: center;
    padding-inline: var(--sp-2);
  }
  .tool-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-sec);
    min-height: var(--sp-10);
    padding: var(--sp-2);
    border-radius: var(--radius-md);
    cursor: pointer;
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    transition: all var(--t-base) var(--ease-standard);
    min-width: 0;
  }
  .sidebar.collapsed .tool-btn {
    width: var(--sp-10);
    height: var(--sp-10);
    flex: none;
  }
  .tool-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
    border-color: var(--border);
  }
  .tool-btn:active {
    transform: scale(0.94);
  }

  @media (max-width: 900px) {
    .sidebar:not(.collapsed) {
      width: 184px;
    }
    .brand-copy span {
      max-width: 98px;
    }
  }

  @media (max-width: 720px) {
    .sidebar,
    .sidebar:not(.collapsed) {
      width: var(--sidebar-collapsed-w);
    }
    .sidebar:not(.collapsed) .brand,
    .sidebar:not(.collapsed) .nav-label,
    .sidebar:not(.collapsed) .nav-text,
    .sidebar:not(.collapsed) .engine-box {
      display: none;
    }
    .sidebar-header {
      justify-content: center;
      padding-inline: var(--sp-3);
    }
    .collapse-toggle {
      width: 100%;
    }
    .side-nav {
      padding-inline: var(--sp-3);
    }
    .nav-item {
      justify-content: center;
      padding-inline: var(--sp-2);
    }
    .sidebar-footer {
      padding: var(--sp-4) 0;
    }
    .bottom-tools {
      flex-direction: column;
      align-items: center;
      gap: var(--sp-3);
    }
    .tool-btn {
      width: var(--sp-10);
      height: var(--sp-10);
      flex: none;
    }
  }
</style>
