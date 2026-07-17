<script lang="ts">
  import { AlertCircle, X } from "lucide-svelte";
  import { fade } from "svelte/transition";
  import { createEventDispatcher } from "svelte";
  import type { ErrorToast } from "./types";

  export let errorToast: ErrorToast | null = null;

  const dispatch = createEventDispatcher<{ retry: void; settings: void; dismiss: void }>();
</script>

{#if errorToast}
  <div class="error-toast" transition:fade={{ duration: 200 }}>
    <AlertCircle size={16} />
    <span class="error-toast-msg">{errorToast.msg}</span>
    {#if errorToast.canRetry}
      <button class="error-toast-action" on:click={() => dispatch("retry")}>重试</button>
    {/if}
    {#if errorToast.showSettings}
      <button class="error-toast-action" on:click={() => dispatch("settings")}>前往设置</button>
    {/if}
    <button class="error-toast-close" on:click={() => dispatch("dismiss")}>
      <X size={14} />
    </button>
  </div>
{/if}

<style>
  .error-toast {
    position: fixed;
    top: 20px;
    left: 50%;
    transform: translateX(-50%);
    z-index: 2000;
    display: flex;
    align-items: center;
    gap: 10px;
    background: var(--danger);
    color: var(--text-inverse);
    padding: 10px 16px;
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    font-size: var(--fs-base);
    font-weight: var(--fw-medium);
    max-width: 80vw;
  }
  .error-toast-msg {
    flex: 1;
    word-break: break-word;
  }
  .error-toast-action {
    background: rgba(255, 255, 255, 0.18);
    border: none;
    color: #fff;
    cursor: pointer;
    padding: 3px 10px;
    border-radius: 4px;
    font-size: var(--fs-xs);
    font-weight: var(--fw-medium);
    transition: background 0.2s;
  }
  .error-toast-action:hover {
    background: rgba(255, 255, 255, 0.3);
  }
  .error-toast-close {
    background: transparent;
    border: none;
    color: #fff;
    cursor: pointer;
    padding: 2px;
    border-radius: 4px;
    display: flex;
    align-items: center;
    opacity: 0.85;
  }
  .error-toast-close:hover {
    opacity: 1;
    background: rgba(255, 255, 255, 0.15);
  }
</style>
