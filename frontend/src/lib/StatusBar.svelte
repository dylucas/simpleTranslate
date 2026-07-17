<script lang="ts">
  export let status = "准备就绪";
</script>

<footer class="app-status-bar">
  <div class="status-item">
    <span
      class="status-dot"
      class:processing={status === "翻译中..."}
      class:done={status === "完成"}
      class:error={status === "翻译失败"}
    ></span>
    {status}
  </div>
  <div class="status-item shortcut-hint">
    <span>Ctrl+Enter 发送 · Ctrl+J 交换 · Ctrl+Shift+H 历史</span>
  </div>
</footer>

<style>
  .app-status-bar {
    height: var(--statusbar-h);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--sp-4);
    font-size: var(--fs-xs);
    color: var(--text-sec);
    background: var(--bg-sidebar);
  }
  .status-item {
    display: flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .shortcut-hint {
    color: var(--text-muted);
  }
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--success);
    box-shadow: 0 0 0 0 var(--success-soft);
  }
  .status-dot.done {
    background: var(--success);
  }
  .status-dot.processing {
    background: var(--warning);
    animation: pulse 1s infinite;
  }
  .status-dot.error {
    background: var(--danger);
  }
  @keyframes pulse {
    0% { opacity: 1; }
    50% { opacity: 0.4; }
    100% { opacity: 1; }
  }
  @media (max-width: 720px) {
    .shortcut-hint span {
      display: none;
    }
  }
</style>
