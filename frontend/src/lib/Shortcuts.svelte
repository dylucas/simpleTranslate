<script>
  import { X, Keyboard } from "lucide-svelte";
  import { fade, fly } from "svelte/transition";

  export let show = false;

  // 快捷键分组定义（与 App.svelte 的 handleGlobalKeydown 保持同步）
  const groups = [
    {
      title: "翻译操作",
      items: [
        { keys: ["Ctrl/Cmd", "Enter"], desc: "发送翻译请求" },
        { keys: ["Ctrl/Cmd", "J"], desc: "交换源语言与目标语言" },
      ],
    },
    {
      title: "输入控制",
      items: [
        { keys: ["Ctrl/Cmd", "L"], desc: "聚焦到输入框" },
        { keys: ["Ctrl/Cmd", "K"], desc: "清空输入内容" },
      ],
    },
    {
      title: "界面与面板",
      items: [
        { keys: ["Ctrl/Cmd", "Shift", "H"], desc: "打开 / 关闭历史记录" },
        { keys: ["Ctrl/Cmd", "M"], desc: "切换深色 / 浅色主题" },
        { keys: ["?"], desc: "打开本快捷键速查面板" },
        { keys: ["Esc"], desc: "关闭弹窗 / 面板" },
      ],
    },
  ];

  function handleClose() {
    show = false;
  }
</script>

{#if show}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div
    class="overlay"
    on:click={handleClose}
    transition:fade={{ duration: 200 }}
  ></div>
  <div
    class="modal"
    on:click|stopPropagation
    transition:fly={{ y: 20, duration: 250 }}
    role="dialog"
    aria-modal="true"
    aria-label="快捷键速查"
  >
    <header class="modal-header">
      <div class="header-title">
        <Keyboard size={18} />
        <h3>快捷键速查</h3>
      </div>
      <button class="close-btn" on:click={handleClose} title="关闭">
        <X size={16} />
      </button>
    </header>
    <div class="modal-body">
      {#each groups as group}
        <div class="group">
          <div class="group-title">{group.title}</div>
          <div class="group-items">
            {#each group.items as item}
              <div class="shortcut-row">
                <span class="shortcut-desc">{item.desc}</span>
                <span class="shortcut-keys">
                  {#each item.keys as k, i}
                    {#if i > 0}<span class="plus">+</span>{/if}
                    <kbd>{k}</kbd>
                  {/each}
                </span>
              </div>
            {/each}
          </div>
        </div>
      {/each}
      <div class="platform-hint">
        macOS 使用 <kbd>Cmd</kbd> (⌘)，其他平台使用 <kbd>Ctrl</kbd>
      </div>
    </div>
  </div>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: var(--bg-overlay);
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
    z-index: var(--z-modal);
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .modal {
    width: 480px;
    max-width: 92vw;
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-xl);
    overflow: hidden;
  }
  .modal-header {
    padding: var(--sp-5) var(--sp-6);
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
  }
  .header-title {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    color: var(--text-main);
  }
  .header-title h3 {
    margin: 0;
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
  }
  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    cursor: pointer;
    padding: var(--sp-1);
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    transition: all var(--t-base) var(--ease-standard);
  }
  .close-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }

  .modal-body {
    padding: var(--sp-4) var(--sp-6) var(--sp-5);
    max-height: 60vh;
    overflow-y: auto;
  }
  .group {
    margin-bottom: var(--sp-5);
  }
  .group:last-of-type {
    margin-bottom: var(--sp-3);
  }
  .group-title {
    font-size: var(--fs-xs);
    font-weight: var(--fw-bold);
    color: var(--primary);
    text-transform: uppercase;
    letter-spacing: var(--tracking-wide);
    margin-bottom: var(--sp-2);
    display: flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .group-items {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }
  .shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px var(--sp-3);
    border-radius: var(--radius-md);
    transition: background var(--t-fast) var(--ease-standard);
  }
  .shortcut-row:hover {
    background: var(--bg-hover);
  }
  .shortcut-desc {
    font-size: var(--fs-base);
    color: var(--text-main);
  }
  .shortcut-keys {
    display: flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .plus {
    color: var(--text-sec);
    font-size: var(--fs-xs);
  }
  kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 26px;
    height: 22px;
    padding: 0 var(--sp-1);
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-bottom-width: 2px;
    border-radius: var(--radius-xs);
    font-family: var(--font-mono);
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    color: var(--text-main);
    line-height: 1;
  }
  .platform-hint {
    margin-top: var(--sp-2);
    padding-top: var(--sp-3);
    border-top: 1px solid var(--border);
    font-size: var(--fs-sm);
    color: var(--text-sec);
    text-align: center;
  }
  .platform-hint kbd {
    min-width: auto;
    height: 18px;
    font-size: 10px;
    padding: 0 var(--sp-1);
  }
</style>
