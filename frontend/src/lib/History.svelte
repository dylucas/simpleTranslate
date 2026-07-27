<script lang="ts">
  import { AlertTriangle, ArrowRight, Download, History as HistoryIcon, Search, Trash2, X } from "@lucide/svelte";
  import { tick } from "svelte";
  import { langs } from "./languages";
  import { ARIA_SHORTCUTS } from "./shortcuts";
  import { MAX_INPUT_BYTES, truncateUtf8 } from "./textLimits";
  import type { HistoryEntry, HistoryPage, HistoryQuery } from "./types";

  interface Props {
    open: boolean;
    queryHistory: (query: HistoryQuery) => Promise<HistoryPage>;
    onClose: () => void;
    onClear: () => Promise<void>;
    onExport: () => Promise<boolean>;
    onError: (message: string) => void;
    onSelect: (entry: HistoryEntry) => void;
  }

  let { open, queryHistory, onClose, onClear, onExport, onError, onSelect }: Props = $props();
  const PAGE_SIZE = 10;
  const SEARCH_DELAY = 250;
  let entries = $state<HistoryEntry[]>([]);
  let total = $state(0);
  let allTotal = $state(0);
  let hasMore = $state(false);
  let loading = $state(false);
  let loadFailed = $state(false);
  let clearing = $state(false);
  let search = $state("");
  let confirming = $state(false);
  let drawer = $state<HTMLElement>();
  let searchInput = $state<HTMLInputElement>();
  let clearHistoryButton = $state<HTMLButtonElement>();
  let cancelClearButton = $state<HTMLButtonElement>();
  let wasOpen = false;
  let requestSequence = 0;
  let searchTimer: ReturnType<typeof setTimeout> | null = null;
  let observedSearch = "";
  let resultSummary = $derived(search.trim() ? `${total} 条匹配记录` : `${total} 条记录`);

  async function loadPage(reset: boolean): Promise<void> {
    if (!open || clearing || loading && !reset) return;
    const requestId = ++requestSequence;
    loading = true;
    try {
      const page = await queryHistory({ query: search.trim(), offset: reset ? 0 : entries.length, limit: PAGE_SIZE });
      if (requestId !== requestSequence || !open) return;
      entries = reset ? page.entries : [...entries, ...page.entries];
      total = page.total;
      allTotal = page.allTotal;
      hasMore = page.hasMore;
      loadFailed = false;
    } catch {
      if (requestId === requestSequence) {
        loadFailed = true;
        onError("历史记录加载失败");
      }
    } finally {
      if (requestId === requestSequence) loading = false;
    }
  }

  function releaseEntries(): void {
    requestSequence += 1;
    if (searchTimer) clearTimeout(searchTimer);
    searchTimer = null;
    observedSearch = "";
    entries = [];
    total = 0;
    allTotal = 0;
    hasMore = false;
    loading = false;
    loadFailed = false;
  }

  $effect(() => {
    if (open && !wasOpen) {
      wasOpen = true;
      void loadPage(true);
      void tick().then(() => searchInput?.focus());
    } else if (!open) {
      wasOpen = false;
      confirming = false;
      search = "";
      releaseEntries();
    }
  });

  $effect(() => {
    const query = search;
    if (!open || !wasOpen) return;
    if (query === observedSearch) return;
    observedSearch = query;
    if (searchTimer) clearTimeout(searchTimer);
    searchTimer = setTimeout(() => {
      searchTimer = null;
      if (query === search) void loadPage(true);
    }, SEARCH_DELAY);
  });

  $effect(() => {
    if (open && confirming) void tick().then(() => cancelClearButton?.focus());
  });

  function languageLabel(code: string): string {
    const normalized = code.trim().toLowerCase();
    if (!normalized) return "未知";
    if (normalized === "auto") return "自动识别";
    return langs[normalized] ?? normalized.toUpperCase();
  }

  function updateSearch(value: string): void {
    search = truncateUtf8(value, MAX_INPUT_BYTES);
  }

  function close(): void {
    if (clearing) return;
    confirming = false;
    onClose();
  }

  function cancelClear(): void {
    if (clearing) return;
    confirming = false;
    void tick().then(() => clearHistoryButton?.focus());
  }

  async function confirmClear(): Promise<void> {
    if (clearing) return;
    clearing = true;
    requestSequence += 1;
    loading = false;
    let failed = false;
    try {
      await onClear();
      entries = [];
      total = 0;
      allTotal = 0;
      hasMore = false;
      loadFailed = false;
      confirming = false;
      if (open) void tick().then(() => searchInput?.focus());
    } catch {
      failed = true;
      onError("历史记录清空失败");
    } finally {
      clearing = false;
      if (failed && open) void loadPage(true);
    }
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (clearing) {
      event.preventDefault();
      event.stopPropagation();
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      if (confirming) cancelClear();
      else close();
      return;
    }
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

  async function exportHistory(): Promise<void> {
    if (!allTotal) return;
    try {
      await onExport();
    } catch {
      onError("历史记录导出失败");
    }
  }

  function handleScroll(event: Event): void {
    const element = event.currentTarget as HTMLElement;
    if (hasMore && !loading && element.scrollTop + element.clientHeight >= element.scrollHeight - 40) void loadPage(false);
  }
</script>

{#if open}
  <button class="backdrop" onclick={close} disabled={clearing} aria-label="关闭历史记录"></button>
  <div class="drawer" bind:this={drawer} role="dialog" aria-modal="true" aria-labelledby="history-title" tabindex="-1" onkeydown={handleKeydown}>
    <header>
      <div class="title"><span><HistoryIcon size={18} /></span><div><h2 id="history-title">历史记录</h2><small aria-live="polite" aria-atomic="true">{resultSummary}</small></div></div>
      <div class="header-actions">
        <button onclick={() => void exportHistory()} disabled={!allTotal || clearing} aria-label="导出历史记录" title="导出 JSON"><Download size={16} /></button>
        <button bind:this={clearHistoryButton} class="danger" onclick={() => (confirming = true)} disabled={(!allTotal && !loadFailed) || clearing} aria-label="清空历史记录" title="清空历史"><Trash2 size={16} /></button>
        <button onclick={close} disabled={clearing} aria-label="关闭历史记录" aria-keyshortcuts={ARIA_SHORTCUTS.closePanel} title="关闭"><X size={18} /></button>
      </div>
    </header>

    <div class="search">
      <Search size={15} />
      <input bind:this={searchInput} value={search} oninput={(event) => updateSearch(event.currentTarget.value)} disabled={clearing} placeholder="搜索原文或译文" aria-label="搜索翻译记录" autocomplete="off" />
      {#if search}<button onclick={() => { search = ""; searchInput?.focus(); }} aria-label="清除搜索" title="清除搜索"><X size={13} /></button>{/if}
    </div>

    {#if confirming}
      <div class="confirm" role="alert">
        <div class="confirm-message"><AlertTriangle size={15} /><span>确定清空全部历史记录？</span></div>
        <div class="confirm-actions">
          <button bind:this={cancelClearButton} onclick={cancelClear} disabled={clearing}>取消</button>
          <button class="danger-solid" onclick={() => void confirmClear()} disabled={clearing}>{clearing ? "清空中…" : "确认清空"}</button>
        </div>
      </div>
    {/if}

    <div class="history-list" onscroll={handleScroll}>
      {#each entries as item (item)}
        <button class="history-item" onclick={() => onSelect(item)} disabled={clearing}>
          <span class="meta"><span class="route"><span>{languageLabel(item.source)}</span><ArrowRight size={11} /><span>{languageLabel(item.target)}</span></span><time>{item.time}</time></span>
          <span class="entry-content"><strong>{item.input}</strong><span class="output">{item.output}</span></span>
        </button>
      {:else}
        <div class="empty" role="status">
          <span class="empty-icon">{#if search}<Search size={22} strokeWidth={1.4} />{:else}<HistoryIcon size={24} strokeWidth={1.3} />{/if}</span>
          <strong>{loading ? "正在加载" : search ? "没有匹配记录" : "暂无翻译记录"}</strong>
          {#if search}<button class="empty-action" onclick={() => { search = ""; searchInput?.focus(); }}>清除搜索</button>{/if}
        </div>
      {/each}
      {#if hasMore}<button class="load-more" onclick={() => void loadPage(false)} disabled={loading || clearing}>{loading ? "加载中…" : "加载更多"}</button>{/if}
    </div>
  </div>
{/if}

<style>
  .backdrop { position: fixed; inset: 0; z-index: var(--z-drawer); border: 0; background: var(--bg-overlay); cursor: default; }
  .drawer { position: fixed; z-index: calc(var(--z-drawer) + 1); top: 0; right: 0; bottom: 0; display: flex; width: min(var(--history-drawer-w, 440px), 100vw); flex-direction: column; outline: 0; border-left: 1px solid var(--border); background: var(--bg-elevated); box-shadow: var(--shadow-xl); }
  header { display: flex; min-height: var(--header-h); flex: 0 0 auto; align-items: center; justify-content: space-between; gap: var(--sp-3); border-bottom: 1px solid var(--border); padding: var(--sp-3) var(--sp-4); }
  .title { display: flex; min-width: 0; align-items: center; gap: var(--sp-2); }
  .title > span { display: grid; width: 30px; height: 30px; place-items: center; border-radius: var(--radius-md); background: var(--primary-soft); color: var(--primary); }
  .title > div { min-width: 0; }
  h2 { margin: 0; color: var(--text-main); font-size: var(--fs-md); line-height: var(--lh-tight); }
  small { display: block; margin-top: 2px; color: var(--text-muted); font-size: var(--fs-xs); font-variant-numeric: tabular-nums; line-height: var(--lh-tight); white-space: nowrap; }
  .header-actions { display: flex; flex: 0 0 auto; align-items: center; gap: var(--sp-1); }
  button { border: 0; background: transparent; color: var(--text-sec); cursor: pointer; font-family: inherit; transition: border-color var(--t-fast) var(--ease-standard), background var(--t-fast) var(--ease-standard), color var(--t-fast) var(--ease-standard); }
  .header-actions button, .search button { display: grid; width: 32px; height: 32px; place-items: center; border: 1px solid transparent; border-radius: var(--radius-md); }
  button:not(.backdrop):hover:not(:disabled) { border-color: var(--border); background: var(--bg-hover); color: var(--text-main); }
  button:not(.backdrop):active:not(:disabled) { background: var(--primary-soft); }
  button:disabled { cursor: not-allowed; opacity: .35; }
  button.danger:hover { color: var(--danger); }
  .search { position: relative; display: flex; flex: 0 0 auto; align-items: center; margin: var(--sp-3); color: var(--text-muted); }
  .search :global(svg) { position: absolute; left: var(--sp-3); }
  input { width: 100%; min-height: 36px; border: 1px solid var(--border); border-radius: var(--radius-md); outline: 0; padding: 0 38px; background: var(--bg-input); color: var(--text-main); font: inherit; }
  input:focus { border-color: var(--primary); box-shadow: 0 0 0 2px var(--primary-soft); }
  .search button { position: absolute; right: var(--sp-1); }
  .confirm { display: flex; flex: 0 0 auto; align-items: center; gap: var(--sp-2); margin: 0 var(--sp-3) var(--sp-3); border: 1px solid var(--warning-border); border-radius: var(--radius-md); padding: var(--sp-2); background: var(--warning-soft); font-size: var(--fs-xs); }
  .confirm-message { display: flex; min-width: 0; flex: 1; align-items: center; gap: 6px; color: var(--text-main); }
  .confirm-message :global(svg) { flex: 0 0 auto; color: var(--warning); }
  .confirm-actions { display: flex; flex: 0 0 auto; align-items: center; gap: var(--sp-1); }
  .confirm button { min-height: 28px; border-radius: var(--radius-sm); padding: 0 var(--sp-2); }
  .confirm .danger-solid { border-color: color-mix(in srgb, var(--danger) 78%, transparent); background: var(--danger); color: var(--text-inverse); }
  .confirm .danger-solid:hover { border-color: var(--danger); background: var(--danger-strong); color: var(--text-inverse); }
  .history-list { display: flex; min-height: 0; flex: 1; flex-direction: column; gap: var(--sp-2); overflow: auto; padding: 0 var(--sp-3) var(--sp-3); }
  .history-item { display: grid; flex: 0 0 auto; gap: var(--sp-2); border: 1px solid var(--border-soft); border-radius: var(--radius-md); padding: var(--sp-3); background: var(--bg-surface); text-align: left; }
  .history-item:hover { border-color: var(--border-strong); background: var(--bg-hover); }
  .history-item:active { border-color: color-mix(in srgb, var(--primary) 45%, var(--border)); background: var(--primary-soft); }
  .history-item:focus-visible { outline-offset: -2px; }
  .meta { display: flex; min-width: 0; align-items: center; justify-content: space-between; gap: var(--sp-2); color: var(--text-muted); font-size: var(--fs-xs); line-height: var(--lh-tight); }
  .route { display: flex; min-width: 0; align-items: center; gap: var(--sp-1); overflow: hidden; color: var(--text-sec); font-weight: var(--fw-medium); text-overflow: ellipsis; white-space: nowrap; }
  .route :global(svg) { flex: 0 0 auto; color: var(--text-muted); }
  time { flex: 0 0 auto; color: var(--text-muted); font-variant-numeric: tabular-nums; white-space: nowrap; }
  .entry-content { display: grid; min-width: 0; gap: var(--sp-1); }
  .history-item strong, .output { display: -webkit-box; overflow: hidden; -webkit-box-orient: vertical; -webkit-line-clamp: 2; line-clamp: 2; overflow-wrap: anywhere; }
  .history-item strong { color: var(--text-main); font-size: var(--fs-base); font-weight: var(--fw-semibold); line-height: var(--lh-snug); }
  .output { color: var(--text-sec); font-size: var(--fs-sm); line-height: var(--lh-normal); }
  .empty { display: grid; min-height: 180px; flex: 1; place-content: center; place-items: center; gap: var(--sp-2); color: var(--text-muted); text-align: center; }
  .empty-icon { display: grid; width: 44px; height: 44px; margin-bottom: var(--sp-1); place-items: center; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--bg-surface); color: var(--text-sec); }
  .empty strong { color: var(--text-sec); font-size: var(--fs-sm); font-weight: var(--fw-medium); }
  .empty-action { min-height: 30px; margin-top: var(--sp-1); border: 1px solid var(--border); border-radius: var(--radius-sm); padding: 0 var(--sp-3); color: var(--primary); }
  .load-more { min-height: 34px; flex: 0 0 auto; border: 1px solid var(--border); border-radius: var(--radius-md); color: var(--primary); }
  @media (min-width: 1180px) {
    .backdrop { right: var(--history-drawer-w, 440px); background: transparent; }
    .drawer { box-shadow: -8px 0 24px rgba(0, 0, 0, .18); }
  }
  @media (max-width: 460px) {
    .drawer { width: 100vw; border-left: 0; }
    header { gap: var(--sp-2); padding-inline: var(--sp-3); }
    .title > span { width: 28px; height: 28px; }
    .header-actions button { width: 30px; height: 30px; }
    .confirm { align-items: flex-start; flex-wrap: wrap; }
    .confirm-message { flex-basis: 100%; }
    .confirm-actions { margin-left: auto; }
    .history-item { padding: 10px; }
  }
</style>
