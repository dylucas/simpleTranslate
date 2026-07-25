<script lang="ts">
  import {
    CheckCircle2, Cloud, Cpu, Eye, EyeOff, Globe, KeyRound,
    RefreshCcw, Save, Settings, X, XCircle,
  } from "@lucide/svelte";
  import { tick } from "svelte";
  import { BAIDU_DOMAIN_OPTIONS } from "./baiduDomains";
  import { cloneConfig, DEFAULT_CONFIG } from "./bridge";
  import { ARIA_SHORTCUTS } from "./shortcuts";
  import type { BaiduConfig, BaiduDomain, CloudConfig, EngineId, ServiceConfig } from "./types";

  interface Props {
    open: boolean;
    config: CloudConfig;
    onClose: () => void;
    onSave: (config: CloudConfig) => Promise<void>;
    onTest: (engine: EngineId, service: ServiceConfig | BaiduConfig) => Promise<void>;
  }

  interface ConnectionState { testing: boolean; ok: boolean; message: string; }

  let { open, config, onClose, onSave, onTest }: Props = $props();
  let draft = $state(cloneConfig(DEFAULT_CONFIG));
  let saving = $state(false);
  let message = $state("");
  let showTencentKey = $state(false);
  let showAliyunId = $state(false);
  let showAliyunKey = $state(false);
  let showBaiduId = $state(false);
  let showBaiduKey = $state(false);
  let connection = $state<Partial<Record<EngineId, ConnectionState>>>({});
  let modal = $state<HTMLElement>();
  let wasOpen = false;
  const connectionRuns: Record<EngineId, number> = { tencent: 0, aliyun: 0, baidu: 0 };

  function resetConnectionTests(): void {
    connectionRuns.tencent += 1;
    connectionRuns.aliyun += 1;
    connectionRuns.baidu += 1;
    connection = {};
  }

  $effect(() => {
    if (open && !wasOpen) {
      wasOpen = true;
      draft = cloneConfig(config);
      resetConnectionTests();
      message = "";
      void tick().then(() => modal?.focus());
    } else if (!open && wasOpen) {
      wasOpen = false;
      resetConnectionTests();
    }
  });

  function updateService(engine: "tencent" | "aliyun", key: keyof ServiceConfig, value: string): void {
    draft[engine] = { ...draft[engine], [key]: value };
    connectionRuns[engine] += 1;
    connection[engine] = undefined;
  }

  function updateBaidu(key: keyof BaiduConfig, value: string): void {
    draft.baidu = { ...draft.baidu, [key]: value } as BaiduConfig;
    connectionRuns.baidu += 1;
    connection.baidu = undefined;
  }

  function close(): void {
    if (!saving) onClose();
  }

  async function save(): Promise<void> {
    if (saving) return;
    saving = true;
    message = "正在保存...";
    try {
      await onSave(cloneConfig(draft));
      message = "设置已保存";
      onClose();
    } catch {
      message = "保存失败，请重试";
    } finally {
      saving = false;
    }
  }

  async function test(engine: EngineId): Promise<void> {
    const run = ++connectionRuns[engine];
    connection[engine] = { testing: true, ok: false, message: "" };
    try {
      await onTest(engine, { ...draft[engine] });
      if (run !== connectionRuns[engine] || !open) return;
      connection[engine] = { testing: false, ok: true, message: "连接成功" };
    } catch (error) {
      if (run !== connectionRuns[engine] || !open) return;
      const text = error instanceof Error ? error.message : String(error);
      connection[engine] = { testing: false, ok: false, message: text.replace(/^Error:\s*/, "") };
    }
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (saving) {
      event.stopPropagation();
      if (event.key === "Tab" || event.key === "Escape") event.preventDefault();
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      close();
      return;
    }
    if (event.key !== "Tab" || !modal) return;
    const focusable = [...modal.querySelectorAll<HTMLElement>("button:not(:disabled), input, select")];
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
</script>

{#if open}
  <button class="backdrop" onclick={close} disabled={saving} aria-label="关闭偏好设置"></button>
  <div class="modal" bind:this={modal} role="dialog" aria-modal="true" aria-labelledby="config-title" tabindex="-1" onkeydown={handleKeydown}>
    <header>
      <div class="title"><span><Settings size={20} /></span><div><h2 id="config-title">偏好设置</h2><small>翻译服务与工作区偏好</small></div></div>
      <button class="icon-btn" onclick={close} disabled={saving} aria-label="关闭偏好设置" aria-keyshortcuts={ARIA_SHORTCUTS.closePanel}><X size={18} /></button>
    </header>

    <main inert={saving} aria-busy={saving}>
      <section class="settings-section">
        <h3><Cpu size={15} />核心偏好</h3>
        <label class="setting-row"><span><strong>默认翻译引擎</strong><small>单引擎模式和对照首选结果</small></span>
          <select value={draft.defaultEngine} onchange={(event) => (draft.defaultEngine = event.currentTarget.value as EngineId)} aria-label="默认翻译引擎">
            <option value="tencent">腾讯混元</option><option value="aliyun">阿里云 MT</option><option value="baidu">百度翻译</option>
          </select>
        </label>
      </section>

      <section class="settings-section">
        <h3><Cloud size={15} />腾讯混元 hy-mt2-pro</h3>
        <div class="fields">
          <label><span>API Key</span><div class="input-wrap"><KeyRound size={15} /><input type={showTencentKey ? "text" : "password"} value={draft.tencent.secretKey} oninput={(event) => updateService("tencent", "secretKey", event.currentTarget.value)} placeholder="TokenHub API Key (sk-...)" /><button type="button" onclick={() => (showTencentKey = !showTencentKey)} aria-label="显示或隐藏腾讯 API Key">{#if showTencentKey}<EyeOff size={15} />{:else}<Eye size={15} />{/if}</button></div></label>
        </div>
        <div class="test-row">
          <button class="test-btn" onclick={() => void test("tencent")} disabled={connection.tencent?.testing || !draft.tencent.secretKey.trim()} aria-label="测试腾讯混元连接">
            <RefreshCcw size={14} class={connection.tencent?.testing ? "spin" : ""} />{connection.tencent?.testing ? "测试中" : "测试连接"}
          </button>
          {#if connection.tencent?.message}<span class:ok={connection.tencent.ok} class="test-message">{#if connection.tencent.ok}<CheckCircle2 size={14} />{:else}<XCircle size={14} />{/if}{connection.tencent.message}</span>{/if}
        </div>
      </section>

      <section class="settings-section">
        <h3><Cloud size={15} />阿里云机器翻译</h3>
        <div class="fields two-columns">
          <label><span>AccessKey ID</span><div class="input-wrap"><KeyRound size={15} /><input type={showAliyunId ? "text" : "password"} value={draft.aliyun.secretId} oninput={(event) => updateService("aliyun", "secretId", event.currentTarget.value)} /><button type="button" onclick={() => (showAliyunId = !showAliyunId)} aria-label="显示或隐藏 AccessKey ID">{#if showAliyunId}<EyeOff size={15} />{:else}<Eye size={15} />{/if}</button></div></label>
          <label><span>AccessKey Secret</span><div class="input-wrap"><KeyRound size={15} /><input type={showAliyunKey ? "text" : "password"} value={draft.aliyun.secretKey} oninput={(event) => updateService("aliyun", "secretKey", event.currentTarget.value)} /><button type="button" onclick={() => (showAliyunKey = !showAliyunKey)} aria-label="显示或隐藏 AccessKey Secret">{#if showAliyunKey}<EyeOff size={15} />{:else}<Eye size={15} />{/if}</button></div></label>
          <label class="endpoint"><span>服务地域或地址</span><div class="input-wrap"><Globe size={15} /><input value={draft.aliyun.region} oninput={(event) => updateService("aliyun", "region", event.currentTarget.value)} placeholder="cn-hangzhou" /></div></label>
        </div>
        <div class="test-row">
          <button class="test-btn" onclick={() => void test("aliyun")} disabled={connection.aliyun?.testing || !draft.aliyun.secretId.trim() || !draft.aliyun.secretKey.trim()} aria-label="测试阿里云连接">
            <RefreshCcw size={14} class={connection.aliyun?.testing ? "spin" : ""} />{connection.aliyun?.testing ? "测试中" : "测试连接"}
          </button>
          {#if connection.aliyun?.message}<span class:ok={connection.aliyun.ok} class="test-message">{#if connection.aliyun.ok}<CheckCircle2 size={14} />{:else}<XCircle size={14} />{/if}{connection.aliyun.message}</span>{/if}
        </div>
      </section>

      <section class="settings-section">
        <h3><Cloud size={15} />百度翻译</h3>
        <div class="fields two-columns">
          <label><span>APP ID</span><div class="input-wrap"><KeyRound size={15} /><input type={showBaiduId ? "text" : "password"} value={draft.baidu.appId} oninput={(event) => updateBaidu("appId", event.currentTarget.value)} /><button type="button" onclick={() => (showBaiduId = !showBaiduId)} aria-label="显示或隐藏百度 APP ID">{#if showBaiduId}<EyeOff size={15} />{:else}<Eye size={15} />{/if}</button></div></label>
          <label><span>密钥</span><div class="input-wrap"><KeyRound size={15} /><input type={showBaiduKey ? "text" : "password"} value={draft.baidu.secretKey} oninput={(event) => updateBaidu("secretKey", event.currentTarget.value)} /><button type="button" onclick={() => (showBaiduKey = !showBaiduKey)} aria-label="显示或隐藏百度密钥">{#if showBaiduKey}<EyeOff size={15} />{:else}<Eye size={15} />{/if}</button></div></label>
          <label class="endpoint"><span>翻译领域</span><select value={draft.baidu.domain} onchange={(event) => updateBaidu("domain", event.currentTarget.value as BaiduDomain)} aria-label="百度翻译领域">
            {#each BAIDU_DOMAIN_OPTIONS as option}<option value={option.value}>{option.label}</option>{/each}
          </select></label>
        </div>
        <div class="test-row">
          <button class="test-btn" onclick={() => void test("baidu")} disabled={connection.baidu?.testing || !draft.baidu.appId.trim() || !draft.baidu.secretKey.trim()} aria-label="测试百度翻译连接">
            <RefreshCcw size={14} class={connection.baidu?.testing ? "spin" : ""} />{connection.baidu?.testing ? "测试中" : "测试连接"}
          </button>
          {#if connection.baidu?.message}<span class:ok={connection.baidu.ok} class="test-message">{#if connection.baidu.ok}<CheckCircle2 size={14} />{:else}<XCircle size={14} />{/if}{connection.baidu.message}</span>{/if}
        </div>
      </section>
    </main>

    <footer>
      <span role="status" aria-live="polite">{message}</span>
      <div><button class="secondary" onclick={close} disabled={saving}>取消</button><button class="primary" onclick={() => void save()} disabled={saving}>{#if saving}<RefreshCcw size={15} class="spin" />{:else}<Save size={15} />{/if}保存配置</button></div>
    </footer>
  </div>
{/if}

<style>
  .backdrop { position: fixed; inset: 0; z-index: var(--z-modal); border: 0; background: var(--bg-overlay); cursor: default; }
  .modal { position: fixed; z-index: calc(var(--z-modal) + 1); top: 50%; left: 50%; display: flex; width: min(680px, calc(100vw - 32px)); max-height: calc(100vh - 32px); flex-direction: column; overflow: hidden; transform: translate(-50%, -50%); outline: 0; border: 1px solid var(--border); border-radius: var(--radius-lg); background: var(--bg-elevated); box-shadow: var(--shadow-xl); }
  header, footer { display: flex; min-height: 52px; flex: 0 0 auto; align-items: center; justify-content: space-between; padding: var(--sp-2) var(--sp-4); }
  header { border-bottom: 1px solid var(--border); }
  footer { border-top: 1px solid var(--border); color: var(--text-muted); font-size: var(--fs-sm); }
  .title { display: flex; align-items: center; gap: var(--sp-2); }
  .title > span { display: grid; width: 30px; height: 30px; place-items: center; border-radius: var(--radius-md); background: var(--primary-soft); color: var(--primary); }
  h2 { margin: 0; color: var(--text-main); font-size: var(--fs-md); }
  small { color: var(--text-muted); font-size: var(--fs-xs); }
  button, input, select { font-family: inherit; }
  button { cursor: pointer; }
  .icon-btn { display: grid; width: 30px; height: 30px; place-items: center; border: 0; border-radius: var(--radius-sm); background: transparent; color: var(--text-sec); }
  .icon-btn:hover { background: var(--bg-hover); color: var(--text-main); }
  main { display: block; min-height: 0; overflow: auto; padding: 0 var(--sp-5); }
  .settings-section { display: grid; gap: var(--sp-3); border-bottom: 1px solid var(--border-soft); padding: var(--sp-4) 0; }
  .settings-section:last-child { border-bottom: 0; }
  h3 { display: flex; align-items: center; gap: var(--sp-2); margin: 0; color: var(--text-sec); font-size: var(--fs-sm); }
  h3 :global(svg) { color: var(--primary); }
  .setting-row { display: flex; align-items: center; justify-content: space-between; gap: var(--sp-4); padding: 0; }
  .setting-row > span { display: grid; gap: var(--sp-1); }
  .setting-row strong { color: var(--text-main); font-size: var(--fs-base); }
  select { min-height: 34px; border: 1px solid var(--border); border-radius: var(--radius-md); padding: 0 var(--sp-3); background: var(--bg-input); color: var(--text-main); }
  .fields { display: grid; gap: var(--sp-3); }
  .two-columns { grid-template-columns: 1fr 1fr; }
  .endpoint { grid-column: 1 / -1; }
  .fields label { display: grid; gap: var(--sp-2); color: var(--text-sec); font-size: var(--fs-sm); }
  .input-wrap { display: flex; min-height: 36px; align-items: center; gap: var(--sp-2); border: 1px solid var(--border); border-radius: var(--radius-md); padding-left: var(--sp-3); background: var(--bg-input); color: var(--text-muted); }
  .input-wrap:focus-within { border-color: var(--primary); box-shadow: 0 0 0 2px var(--primary-soft); }
  .input-wrap input { min-width: 0; flex: 1; border: 0; outline: 0; background: transparent; color: var(--text-main); }
  .input-wrap button { display: grid; width: 34px; align-self: stretch; place-items: center; border: 0; background: transparent; color: var(--text-muted); }
  .test-row { display: flex; min-height: 30px; align-items: center; gap: var(--sp-3); }
  .test-btn, footer button { display: inline-flex; min-height: 32px; align-items: center; justify-content: center; gap: var(--sp-2); border: 1px solid var(--border); border-radius: var(--radius-md); padding: 0 var(--sp-3); background: var(--bg-surface); color: var(--text-sec); }
  .test-btn:hover:not(:disabled), footer .secondary:hover { background: var(--bg-hover); color: var(--text-main); }
  button:disabled { cursor: not-allowed; opacity: .45; }
  .test-message { display: flex; align-items: center; gap: var(--sp-1); color: var(--danger); font-size: var(--fs-sm); }
  .test-message.ok { color: var(--success); }
  footer > div { display: flex; gap: var(--sp-2); }
  footer .primary { border-color: var(--primary); background: var(--primary); color: white; }
  .modal :global(.spin) { animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (max-width: 620px) { .modal { width: calc(100vw - 12px); max-height: calc(100vh - 12px); } header, footer { padding-inline: var(--sp-3); } main { padding-inline: var(--sp-3); } .two-columns { grid-template-columns: 1fr; } .endpoint { grid-column: auto; } footer { align-items: flex-end; flex-direction: column; gap: var(--sp-2); } }
</style>
