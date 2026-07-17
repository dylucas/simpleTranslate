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

<aside class="sidebar" class:collapsed={sidebarCollapsed}>
  <div class="sidebar-header">
    {#if !sidebarCollapsed}
      <div class="brand" transition:fade={{ duration: 150 }}>
        <div class="brand-icon"><Languages size={22} /></div>
        <span>Translate</span>
      </div>
    {/if}
    <button
      class="collapse-toggle"
      class:centered={sidebarCollapsed}
      on:click={() => dispatch("toggleSidebar")}
      title={sidebarCollapsed ? "展开" : "收起"}
    >
      <div class="icon-wrapper" class:rotated={sidebarCollapsed}>
        <PanelLeftClose size={18} />
      </div>
    </button>
  </div>

  <nav class="side-nav">
    <button class="nav-item" on:click={() => dispatch("openHistory")} title="历史记录">
      <div class="nav-icon"><HistoryIcon size={20} /></div>
      {#if !sidebarCollapsed}
        <span class="nav-text" transition:fade={{ duration: 100 }}>历史记录</span>
      {/if}
    </button>
  </nav>

  <div class="sidebar-footer">
    {#if !sidebarCollapsed}
      <div class="engine-box" transition:fade={{ duration: 100 }}>
        <!-- svelte-ignore a11y-label-has-associated-control -->
        <label>翻译引擎</label>
        <div class="engine-pills">
          <button
            class:active={activeEngine === "tencent"}
            on:click={() => dispatch("selectEngine", "tencent")}>混元</button
          >
          <button
            class:active={activeEngine === "aliyun"}
            on:click={() => dispatch("selectEngine", "aliyun")}>阿里</button
          >
        </div>
      </div>
    {/if}

    <div class="bottom-tools" class:column-layout={sidebarCollapsed}>
      <button class="tool-btn" on:click={() => dispatch("toggleTheme")} title="切换主题">
        {#if isDark}<Sun size={18} />{:else}<Moon size={18} />{/if}
      </button>
      <button class="tool-btn" on:click={() => dispatch("openSettings")} title="设置">
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
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }
  .sidebar.collapsed {
    width: var(--sidebar-collapsed-w);
  }
  .sidebar-header {
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 16px;
    position: relative;
  }
  .brand {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    font-weight: var(--fw-bold);
    font-size: var(--fs-lg);
    letter-spacing: var(--tracking-tight);
    color: var(--text-main);
    white-space: nowrap;
    overflow: hidden;
  }
  .brand-icon {
    width: 34px;
    height: 34px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--accent-grad);
    color: var(--text-inverse);
    box-shadow: var(--shadow-glow);
    flex-shrink: 0;
  }
  .collapse-toggle {
    background: transparent;
    border: none;
    color: var(--text-sec);
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
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
    padding: var(--sp-4) var(--sp-3);
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
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
    height: 44px;
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
    min-width: 24px;
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
    gap: var(--sp-4);
    padding: var(--sp-4) var(--sp-3);
    transition: padding var(--t-slow) var(--ease-standard);
  }
  .engine-box label {
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    color: var(--text-sec);
    margin-bottom: var(--sp-2);
    display: block;
    text-transform: uppercase;
    letter-spacing: var(--tracking-wider);
  }
  .engine-pills {
    display: flex;
    background: var(--bg-hover);
    border-radius: var(--radius-sm);
    padding: 3px;
  }
  .engine-pills button {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--text-sec);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-xs);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
  }
  .engine-pills button:hover {
    color: var(--text-main);
  }
  .engine-pills button.active {
    background: var(--bg-surface);
    color: var(--text-main);
    box-shadow: var(--shadow-sm);
    font-weight: var(--fw-semibold);
  }
  .bottom-tools {
    display: flex;
    gap: var(--sp-2);
    justify-content: space-between;
    transition: all var(--t-slow) var(--ease-standard);
  }
  .engine-box {
    overflow: hidden;
    transition:
      max-height var(--t-slow) var(--ease-standard),
      opacity var(--t-base) var(--ease-standard);
  }
  .sidebar.collapsed .sidebar-footer {
    padding: var(--sp-4) 0;
  }
  .sidebar.collapsed .bottom-tools {
    flex-direction: column;
    align-items: center;
    gap: var(--sp-3);
  }
  .tool-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-sec);
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
    width: 40px;
    height: 40px;
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
</style>
