<script>
    import { ArrowLeftRight, History, Trash2, X } from "lucide-svelte";
    import { fade, fly } from "svelte/transition";
    import { createEventDispatcher } from "svelte";

    export let show = false;
    export let history = [];

    const dispatch = createEventDispatcher();

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
                <button class="icon-btn" on:click={handleClear} title="清空历史"
                    ><Trash2 size={16} /></button
                >
                <button class="icon-btn" on:click={handleClose}
                    ><X size={18} /></button
                >
            </div>
        </div>
        <div class="drawer-content">
            {#each history as item}
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
                    <p>暂无翻译记录</p>
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
        background: rgba(0, 0, 0, 0.5);
        backdrop-filter: blur(2px);
        z-index: 50;
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
        box-shadow: -4px 0 15px rgba(0, 0, 0, 0.3);
    }
    .drawer-header {
        height: 64px;
        padding: 0 20px;
        border-bottom: 1px solid var(--border);
        display: flex;
        align-items: center;
        justify-content: space-between;
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
