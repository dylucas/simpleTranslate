<script lang="ts">
  import { AlertCircle, X } from "@lucide/svelte";
  import type { ErrorToast as ErrorToastValue } from "./types";

  interface Props {
    errorToast: ErrorToastValue | null;
    onRetry: () => void;
    onSettings: () => void;
    onDismiss: () => void;
  }

  let { errorToast, onRetry, onSettings, onDismiss }: Props = $props();
</script>

{#if errorToast}
  <div class="error-toast" role="alert" aria-live="assertive">
    <AlertCircle size={16} />
    <span>{errorToast.msg}</span>
    {#if errorToast.canRetry}<button onclick={onRetry}>重试</button>{/if}
    {#if errorToast.showSettings}<button onclick={onSettings}>前往设置</button>{/if}
    <button class="close" onclick={onDismiss} aria-label="关闭错误提示"><X size={14} /></button>
  </div>
{/if}

<style>
  .error-toast {
    position: fixed;
    top: var(--sp-3);
    left: 50%;
    z-index: var(--z-toast);
    display: flex;
    width: max-content;
    max-width: min(560px, calc(100vw - var(--sp-8)));
    align-items: center;
    gap: var(--sp-2);
    padding: 9px var(--sp-3);
    transform: translateX(-50%);
    border: 1px solid color-mix(in srgb, var(--danger) 60%, transparent);
    border-radius: var(--radius-md);
    background: var(--danger-strong);
    box-shadow: var(--shadow-md);
    color: var(--text-inverse);
    font-size: var(--fs-sm);
  }
  .error-toast span { flex: 1; overflow-wrap: anywhere; }
  button {
    border: 0;
    border-radius: var(--radius-sm);
    background: var(--on-color-soft);
    color: inherit;
    cursor: pointer;
    padding: var(--sp-1) var(--sp-2);
    font: inherit;
  }
  button:hover { background: var(--on-color-hover); }
  .close { display: grid; width: 28px; height: 28px; padding: 0; place-items: center; background: transparent; }
</style>
