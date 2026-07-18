<script lang="ts">
  import type { EngineId } from "./types";
  import { langs } from "./languages";

  interface Props {
    status: string;
    bridgeKind: "wails" | "mock";
    source: string;
    target: string;
    activeEngine: EngineId;
    compareMode: boolean;
  }
  let { status, bridgeKind, source, target, activeEngine, compareMode }: Props = $props();
  let statusClass = $derived(status === "翻译中..." ? "processing" : status.includes("失败") ? "error" : "done");
  let routeLabel = $derived(`${source === "auto" ? "自动识别" : (langs[source] ?? source)} → ${langs[target] ?? target}`);
  let engineLabel = $derived(compareMode ? "多引擎对照" : activeEngine === "tencent" ? "腾讯混元" : "阿里云 MT");
</script>

<footer class="status-bar">
  <div class="status" role="status" aria-live="polite">
    <span class="status-dot {statusClass}"></span>{status}
    {#if bridgeKind === "mock"}<span class="preview-badge">预览</span>{/if}
  </div>
  <span class="context"><span>{routeLabel}</span><i></i><span>{engineLabel}</span></span>
</footer>

<style>
  .status-bar {
    display: flex;
    height: var(--statusbar-h);
    flex: 0 0 auto;
    align-items: center;
    justify-content: space-between;
    border-top: 1px solid var(--border-soft);
    padding: 0 var(--sp-3);
    background: var(--bg-rail);
    color: var(--text-muted);
    font-size: var(--fs-xs);
  }
  .status { display: flex; align-items: center; gap: var(--sp-2); color: var(--text-sec); }
  .status-dot { width: 6px; height: 6px; border-radius: 50%; background: var(--success); }
  .status-dot.processing { background: var(--warning); animation: pulse 1s infinite; }
  .status-dot.error { background: var(--danger); }
  .preview-badge { border: 1px solid var(--border); border-radius: var(--radius-full); padding: 1px 6px; }
  .context { display: flex; min-width: 0; align-items: center; gap: var(--sp-2); overflow: hidden; color: var(--text-muted); white-space: nowrap; }
  .context span { overflow: hidden; text-overflow: ellipsis; }
  .context i { width: 1px; height: 10px; flex: 0 0 auto; background: var(--border-strong); }
  @keyframes pulse { 50% { opacity: .35; } }
  @media (max-width: 720px) { .context { max-width: 55%; } }
  @media (max-width: 420px) { .context i, .context span:last-child { display: none; } }
</style>
