<script>
  import { SaveConfig, GetConfig, TestConnection } from "../../wailsjs/go/main/App";
  import {
    X,
    Save,
    Globe,
    Cloud,
    RefreshCcw,
    Settings,
    KeyRound,
    Cpu,
    CheckCircle2,
    XCircle,
  } from "lucide-svelte";
  import { fade, fly } from "svelte/transition";
  import { tick } from "svelte";
  import { configStore } from "./store";

  export let show = false;
  export let isDark = true; // 接收父组件传入的主题状态

  let saving = false;
  let message = "";
  let config;
  // 连接测试状态：{ [engine]: { testing, ok, msg } }
  let connStatus = {};
  let modalEl;
  let wasShown = false;

  // 每次打开同步 Store 数据（使用 $configStore 自动订阅，避免手动 subscribe 带来的取消订阅问题）
  $: if (show) {
    config = JSON.parse(JSON.stringify($configStore));
  }

  $: if (show && !wasShown) {
    wasShown = true;
    tick().then(() => modalEl?.focus());
  }
  $: if (!show) {
    wasShown = false;
  }

  async function handleSave() {
    saving = true;
    message = "正在同步...";
    try {
      // @ts-ignore
      await SaveConfig(config);
      configStore.set(config);
      message = "设置已保存";
      setTimeout(() => {
        message = "";
        show = false;
      }, 500);
    } catch (e) {
      message = "保存失败";
    } finally {
      saving = false;
    }
  }

  // 测试指定引擎的连接（先保存当前配置，再用最小请求验证凭据）
  async function handleTest(engine) {
    if (!config) return;
    connStatus = { ...connStatus, [engine]: { testing: true, ok: false, msg: "" } };
    try {
      await SaveConfig(config);
      configStore.set(config);
      await TestConnection(engine);
      connStatus = { ...connStatus, [engine]: { testing: false, ok: true, msg: "连接成功" } };
    } catch (e) {
      connStatus = { ...connStatus, [engine]: { testing: false, ok: false, msg: String(e) } };
    }
  }
</script>

