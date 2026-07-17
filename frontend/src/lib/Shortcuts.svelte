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
    background: rgba(0, 0, 0, 0.5);
    backdrop-filter: blur(4px);
    z-index: 1500;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .modal {
    width: 480px;
    max-width: 92vw;
    background: var(--bg-surface, #1a1c23);
    border: 1px solid var(--border, #2a2d38);
    border-radius: 16px;
    box-shadow: 0 20px 50px rgba(0, 0, 0, 0.5);
    overflow: hidden;
  }
  .modal-header {
    padding: 18px 22px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border, #2a2d38);
  }
  .header-title {
    display: flex;
    align-items: center;
    gap: 10px;
    color: var(--text-main, #e2e8f0);
  }
  .header-title h3 {
    margin: 0;
    font-size: 15px;
    font-weight: 600;
  }
  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-sec, #94a3b8);
    cursor: pointer;
    padding: 6px;
    border-radius: 6px;
    display: flex;
    align-items: center;
  }
  .close-btn:hover {
    background: var(--bg-hover, #2a2d38);
    color: var(--text-main, #e2e8f0);
  }

  .modal-body {
    padding: 16px 22px 20px;
    max-height: 60vh;
    overflow-y: auto;
  }
  .group {
    margin-bottom: 18px;
  }
  .group:last-of-type {
    margin-bottom: 12px;
  }
  .group-title {
    font-size: 11px;
    font-weight: 700;
    color: var(--primary, #3b82f6);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
  }
  .group-items {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .shortcut-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 7px 10px;
    border-radius: 8px;
    transition: background 0.15s;
  }
  .shortcut-row:hover {
    background: var(--bg-hover, rgba(255, 255, 255, 0.04));
  }
  .shortcut-desc {
    font-size: 13px;
    color: var(--text-main, #e2e8f0);
  }
  .shortcut-keys {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .plus {
    color: var(--text-sec, #94a3b8);
    font-size: 11px;
  }
  kbd {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 26px;
    height: 22px;
    padding: 0 6px;
    background: var(--bg-input, #0c0d10);
    border: 1px solid var(--border, #2a2d38);
    border-bottom-width: 2px;
    border-radius: 5px;
    font-family: ui-monospace, "SF Mono", Menlo, monospace;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-main, #e2e8f0);
    line-height: 1;
  }
  .platform-hint {
    margin-top: 8px;
    padding-top: 14px;
    border-top: 1px solid var(--border, #2a2d38);
    font-size: 12px;
    color: var(--text-sec, #94a3b8);
    text-align: center;
  }
  .platform-hint kbd {
    min-width: auto;
    height: 18px;
    font-size: 10px;
    padding: 0 4px;
  }
</style>
