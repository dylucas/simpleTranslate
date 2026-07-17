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
  import { configStore } from "./store";

  export let show = false;
  export let isDark = true; // 接收父组件传入的主题状态

  let saving = false;
  let message = "";
  let config;
  // 连接测试状态：{ [engine]: { testing, ok, msg } }
  let connStatus = {};

  // 每次打开同步 Store 数据（使用 $configStore 自动订阅，避免手动 subscribe 带来的取消订阅问题）
  $: if (show) {
    config = JSON.parse(JSON.stringify($configStore));
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
      on:click|stopPropagation
      transition:fly={{ y: 20, duration: 300 }}
    >
      <header class="modal-header">
        <div class="header-main">
          <div class="brand-icon">
            <Settings size={22} strokeWidth={2.5} />
          </div>
          <div class="header-info">
            <h3>偏好设置</h3>
            <span>管理您的 API 凭据和翻译服务</span>
          </div>
        </div>
        <button class="close-btn" on:click={() => (show = false)}>
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
                  <select bind:value={config.defaultEngine}>
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
        <div class="footer-status">{message}</div>
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
    background: rgba(0, 0, 0, 0.55);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  /* 浅色模式下使用更轻的遮罩 */
  .overlay.light-mode {
    background: rgba(15, 23, 42, 0.35);
  }

  .modal {
    width: 580px;
    max-width: 95vw;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 24px;
    box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  /* Header */
  .modal-header {
    padding: 24px 32px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border);
  }

  .header-main {
    display: flex;
    gap: 16px;
    align-items: center;
  }

  .brand-icon {
    background: var(--primary);
    color: white;
    width: 42px;
    height: 42px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 12px;
    box-shadow: 0 4px 12px var(--accent-glow);
  }

  .header-info h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--text-main);
  }

  .header-info span {
    font-size: 12px;
    color: var(--text-sec);
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    cursor: pointer;
    padding: 8px;
    border-radius: 50%;
    transition: 0.2s;
  }

  .close-btn:hover {
    background: var(--bg-surface);
    color: var(--text-main);
  }

  /* Content */
  .modal-content {
    padding: 24px 32px;
    max-height: 65vh;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 28px;
  }

  .group-title {
    font-size: 11px;
    font-weight: 700;
    color: var(--primary);
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .settings-card,
  .input-card {
    background: var(--bg-surface);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 20px;
  }

  .setting-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .item-label {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .main-label {
    font-size: 14px;
    font-weight: 500;
    color: var(--text-main);
  }

  .sub-label {
    font-size: 12px;
    color: var(--text-sec);
  }

  /* Forms */
  .input-card {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .input-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .input-field {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .input-field label {
    font-size: 12px;
    font-weight: 500;
    color: var(--text-sec);
    margin-left: 2px;
  }

  .input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }

  .input-wrapper :global(svg) {
    position: absolute;
    left: 12px;
    color: var(--text-sec);
    pointer-events: none;
  }

  input,
  select {
    width: 100%;
    background: var(--bg-input);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 10px 12px 10px 36px;
    color: var(--text-main);
    font-size: 14px;
    transition: all 0.2s;
    outline: none;
  }

  select {
    padding-left: 12px;
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
    gap: 10px;
    flex-wrap: wrap;
  }
  .test-btn {
    background: var(--bg-input);
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: 8px 14px;
    border-radius: 10px;
    font-size: 13px;
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    transition: all 0.2s;
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
    gap: 4px;
    font-size: 12px;
    font-weight: 500;
  }
  .test-msg.ok {
    color: #10b981;
  }
  .test-msg.err {
    color: #ef4444;
    max-width: 280px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Footer */
  .modal-footer {
    padding: 20px 32px 32px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    background: var(--bg-elevated);
  }

  .footer-status {
    font-size: 13px;
    color: var(--primary);
    font-weight: 500;
  }

  .footer-actions {
    display: flex;
    gap: 12px;
  }

  .primary-btn {
    background: var(--primary);
    color: white;
    border: none;
    padding: 10px 20px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
    transition: 0.2s;
  }

  .primary-btn:hover {
    filter: brightness(1.1);
    transform: translateY(-1px);
  }

  .primary-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .secondary-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: 10px 20px;
    border-radius: 10px;
    font-size: 14px;
    cursor: pointer;
    transition: 0.2s;
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

  /* Custom Scrollbar */
  .modal-content::-webkit-scrollbar {
    width: 5px;
  }
  .modal-content::-webkit-scrollbar-track {
    background: transparent;
  }
  .modal-content::-webkit-scrollbar-thumb {
    background: var(--border);
    border-radius: 10px;
  }
</style>
