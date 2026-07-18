<script lang="ts">
  import { AlertCircle, X } from "lucide-svelte";
  import { fade } from "svelte/transition";
  import { createEventDispatcher } from "svelte";
  import type { ErrorToast } from "./types";

  export let errorToast: ErrorToast | null = null;

  const dispatch = createEventDispatcher<{ retry: void; settings: void; dismiss: void }>();
</script>

{#if errorToast}
  <div class="error-toast" role="alert" aria-live="assertive" transition:fade={{ duration: 200 }}>
    <AlertCircle size={16} />
    <span class="error-toast-msg">{errorToast.msg}</span>
    {#if errorToast.canRetry}
      <button class="error-toast-action" on:click={() => dispatch("retry")}>重试</button>
    {/if}
    {#if errorToast.showSettings}
      <button class="error-toast-action" on:click={() => dispatch("settings")}>前往设置</button>
    {/if}
    <button class="error-toast-close" on:click={() => dispatch("dismiss")} aria-label="关闭错误提示">
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
    z-index: var(--z-toast);
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    background: var(--danger-strong);
    color: var(--text-inverse);
    padding: var(--sp-3) var(--sp-4);
    border-radius: var(--radius-md);
    box-shadow: var(--shadow-lg);
    font-size: var(--fs-base);
    font-weight: var(--fw-medium);
    width: max-content;
    max-width: min(560px, calc(100vw - var(--sp-8)));
    border: 1px solid var(--danger);
  }
  .error-toast-msg {
    flex: 1;
    word-break: break-word;
  }
  .error-toast-action {
    background: var(--on-color-soft);
    border: none;
    color: var(--text-inverse);
    cursor: pointer;
    min-height: var(--sp-6);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-sm);
    font-size: var(--fs-xs);
    font-weight: var(--fw-medium);
    transition: background var(--t-base) var(--ease-standard);
  }
  .error-toast-action:hover {
    background: var(--on-color-hover);
  }
  .error-toast-close {
    background: transparent;
    border: none;
    color: var(--text-inverse);
    cursor: pointer;
    width: var(--sp-6);
    height: var(--sp-6);
    padding: 0;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0.85;
  }
  .error-toast-close:hover {
    opacity: 1;
    background: var(--on-color-soft);
  }
</style>
