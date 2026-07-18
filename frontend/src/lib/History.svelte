<script>
    import { ArrowLeftRight, History, Trash2, X, Search, Download } from "lucide-svelte";
    import { fade, fly } from "svelte/transition";
    import { createEventDispatcher } from "svelte";
    import { tick } from "svelte";

    export let show = false;
    export let history = [];

    const dispatch = createEventDispatcher();

    let searchTerm = "";
    let confirmingClear = false;
    let drawerEl;
    let wasShown = false;

    function handleItemClick(item) {
        dispatch("select", item);
        show = false;
    }

    function handleClear() {
        dispatch("clear");
        confirmingClear = false;
    }

    function handleClose() {
        show = false;
        confirmingClear = false;
        dispatch("close");
    }

    // 按输入或输出文本搜索（大小写不敏感）
    $: filteredHistory = (() => {
        const kw = searchTerm.trim().toLowerCase();
        if (!kw) return history;
        return history.filter(
            (item) =>
                (item.input || "").toLowerCase().includes(kw) ||
                (item.output || "").toLowerCase().includes(kw)
        );
    })();

    $: if (show && !wasShown) {
        wasShown = true;
        tick().then(() => drawerEl?.focus());
    }
    $: if (!show) {
        wasShown = false;
    }

    // 导出历史记录为 JSON 文件
    function handleExport() {
        if (!history || history.length === 0) return;
        const blob = new Blob([JSON.stringify(history, null, 2)], {
            type: "application/json",
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        const ts = new Date()
            .toISOString()
            .replace(/[:.]/g, "-")
            .slice(0, 19);
        a.download = `simpleTranslate-history-${ts}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    }
</script>

{#if show}
    <div
        class="backdrop"
        on:click={handleClose}
        aria-hidden="true"
        transition:fade={{ duration: 200 }}
    ></div>
    <aside
        class="drawer"
        bind:this={drawerEl}
        role="dialog"
        aria-modal="true"
        aria-labelledby="history-title"
        tabindex="-1"
        transition:fly={{ x: 360, duration: 300, opacity: 1 }}
    >
        <div class="drawer-header">
            <div class="drawer-title">
                <div class="title-icon"><History size={18} /></div>
                <div>
                    <h3 id="history-title">历史记录</h3>
                    <span>{history.length} 条记录</span>
                </div>
            </div>
            <div class="drawer-actions">
                <button
                    class="icon-btn"
                    on:click={handleExport}
                    disabled={!history || history.length === 0}
                    title="导出为 JSON"
                    aria-label="导出历史记录"
                ><Download size={16} /></button>
                <button
                    class="icon-btn danger"
                    on:click={() => (confirmingClear = true)}
                    disabled={!history || history.length === 0}
                    title="清空历史"
                    aria-label="清空历史记录"
                    ><Trash2 size={16} /></button
                >
                <button class="icon-btn" on:click={handleClose} aria-label="关闭历史记录"
                    ><X size={18} /></button
                >
            </div>
        </div>
        <div class="search-bar">
            <div class="search-input-wrapper">
                <Search size={14} />
                <input
                    type="text"
                    bind:value={searchTerm}
                    placeholder="搜索翻译记录..."
                    aria-label="搜索翻译记录"
                />
                {#if searchTerm}
                    <button class="search-clear" on:click={() => (searchTerm = "")} aria-label="清除搜索内容">
                        <X size={12} />
                    </button>
                {/if}
            </div>
            {#if searchTerm}
                <span class="search-count">{filteredHistory.length} 条匹配</span>
            {/if}
        </div>
        {#if confirmingClear}
            <div class="clear-confirm" role="alert">
                <span>确定清空全部历史记录？此操作不可撤销。</span>
                <div>
                    <button class="confirm-cancel" on:click={() => (confirmingClear = false)}>取消</button>
                    <button class="confirm-danger" on:click={handleClear}>确认清空</button>
                </div>
            </div>
        {/if}
        <div class="drawer-content">
            {#each filteredHistory as item}
                <button
                    class="history-item"
                    on:click={() => handleItemClick(item)}
                >
                    <div class="h-meta">
                        <span class="h-lang"
                            >{(item.source || "auto").toUpperCase()}
                            <ArrowLeftRight size={10} />
                            {(item.target || "zh").toUpperCase()}</span
                        >
                        <span class="h-time">{item.time}</span>
                    </div>
                    <span class="h-preview">{item.input}</span>
                    <span class="h-output">{item.output}</span>
                </button>
            {:else}
                <div class="empty-state">
                    <History size={40} strokeWidth={1} />
                    <p>{searchTerm ? "无匹配记录" : "暂无翻译记录"}</p>
                </div>
            {/each}
        </div>
    </aside>
{/if}

<style>
    /* 历史抽屉 */
    .backdrop {
        position: fixed;
        inset: 0;
        background: var(--bg-overlay);
        backdrop-filter: blur(4px);
        -webkit-backdrop-filter: blur(4px);
        z-index: var(--z-drawer);
    }
    .drawer {
        position: fixed;
        top: 0;
        right: 0;
        bottom: 0;
        width: min(360px, 100vw);
        background: var(--bg-elevated);
        z-index: calc(var(--z-drawer) + 1);
        border-left: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        box-shadow: var(--shadow-xl);
    }
    .drawer-header {
        min-height: var(--header-h);
        padding: var(--sp-3) var(--sp-4);
        border-bottom: 1px solid var(--border);
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-shrink: 0;
    }
    .drawer-title {
        display: flex;
        align-items: center;
        gap: var(--sp-3);
        min-width: 0;
    }
    .title-icon {
        width: var(--sp-8);
        height: var(--sp-8);
        display: inline-flex;
        align-items: center;
        justify-content: center;
        color: var(--primary);
        background: var(--primary-soft);
        border-radius: var(--radius-md);
    }
    .drawer-header h3 {
        margin: 0;
        font-size: var(--fs-lg);
        font-weight: var(--fw-semibold);
        line-height: var(--lh-tight);
    }
    .drawer-title span {
        display: block;
        margin-top: var(--sp-1);
        color: var(--text-muted);
        font-size: var(--fs-xs);
    }

    .drawer-actions {
        display: flex;
        gap: var(--sp-1);
    }
    .icon-btn {
        background: transparent;
        border: 1px solid transparent;
        color: var(--text-sec);
        width: var(--sp-8);
        height: var(--sp-8);
        padding: 0;
        border-radius: var(--radius-md);
        cursor: pointer;
        display: inline-flex;
        align-items: center;
        justify-content: center;
        transition: color var(--t-base) var(--ease-standard),
            background var(--t-base) var(--ease-standard),
            border-color var(--t-base) var(--ease-standard);
    }
    .icon-btn:hover {
        background: var(--bg-hover);
        color: var(--text-main);
        border-color: var(--border);
    }
    .icon-btn.danger:hover {
        color: var(--danger);
        background: var(--danger-soft);
    }
    .icon-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .search-bar {
        padding: var(--sp-3) var(--sp-4);
        border-bottom: 1px solid var(--border);
        display: flex;
        align-items: center;
        gap: var(--sp-2);
        flex-shrink: 0;
    }
    .search-input-wrapper {
        position: relative;
        display: flex;
        align-items: center;
        flex: 1;
    }
    .search-input-wrapper :global(svg) {
        position: absolute;
        left: var(--sp-3);
        color: var(--text-sec);
        pointer-events: none;
    }
    .search-input-wrapper input {
        width: 100%;
        background: var(--bg-input);
        border: 1px solid var(--border);
        border-radius: var(--radius-md);
        padding: var(--sp-2) var(--sp-8) var(--sp-2) var(--sp-8);
        color: var(--text-main);
        font-size: var(--fs-base);
        transition: border-color var(--t-base) var(--ease-standard),
            box-shadow var(--t-base) var(--ease-standard);
    }
    .search-input-wrapper input:focus {
        border-color: var(--primary);
        box-shadow: 0 0 0 3px var(--primary-soft);
    }
    .search-clear {
        position: absolute;
        right: var(--sp-2);
        background: transparent;
        border: none;
        color: var(--text-sec);
        cursor: pointer;
        width: var(--sp-6);
        height: var(--sp-6);
        padding: 0;
        border-radius: var(--radius-sm);
        display: flex;
        align-items: center;
        justify-content: center;
    }
    .search-clear:hover {
        color: var(--text-main);
        background: var(--bg-hover);
    }
    .search-count {
        font-size: var(--fs-xs);
        color: var(--text-sec);
        font-weight: 600;
        white-space: nowrap;
    }

    .clear-confirm {
        display: flex;
        flex-direction: column;
        gap: var(--sp-3);
        margin: var(--sp-3) var(--sp-4) 0;
        padding: var(--sp-3);
        color: var(--text-main);
        background: var(--danger-soft);
        border: 1px solid var(--danger);
        border-radius: var(--radius-lg);
        font-size: var(--fs-sm);
    }
    .clear-confirm > div {
        display: flex;
        justify-content: flex-end;
        gap: var(--sp-2);
    }
    .confirm-cancel,
    .confirm-danger {
        border: 1px solid var(--border);
        border-radius: var(--radius-md);
        padding: var(--sp-1) var(--sp-3);
        font-size: var(--fs-sm);
        font-weight: var(--fw-semibold);
        cursor: pointer;
    }
    .confirm-cancel {
        color: var(--text-sec);
        background: var(--bg-surface);
    }
    .confirm-danger {
        color: var(--text-inverse);
        background: var(--danger-strong);
        border-color: var(--danger-strong);
    }

    .drawer-content {
        flex: 1;
        overflow-y: auto;
        padding: var(--sp-3);
    }
    .history-item {
        width: 100%;
        padding: var(--sp-3);
        border-radius: var(--radius-lg);
        margin-bottom: var(--sp-2);
        cursor: pointer;
        border: 1px solid var(--border-soft);
        background: var(--bg-surface);
        color: inherit;
        text-align: left;
        box-shadow: var(--shadow-sm);
        transition: background var(--t-base) var(--ease-standard),
            border-color var(--t-base) var(--ease-standard),
            transform var(--t-fast) var(--ease-standard);
    }
    .history-item:hover {
        background: var(--bg-hover);
        border-color: var(--border-strong);
        transform: translateY(-1px);
    }
    .h-meta {
        display: flex;
        justify-content: space-between;
        font-size: var(--fs-xs);
        color: var(--text-sec);
        margin-bottom: var(--sp-2);
        font-weight: var(--fw-semibold);
    }
    .h-lang {
        display: flex;
        align-items: center;
        gap: var(--sp-1);
        color: var(--primary);
    }
    .h-preview {
        display: -webkit-box;
        font-size: var(--fs-base);
        font-weight: var(--fw-medium);
        color: var(--text-main);
        line-height: var(--lh-snug);
        line-clamp: 2;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
    }
    .h-output {
        display: -webkit-box;
        margin-top: var(--sp-2);
        color: var(--text-sec);
        font-size: var(--fs-sm);
        line-height: var(--lh-snug);
        line-clamp: 2;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
    }
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: var(--text-sec);
        gap: var(--sp-3);
    }

    @media (max-width: 480px) {
        .drawer {
            width: 100vw;
        }
    }
</style>
