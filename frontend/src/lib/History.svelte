<script lang="ts">
  import { ArrowLeftRight, Download, History as HistoryIcon, Search, Trash2, X } from "@lucide/svelte";
  import { tick } from "svelte";
  import type { HistoryEntry } from "./types";

  interface Props {
    open: boolean;
    history: HistoryEntry[];
    onClose: () => void;
    onClear: () => void;
    onSelect: (entry: HistoryEntry) => void;
  }

  let { open, history, onClose, onClear, onSelect }: Props = $props();
  let search = $state("");
  let confirming = $state(false);
  let drawer = $state<HTMLElement>();
  let wasOpen = false;
  let filtered = $derived.by(() => {
    const query = search.trim().toLowerCase();
    return query
      ? history.filter((item) => item.input.toLowerCase().includes(query) || item.output.toLowerCase().includes(query))
      : history;
  });

  $effect(() => {
    if (open && !wasOpen) {
      wasOpen = true;
      void tick().then(() => drawer?.focus());
    } else if (!open) {
      wasOpen = false;
      confirming = false;
      search = "";
    }
  });

  function close(): void {
    confirming = false;
    onClose();
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (event.key === "Escape") close();
    if (event.key !== "Tab" || !drawer) return;
    const focusable = [...drawer.querySelectorAll<HTMLElement>("button:not(:disabled), input")];
    if (!focusable.length) return;
    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function exportHistory(): void {
    if (!history.length) return;
    const blob = new Blob([JSON.stringify(history, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `simpleTranslate-history-${new Date().toISOString().slice(0, 10)}.json`;
    anchor.click();
    URL.revokeObjectURL(url);
  }
</script>

{#if open}
  <button class="backdrop" onclick={close} aria-label="关闭历史记录"></button>
  <div class="drawer" bind:this={drawer} role="dialog" aria-modal="true" aria-labelledby="history-title" tabindex="-1" onkeydown={handleKeydown}>
    <header>
      <div class="title"><span><HistoryIcon size={18} /></span><div><h2 id="history-title">历史记录</h2><small>{history.length} 条记录</small></div></div>
      <div class="header-actions">
        <button onclick={exportHistory} disabled={!history.length} aria-label="导出历史记录" title="导出 JSON"><Download size={16} /></button>
        <button class="danger" onclick={() => (confirming = true)} disabled={!history.length} aria-label="清空历史记录" title="清空历史"><Trash2 size={16} /></button>
        <button onclick={close} aria-label="关闭历史记录"><X size={18} /></button>
      </div>
    </header>

    <div class="search">
      <Search size={15} />
      <input bind:value={search} placeholder="搜索原文或译文" aria-label="搜索翻译记录" />
      {#if search}<button onclick={() => (search = "")} aria-label="清除搜索"><X size={13} /></button>{/if}
    </div>

    {#if confirming}
      <div class="confirm" role="alert">
        <span>确定清空全部历史记录？</span>
        <button onclick={() => (confirming = false)}>取消</button>
        <button class="danger-solid" onclick={() => { onClear(); confirming = false; }}>确认清空</button>
      </div>
    {/if}

    <div class="history-list">
      {#each filtered as item (item.id)}
        <button class="history-item" onclick={() => onSelect(item)}>
          <span class="meta"><span>{item.source.toUpperCase()} <ArrowLeftRight size={10} /> {item.target.toUpperCase()}</span><time>{item.time}</time></span>
          <strong>{item.input}</strong>
          <span class="output">{item.output}</span>
        </button>
      {:else}
        <div class="empty"><HistoryIcon size={36} strokeWidth={1.2} /><span>{search ? "没有匹配记录" : "暂无翻译记录"}</span></div>
      {/each}
    </div>
  </div>
{/if}

<style>
  .backdrop { position: fixed; inset: 0; z-index: var(--z-drawer); border: 0; background: var(--bg-overlay); cursor: default; }
  .drawer { position: fixed; z-index: calc(var(--z-drawer) + 1); top: 0; right: 0; bottom: 0; display: flex; width: min(380px, 100vw); flex-direction: column; outline: 0; border-left: 1px solid var(--border); background: var(--bg-elevated); box-shadow: var(--shadow-xl); }
  header { display: flex; min-height: var(--header-h); align-items: center; justify-content: space-between; border-bottom: 1px solid var(--border); padding: var(--sp-3) var(--sp-4); }
  .title { display: flex; align-items: center; gap: var(--sp-3); }
  .title > span { display: grid; width: 32px; height: 32px; place-items: center; border-radius: var(--radius-md); background: var(--primary-soft); color: var(--primary); }
  h2 { margin: 0; color: var(--text-main); font-size: var(--fs-lg); }
  small { color: var(--text-muted); }
  .header-actions, .confirm { display: flex; align-items: center; gap: var(--sp-1); }
  button { border: 0; background: transparent; color: var(--text-sec); cursor: pointer; font-family: inherit; }
  .header-actions button, .search button { display: grid; width: 32px; height: 32px; place-items: center; border-radius: var(--radius-md); }
  button:hover:not(:disabled) { background: var(--bg-hover); color: var(--text-main); }
  button:disabled { cursor: not-allowed; opacity: .35; }
  button.danger:hover { color: var(--danger); }
  .search { position: relative; display: flex; align-items: center; margin: var(--sp-3) var(--sp-4); color: var(--text-muted); }
  .search :global(svg) { position: absolute; left: var(--sp-3); }
  input { width: 100%; min-height: 38px; border: 1px solid var(--border); border-radius: var(--radius-md); outline: 0; padding: 0 38px; background: var(--bg-input); color: var(--text-main); font: inherit; }
  input:focus { border-color: var(--primary); box-shadow: 0 0 0 3px var(--primary-soft); }
  .search button { position: absolute; right: var(--sp-1); }
  .confirm { margin: 0 var(--sp-4) var(--sp-3); border: 1px solid var(--warning-border); border-radius: var(--radius-md); padding: var(--sp-2); background: var(--warning-soft); font-size: var(--fs-sm); }
  .confirm span { flex: 1; color: var(--text-main); }
  .confirm button { min-height: 28px; border-radius: var(--radius-sm); padding: 0 var(--sp-2); }
  .confirm .danger-solid { background: var(--danger); color: white; }
  .history-list { display: flex; min-height: 0; flex: 1; flex-direction: column; gap: var(--sp-2); overflow: auto; padding: 0 var(--sp-4) var(--sp-4); }
  .history-item { display: grid; flex: 0 0 auto; gap: var(--sp-2); border: 1px solid var(--border-soft); border-radius: var(--radius-md); padding: var(--sp-3); background: var(--bg-surface); text-align: left; }
  .history-item:hover { border-color: var(--border-strong); }
  .meta { display: flex; align-items: center; justify-content: space-between; color: var(--text-muted); font-size: var(--fs-xs); }
  .meta > span { display: flex; align-items: center; gap: var(--sp-1); }
  .history-item strong, .output { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .history-item strong { color: var(--text-main); font-size: var(--fs-base); }
  .output { color: var(--text-sec); font-size: var(--fs-sm); }
  .empty { display: grid; flex: 1; place-content: center; place-items: center; gap: var(--sp-3); color: var(--text-muted); }
</style>
