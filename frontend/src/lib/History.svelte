<script>
    import { ArrowLeftRight, History, Trash2, X, Search, Download } from "lucide-svelte";
    import { fade, fly } from "svelte/transition";
    import { createEventDispatcher } from "svelte";

    export let show = false;
    export let history = [];

    const dispatch = createEventDispatcher();

    let searchTerm = "";

    function handleItemClick(item) {
        dispatch("select", item);
        show = false;
    }

    function handleClear() {
        dispatch("clear");
    }

    function handleClose() {
        show = false;
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
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <div
        class="backdrop"
        on:click={handleClose}
        transition:fade={{ duration: 200 }}
    ></div>
    <aside
        class="drawer"
        transition:fly={{ x: 360, duration: 300, opacity: 1 }}
    >
        <div class="drawer-header">
            <h3>历史记录</h3>
            <div class="drawer-actions">
                <button
                    class="icon-btn"
                    on:click={handleExport}
                    disabled={!history || history.length === 0}
                    title="导出为 JSON"
                ><Download size={16} /></button>
                <button class="icon-btn" on:click={handleClear} title="清空历史"
                    ><Trash2 size={16} /></button
                >
                <button class="icon-btn" on:click={handleClose}
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
                />
                {#if searchTerm}
                    <button class="search-clear" on:click={() => (searchTerm = "")}>
                        <X size={12} />
                    </button>
                {/if}
            </div>
            {#if searchTerm}
                <span class="search-count">{filteredHistory.length} 条匹配</span>
            {/if}
        </div>
        <div class="drawer-content">
            {#each filteredHistory as item}
                <!-- svelte-ignore a11y-click-events-have-key-events -->
                <div
                    class="history-item"
                    on:click={() => handleItemClick(item)}
                >
                    <div class="h-meta">
                        <span class="h-lang"
                            >{item.source.toUpperCase()}
                            <ArrowLeftRight size={10} />
                            {item.target.toUpperCase()}</span
                        >
                        <span class="h-time">{item.time}</span>
                    </div>
                    <div class="h-preview">{item.input}</div>
                </div>
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
        backdrop-filter: blur(2px);
        -webkit-backdrop-filter: blur(2px);
        z-index: var(--z-drawer);
    }
    .drawer {
        position: fixed;
        top: 0;
        right: 0;
        bottom: 0;
        width: 320px;
        background: var(--bg-surface);
        z-index: 51;
        border-left: 1px solid var(--border);
        display: flex;
        flex-direction: column;
        box-shadow: var(--shadow-lg);
    }
    .drawer-header {
        height: 64px;
        padding: 0 20px;
        border-bottom: 1px solid var(--border);
        display: flex;
        align-items: center;
        justify-content: space-between;
        flex-shrink: 0;
    }
    .drawer-header h3 {
        margin: 0;
        font-size: 16px;
        font-weight: 600;
    }

    .drawer-actions {
        display: flex;
        gap: 4px;
    }
    .icon-btn {
        background: transparent;
        border: none;
        color: var(--text-sec);
        padding: 6px;
        border-radius: 4px;
        cursor: pointer;
    }
    .icon-btn:hover {
        background: var(--bg-hover);
        color: var(--text-main);
    }
    .icon-btn:disabled {
        opacity: 0.4;
        cursor: not-allowed;
    }

    .search-bar {
        padding: 10px 16px;
        border-bottom: 1px solid var(--border);
        display: flex;
        align-items: center;
        gap: 8px;
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
        left: 10px;
        color: var(--text-sec);
        pointer-events: none;
    }
    .search-input-wrapper input {
        width: 100%;
        background: var(--bg-hover);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 7px 28px 7px 30px;
        color: var(--text-main);
        font-size: 13px;
        outline: none;
        transition: border-color 0.2s;
    }
    .search-input-wrapper input:focus {
        border-color: var(--primary);
    }
    .search-clear {
        position: absolute;
        right: 6px;
        background: transparent;
        border: none;
        color: var(--text-sec);
        cursor: pointer;
        padding: 2px;
        border-radius: 4px;
        display: flex;
        align-items: center;
    }
    .search-clear:hover {
        color: var(--text-main);
        background: var(--bg-hover);
    }
    .search-count {
        font-size: 11px;
        color: var(--text-sec);
        font-weight: 600;
        white-space: nowrap;
    }

    .drawer-content {
        flex: 1;
        overflow-y: auto;
        padding: 10px;
    }
    .history-item {
        padding: 12px;
        border-radius: 8px;
        margin-bottom: 8px;
        cursor: pointer;
        border: 1px solid transparent;
        transition: 0.2s;
    }
    .history-item:hover {
        background: var(--bg-hover);
        border-color: var(--border);
    }
    .h-meta {
        display: flex;
        justify-content: space-between;
        font-size: 11px;
        color: var(--text-sec);
        margin-bottom: 6px;
        font-weight: 600;
    }
    .h-lang {
        display: flex;
        align-items: center;
        gap: 4px;
        color: var(--primary);
    }
    .h-preview {
        font-size: 13px;
        color: var(--text-main);
        line-height: 1.4;
        display: -webkit-box;
        line-clamp: 2;
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
        gap: 10px;
        opacity: 0.5;
    }
</style>