{#if show}
  <!-- svelte-ignore a11y-click-events-have-key-events -->
  <div
    class="overlay"
    class:light-mode={!isDark}
    on:click={() => (show = false)}
    transition:fade={{ duration: 200 }}
  >
    <div
      class="modal"
      bind:this={modalEl}
      role="dialog"
      aria-modal="true"
      aria-labelledby="config-title"
      aria-describedby="config-description"
      tabindex="-1"
      on:click|stopPropagation
      transition:fly={{ y: 20, duration: 300 }}
    >
      <header class="modal-header">
        <div class="header-main">
          <div class="brand-icon">
            <Settings size={22} strokeWidth={2.5} />
          </div>
          <div class="header-info">
            <h3 id="config-title">偏好设置</h3>
            <span id="config-description">管理 API 凭据和翻译服务</span>
          </div>
        </div>
        <button class="close-btn" on:click={() => (show = false)} aria-label="关闭偏好设置">
          <X size={18} />
        </button>
      </header>

      <main class="modal-content">
        <section class="settings-group">
          <div class="group-title"><Cpu size={14} /> 核心偏好</div>
          <div class="settings-card">
            <div class="setting-item">
              <div class="item-label">
                <span class="main-label">默认翻译引擎</span>
                <span class="sub-label">切换翻译时首选的云服务商</span>
              </div>
              <div class="item-control">
                <div class="select-wrapper">
                  <select bind:value={config.defaultEngine} aria-label="默认翻译引擎">
                    <option value="tencent">腾讯混元 (hy-mt2-pro)</option>
                    <option value="aliyun">阿里云 (MT)</option>
                  </select>
                </div>
              </div>
            </div>
          </div>
        </section>

        <section class="settings-group">
          <div class="group-title"><Cloud size={14} /> 腾讯混元 hy-mt2-pro</div>
          <div class="input-card">
            <div class="input-field">
              <label for="t-sk">API Key</label>
              <div class="input-wrapper">
                <KeyRound size={14} />
                <input
                  id="t-sk"
                  type="password"
                  bind:value={config.tencent.secretKey}
                  placeholder="TokenHub API Key (sk-...)"
                />
              </div>
            </div>
            <div class="test-row">
              <button
                class="test-btn"
                on:click={() => handleTest("tencent")}
                disabled={connStatus?.tencent?.testing || !config?.tencent?.secretKey}
              >
                {#if connStatus?.tencent?.testing}
                  <RefreshCcw size={14} class="spin" />
                  测试中...
                {:else}
                  <RefreshCcw size={14} />
                  测试连接
                {/if}
              </button>
              {#if connStatus?.tencent?.msg}
                <span class="test-msg" class:ok={connStatus?.tencent?.ok} class:err={!connStatus?.tencent?.ok}>
                  {#if connStatus?.tencent?.ok}<CheckCircle2 size={13} />{:else}<XCircle size={13} />{/if}
                  {connStatus.tencent.msg}
                </span>
              {/if}
            </div>
          </div>
        </section>

        <section class="settings-group">
          <div class="group-title"><Cloud size={14} /> 阿里云 MT</div>
          <div class="input-card">
            <div class="input-row">
              <div class="input-field">
                <label for="a-sid">AccessKey ID</label>
                <div class="input-wrapper">
                  <KeyRound size={14} />
                  <input
                    id="a-sid"
                    type="password"
                    bind:value={config.aliyun.secretId}
                    placeholder="AccessKey ID"
                  />
                </div>
              </div>
              <div class="input-field">
                <label for="a-sk">AccessKey Secret</label>
                <div class="input-wrapper">
                  <KeyRound size={14} />
                  <input
                    id="a-sk"
                    type="password"
                    bind:value={config.aliyun.secretKey}
                    placeholder="AccessKey Secret"
                  />
                </div>
              </div>
            </div>
            <div class="input-field">
              <label for="a-reg">服务地址</label>
              <div class="input-wrapper">
                <Globe size={14} />
                <input
                  id="a-reg"
                  bind:value={config.aliyun.region}
                  placeholder="默认: mt.cn-hangzhou.aliyuncs.com"
                />
              </div>
            </div>
            <div class="test-row">
              <button
                class="test-btn"
                on:click={() => handleTest("aliyun")}
                disabled={connStatus?.aliyun?.testing || !config?.aliyun?.secretId}
              >
                {#if connStatus?.aliyun?.testing}
                  <RefreshCcw size={14} class="spin" />
                  测试中...
                {:else}
                  <RefreshCcw size={14} />
                  测试连接
                {/if}
              </button>
              {#if connStatus?.aliyun?.msg}
                <span class="test-msg" class:ok={connStatus?.aliyun?.ok} class:err={!connStatus?.aliyun?.ok}>
                  {#if connStatus?.aliyun?.ok}<CheckCircle2 size={13} />{:else}<XCircle size={13} />{/if}
                  {connStatus.aliyun.msg}
                </span>
              {/if}
            </div>
          </div>
        </section>
      </main>

      <footer class="modal-footer">
        <div class="footer-status" role="status" aria-live="polite">{message}</div>
        <div class="footer-actions">
          <button class="secondary-btn" on:click={() => (show = false)}>
            取消
          </button>
          <button class="primary-btn" on:click={handleSave} disabled={saving}>
            {#if saving}
              <RefreshCcw size={16} class="spin" />
            {:else}
              <Save size={16} />
            {/if}
            保存配置
          </button>
        </div>
      </footer>
    </div>
  </div>
{/if}

<style>
  /* 主题变量继承自全局 .app-shell / .light-mode，此处仅定义遮罩层 */
  .overlay {
    position: fixed;
    inset: 0;
    background: var(--bg-overlay);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--sp-4);
    z-index: var(--z-modal);
  }

  .modal {
    width: 640px;
    max-width: calc(100vw - var(--sp-8));
    max-height: calc(100vh - var(--sp-8));
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--radius-2xl);
    box-shadow: var(--shadow-xl);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  /* Header */
  .modal-header {
    padding: var(--sp-5) var(--sp-6);
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border);
  }

  .header-main {
    display: flex;
    gap: var(--sp-3);
    align-items: center;
  }

  .brand-icon {
    background: var(--accent-grad);
    color: var(--text-inverse);
    width: var(--sp-10);
    height: var(--sp-10);
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-glow);
    flex-shrink: 0;
  }

  .header-info h3 {
    margin: 0;
    font-size: var(--fs-lg);
    font-weight: var(--fw-semibold);
    color: var(--text-main);
  }

  .header-info span {
    font-size: var(--fs-sm);
    color: var(--text-sec);
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    cursor: pointer;
    width: var(--sp-8);
    height: var(--sp-8);
    padding: 0;
    border-radius: var(--radius-full);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard);
  }

  .close-btn:hover {
    background: var(--bg-surface);
    color: var(--text-main);
  }

  /* Content */
  .modal-content {
    flex: 1;
    min-height: 0;
    padding: var(--sp-5) var(--sp-6);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: var(--sp-6);
  }

  .group-title {
    font-size: var(--fs-xs);
    font-weight: var(--fw-bold);
    color: var(--primary);
    text-transform: uppercase;
    letter-spacing: var(--tracking-widest);
    margin-bottom: var(--sp-3);
    display: flex;
    align-items: center;
    gap: var(--sp-2);
  }

  .settings-card,
  .input-card {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    padding: var(--sp-4);
    box-shadow: var(--shadow-sm);
  }

  .setting-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--sp-4);
  }

  .item-label {
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }

  .main-label {
    font-size: var(--fs-md);
    font-weight: var(--fw-medium);
    color: var(--text-main);
  }

  .sub-label {
    font-size: var(--fs-sm);
    color: var(--text-sec);
  }

  /* Forms */
  .input-card {
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
  }

  .input-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--sp-4);
  }

  .input-field {
    display: flex;
    flex-direction: column;
    gap: var(--sp-2);
  }

  .input-field label {
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    color: var(--text-sec);
    margin-left: var(--sp-1);
  }

  .input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .input-wrapper :global(svg) {
    position: absolute;
    left: var(--sp-3);
    color: var(--text-sec);
    pointer-events: none;
  }

  input,
  select {
    width: 100%;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    padding: var(--sp-3) var(--sp-3) var(--sp-3) var(--sp-10);
    color: var(--text-main);
    font-size: var(--fs-md);
    transition: border-color var(--t-base) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard);
  }

  select {
    padding-left: var(--sp-3);
    appearance: none;
    cursor: pointer;
  }

  input:focus,
  select:focus {
    border-color: var(--primary);
    box-shadow: 0 0 0 3px var(--accent-glow);
  }

  .test-row {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    flex-wrap: wrap;
  }
  .test-btn {
    background: var(--bg-input);
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: var(--sp-2) var(--sp-3);
    border-radius: var(--radius-md);
    font-size: var(--fs-base);
    font-weight: var(--fw-medium);
    display: inline-flex;
    align-items: center;
    gap: var(--sp-2);
    cursor: pointer;
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard),
      border-color var(--t-base) var(--ease-standard);
  }
  .test-btn:hover:not(:disabled) {
    border-color: var(--primary);
    color: var(--primary);
  }
  .test-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .test-msg {
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
  }
  .test-msg.ok {
    color: var(--success);
  }
  .test-msg.err {
    color: var(--danger);
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Footer */
  .modal-footer {
    padding: var(--sp-4) var(--sp-6) var(--sp-5);
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: var(--bg-elevated);
  }

  .footer-status {
    font-size: var(--fs-base);
    color: var(--primary);
    font-weight: var(--fw-medium);
  }

  .footer-actions {
    display: flex;
    gap: var(--sp-3);
  }

  .primary-btn {
    background: var(--accent-grad);
    color: var(--text-inverse);
    border: none;
    padding: var(--sp-2) var(--sp-5);
    border-radius: var(--radius-md);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    cursor: pointer;
    box-shadow: var(--shadow-glow);
    transition: transform var(--t-fast) var(--ease-standard),
      filter var(--t-base) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard);
  }

  .primary-btn:hover:not(:disabled) {
    filter: brightness(1.08);
    transform: translateY(-1px);
    box-shadow: var(--shadow-glow);
  }

  .primary-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .secondary-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: var(--sp-2) var(--sp-5);
    border-radius: var(--radius-md);
    font-size: var(--fs-md);
    cursor: pointer;
    transition: background var(--t-base) var(--ease-standard),
      border-color var(--t-base) var(--ease-standard);
  }

  .secondary-btn:hover {
    background: var(--bg-surface);
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }

  :global(.spin) {
    animation: spin 0.9s linear infinite;
  }

  /* Custom Scrollbar */
  .modal-content::-webkit-scrollbar {
    width: var(--sp-2);
  }
  .modal-content::-webkit-scrollbar-track {
    background: transparent;
  }
  .modal-content::-webkit-scrollbar-thumb {
    background: var(--border);
    border-radius: var(--radius-full);
  }

  @media (max-width: 640px) {
    .overlay {
      padding: var(--sp-2);
    }
    .modal {
      max-width: calc(100vw - var(--sp-4));
      max-height: calc(100vh - var(--sp-4));
      border-radius: var(--radius-xl);
    }
    .modal-header,
    .modal-content,
    .modal-footer {
      padding-inline: var(--sp-4);
    }
    .input-row {
      grid-template-columns: 1fr;
    }
    .setting-item {
      align-items: stretch;
      flex-direction: column;
    }
    .item-control {
      width: 100%;
    }
    .modal-footer {
      gap: var(--sp-3);
    }
  }

  @media (max-height: 520px) {
    .overlay {
      padding: 0;
    }
    .modal {
      width: 100vw;
      max-width: 100vw;
      max-height: 100vh;
      height: 100vh;
      border-radius: 0;
    }
  }
</style>
