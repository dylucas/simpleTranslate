<script>
// @ts-nocheck

  import { TranslateText, TranslateMulti, GetConfig, LoadHistory, SaveHistory } from "../wailsjs/go/main/App";
  import {
    Languages,
    ArrowLeftRight,
    History as HistoryIcon,
    Sun,
    Moon,
    Copy,
    Check,
    X,
    Settings,
    PanelLeftClose, // 使用更语义化的图标
    Volume2,
    Square,
    Zap,
    AlertCircle,
    Clipboard,
    CornerDownLeft,
  } from "lucide-svelte";
  import Config from "./lib/Config.svelte";
  import History from "./lib/History.svelte";
  // @ts-ignore
  import { onMount, onDestroy } from "svelte";
  // @ts-ignore
  import { fade } from "svelte/transition";
  import { configStore, initConfig, updateAndSaveConfig } from "./lib/store";
  // @ts-ignore
  import { ClipboardGetText } from "../wailsjs/runtime/runtime";

  // --- 状态控制增强 ---
  let isProcessing = false; // 并发锁
  let translateSeq = 0; // 请求序列号，用于丢弃过期响应
  let lastTranslatedInput = ""; // 上次发起翻译的输入，用于判断响应是否过期
  let pendingRetry = false; // 请求期间输入又变化，需重试
  let speakingText = null; // 当前正在朗读的文本
  let autoTranslate = true; // 自动翻译（防抖）
  let suppressAuto = false; // 程序化修改 input 时抑制自动翻译
  let autoTimer = null; // 防抖计时器
  let historySaveTimer = null; // 历史保存防抖计时器
  let errorToast = null; // { msg, ts } 错误提示
  let clipboardWatch = false; // 剪贴板监听开关
  let lastClipboardText = ""; // 上次剪贴板内容，用于变化检测
  let clipboardTimer = null; // 剪贴板轮询计时器

  // --- 核心状态 ---
  let input = "";
  let output = "";
  let compareOutputs = {}; // { [engine]: { text, error } }
  let compareMode = false;
  let compareEngines = ["tencent", "aliyun"];
  let source = "auto";
  let target = "zh";
  let lastDetectedLang = ""; // 后端识别出的原始语种代码（如 "en"），供 retranslateOutput 在 auto 模式下使用
  let status = "准备就绪";
  let copied = false;
  let copiedEngines = {}; // { [engine]: boolean } 跟踪每个引擎的复制状态

  // 错误提示：4 秒后自动消失
  function showError(msg) {
    const text = String(msg || "翻译失败").replace(/^Error:\s*/, "");
    errorToast = { msg: text, ts: Date.now() };
    setTimeout(() => {
      if (errorToast && errorToast.msg === text) errorToast = null;
    }, 4000);
  }

  // --- 界面控制 ---
  let showConfig = false;
  let showHistory = false;
  let history = [];
  let inputEl; // 用于聚焦输入框的引用
  let autoDetectLang = "自动识别";

  const langs = {
    zh: "中文",
    en: "英文",
    jp: "日语",
    kr: "韩语",
    fr: "法语",
    de: "德语",
    ru: "俄语",
    es: "西语",
  };

  const langMap = {
    zh: "zh-CN",
    en: "en-US",
    jp: "ja-JP",
    kr: "ko-KR",
    fr: "fr-FR",
    de: "de-DE",
    ru: "ru-RU",
    es: "es-ES",
  };

  function getSpeechLang(code) {
    return langMap[code] || "en-US";
  }

  function handleSpeak(text, langCode) {
    if (!text) return;

    if (speakingText === text) {
      // 如果点击的是当前正在朗读的文本，则停止
      window.speechSynthesis.cancel();
      speakingText = null;
      return;
    }

    // 停止之前的朗读
    window.speechSynthesis.cancel();

    const u = new SpeechSynthesisUtterance(text);
    u.lang = getSpeechLang(langCode);
    u.onend = () => {
      speakingText = null;
    };
    u.onerror = () => {
      speakingText = null;
    };
    
    speakingText = text;
    window.speechSynthesis.speak(u);
  }

  // 历史记录变更时防抖同步到后端（避免高频自动翻译时频繁写盘）
  function scheduleHistorySave() {
    if (history === undefined) return;
    clearTimeout(historySaveTimer);
    historySaveTimer = setTimeout(() => {
      SaveHistory(history).catch((e) => {
        console.error("保存历史失败:", e);
        showError("保存历史失败");
      });
    }, 500);
  }

  $: if (history !== undefined) {
    scheduleHistorySave();
  }

  // 从 Store 响应式获取配置（使用 $configStore 自动订阅，组件销毁时自动取消，避免内存泄漏）
  $: isDark = $configStore?.isDark ?? true;
  $: sidebarCollapsed = $configStore?.sidebarCollapsed ?? false;
  $: activeEngine = $configStore?.defaultEngine || "tencent";
  $: compareMode = !!($configStore?.compareMode ?? false);
  $: compareEngines = Array.isArray($configStore?.compareEngines) && $configStore.compareEngines.length
    ? $configStore.compareEngines
    : ["tencent", "aliyun"];

  // 检测当前激活引擎是否缺少凭据，用于在 UI 上提示用户
  $: apiKeyMissing = (() => {
    const cfg = $configStore;
    if (!cfg) return false;
    if (activeEngine === "aliyun") {
      return !cfg.aliyun?.secretId || !cfg.aliyun?.secretKey;
    }
    return !cfg.tencent?.secretKey;
  })();

  // 剪贴板监听开关同步自配置
  $: clipboardWatch = !!($configStore?.clipboardWatch ?? false);

  // 响应式启停剪贴板轮询
  $: if (clipboardWatch !== undefined) {
    updateClipboardWatcher(clipboardWatch);
  }

  function updateClipboardWatcher(on) {
    if (on && !clipboardTimer) {
      startClipboardWatch();
    } else if (!on && clipboardTimer) {
      stopClipboardWatch();
    }
  }

  function startClipboardWatch() {
    // 先记录当前剪贴板内容作为基线，避免开启即触发
    ClipboardGetText()
      .then((text) => {
        lastClipboardText = text || "";
      })
      .catch(() => {});
    clipboardTimer = setInterval(async () => {
      try {
        const text = await ClipboardGetText();
        if (
          text &&
          text !== lastClipboardText &&
          text.trim().length > 0 &&
          text.length < 5000 && // 防止超大文本触发
          !isProcessing
        ) {
          lastClipboardText = text;
          suppressAuto = true;
          input = text;
          // 立即触发翻译
          setTimeout(() => {
            suppressAuto = false;
            translate();
          }, 50);
        }
      } catch (e) {
        // 读取失败静默忽略，下次重试
      }
    }, 1500);
  }

  function stopClipboardWatch() {
    if (clipboardTimer) {
      clearInterval(clipboardTimer);
      clipboardTimer = null;
    }
  }

  // 切换剪贴板监听
  function toggleClipboardWatch() {
    updateAndSaveConfig("clipboardWatch", !clipboardWatch).catch((e) =>
      showError("切换剪贴板监听失败")
    );
  }

  // 结果再翻译：把当前输出作为新输入，交换源/目标语言后重新翻译
  function retranslateOutput() {
    if (!output || isProcessing) return;
    // output 是原 target 语言的文本：新源 = 原 target，新目标 = 原 source
    // 原 source 为 auto 时，用后端识别出的真实语种（lastDetectedLang）作为新目标，
    // 兜底为 zh（兼容识别失败或未翻译过的场景）
    const origSource = source;
    const origTarget = target;
    let newSource = langs[origTarget] ? origTarget : "auto";
    let newTarget = origSource === "auto"
      ? (lastDetectedLang || "zh")
      : origSource;
    // 新源与新目标相同时，按习惯切换为反方向（zh↔en）
    if (newTarget === newSource) {
      newTarget = newSource === "zh" ? "en" : "zh";
    }
    // 启用再翻译后，原识别结果失效，清空避免下次误用
    lastDetectedLang = "";
    suppressAuto = true;
    input = output;
    output = "";
    source = newSource;
    target = newTarget;
    setTimeout(() => {
      suppressAuto = false;
      translate();
    }, 50);
  }

  // 切换主题的函数
  function toggleTheme() {
    updateAndSaveConfig("isDark", !$configStore.isDark);
  }

  // 切换侧边栏并保存状态
  function toggleSidebar() {
    updateAndSaveConfig("sidebarCollapsed", !sidebarCollapsed);
  }

  onMount(async () => {
    await initConfig();
    // 从后端加载持久化历史记录
    try {
      const saved = await LoadHistory();
      if (Array.isArray(saved)) history = saved;
    } catch (e) {
      console.error("加载历史失败:", e);
    }
  });

  onDestroy(() => {
    stopClipboardWatch();
    clearTimeout(historySaveTimer);
    clearTimeout(autoTimer);
  });

  // 自动翻译防抖：输入变化后延迟触发（可开关）
  $: if (autoTranslate && !suppressAuto) {
    handleAutoTranslate(input);
  }

  function handleAutoTranslate(val) {
    clearTimeout(autoTimer);
    if (!val || !val.trim() || val.trim().length < 1) return;
    autoTimer = setTimeout(() => {
      if (val === input) translate();
    }, 700);
  }

  async function translate() {
    if (!input.trim()) return;
    // 已有请求在途：标记需重试，让在途请求结束后重新触发
    if (isProcessing) {
      pendingRetry = true;
      return;
    }
    isProcessing = true;
    status = "翻译中...";
    // 捕获本次请求的快照，用于响应返回时判断是否已过期
    const seq = ++translateSeq;
    const reqInput = input;
    const reqSource = source;
    const reqTarget = target;
    const reqCompare = compareMode;
    lastTranslatedInput = reqInput;
    pendingRetry = false;
    try {
      let res;
      const engines = reqCompare
        ? (Array.isArray(compareEngines) ? compareEngines : ["tencent", "aliyun"])
        : [];
      if (reqCompare) {
        res = await TranslateMulti(reqInput, reqSource, reqTarget, engines);
      } else {
        res = await TranslateText(reqInput, reqSource, reqTarget, activeEngine);
      }
      // 丢弃过期响应：若期间用户又改了输入，结果不再应用
      if (seq !== translateSeq) return;

      if (reqCompare) {
        compareOutputs = res.results || {};
        const preferredEngine = activeEngine || engines[0];
        output = compareOutputs?.[preferredEngine]?.text || "";
      } else {
        output = res.text;
        compareOutputs = {};
      }
      if (reqSource === "auto") {
        lastDetectedLang = res.autoSrc || "";
        let detected = langs[res.autoSrc] || res.autoSrc;
        autoDetectLang = `自动 (${detected})`;
      }
      // 仅当用户未在请求期间手动改 target 时，才同步后端兜底后的 target
      if (reqTarget === target) {
        target = res.target || target;
      }
      status = "完成";
      addHistory(reqInput, output, reqSource, target);
    } catch (e) {
      if (seq !== translateSeq) return;
      status = "翻译失败";
      showError(e);
      console.error(e);
    } finally {
      isProcessing = false;
      // 请求期间输入又变化，重新触发一次翻译
      if (pendingRetry && input.trim() && input !== lastTranslatedInput) {
        pendingRetry = false;
        translate();
      }
    }
  }

  function addHistory(input, output, src, tgt) {
    // 去重：与最近一条相同输入+语种则跳过
    if (
      history.length > 0 &&
      history[0].input === input &&
      history[0].source === src &&
      history[0].target === tgt
    ) {
      return;
    }
    const entry = {
      id: Date.now(),
      input,
      output,
      source: src,
      target: tgt,
      time: new Date().toLocaleString([], {
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
      }),
    };
    history = [entry, ...history].slice(0, 200);
  }

  function clearHistory() {
    history = []; // 清空历史记录
    SaveHistory([]).catch((e) => console.error("清空历史失败:", e));
  }

  // 切换对照引擎选择（至少保留一个）
  function toggleCompareEngine(eng) {
    let next = Array.isArray(compareEngines) ? [...compareEngines] : [];
    if (next.includes(eng)) {
      if (next.length <= 1) return; // 至少保留一个
      next = next.filter((e) => e !== eng);
    } else {
      next.push(eng);
    }
    updateAndSaveConfig("compareEngines", next);
    // 切换后清空旧结果，触发重新翻译
    compareOutputs = {};
  }

  function handleCopy() {
    const textToCopy = output;
    if (!textToCopy) return;
    navigator.clipboard.writeText(textToCopy);
    copied = true;
    setTimeout(() => (copied = false), 2000);
  }

  function handleCopyEngine(engine) {
    const textToCopy = compareOutputs?.[engine]?.text;
    if (!textToCopy) return;
    navigator.clipboard.writeText(textToCopy);
    copiedEngines[engine] = true;
    setTimeout(() => {
      copiedEngines[engine] = false;
      copiedEngines = { ...copiedEngines }; // 触发响应式更新
    }, 2000);
  }

  /**
   * @param {{ ctrlKey: any; metaKey: any; key: string; preventDefault: () => void; }} e
   */
  function handleGlobalKeydown(e) {
    // 发送翻译：Ctrl/Cmd + Enter
    if ((e.ctrlKey || e.metaKey) && e.key === "Enter") {
      e.preventDefault();
      translate();
      return;
    }

    // 聚焦输入：Ctrl/Cmd + L
    if ((e.ctrlKey || e.metaKey) && (e.key === "l" || e.key === "L")) {
      e.preventDefault();
      if (inputEl) inputEl.focus();
      return;
    }

    // 清空输入：Ctrl/Cmd + K
    if ((e.ctrlKey || e.metaKey) && (e.key === "k" || e.key === "K")) {
      e.preventDefault();
      input = "";
      return;
    }

    // 交换语言：Ctrl/Cmd + J
    if ((e.ctrlKey || e.metaKey) && (e.key === "j" || e.key === "J")) {
      e.preventDefault();
      [source, target] = [target, source];
      return;
    }

    // 切换历史面板：Ctrl/Cmd + Shift + H
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && (e.key === "h" || e.key === "H")) {
      e.preventDefault();
      showHistory = !showHistory;
      return;
    }

    // 切换主题：Ctrl/Cmd + M
    if ((e.ctrlKey || e.metaKey) && (e.key === "m" || e.key === "M")) {
      e.preventDefault();
      updateAndSaveConfig("isDark", !$configStore.isDark);
      return;
    }

    // 关闭面板：Esc
    if (e.key === "Escape") {
      if (showHistory) showHistory = false;
    }
  }

  function handleHistorySelect(event) {
    const item = event.detail;
    suppressAuto = true; // 选择历史时抑制自动翻译
    input = item.input;
    output = item.output;
    // 还原语言对，让用户看到原翻译上下文
    if (item.source) source = item.source;
    if (item.target) target = item.target;
    showHistory = false;
    setTimeout(() => (suppressAuto = false), 50);
  }

  function handleHistoryClose() {
    showHistory = false;
  }

  function handleHistoryClear() {
    clearHistory();
  }

  // 监视配置窗口的关闭
  $: if (!showConfig) {
    refreshConfig();
  }

  // 抽离出更新配置的逻辑：关闭设置面板时重新同步整个配置到 store，
  // 确保所有派生状态（主题、引擎、对照、剪贴板监听等）保持一致
  async function refreshConfig() {
    try {
      const cfg = await GetConfig();
      if (cfg) {
        configStore.set(cfg);
      }
    } catch (e) {
      console.error("读取配置失败:", e);
    }
  }
</script>

<svelte:window on:keydown={handleGlobalKeydown} />

<div class="app-shell" class:light-mode={!isDark}>
  {#if errorToast}
    <div class="error-toast" transition:fade={{ duration: 200 }}>
      <AlertCircle size={16} />
      <span class="error-toast-msg">{errorToast.msg}</span>
      <button class="error-toast-retry" on:click={() => { errorToast = null; translate(); }}>
        重试
      </button>
      <button class="error-toast-close" on:click={() => (errorToast = null)}>
        <X size={14} />
      </button>
    </div>
  {/if}
  <aside class="sidebar" class:collapsed={sidebarCollapsed}>
    <div class="sidebar-header">
      {#if !sidebarCollapsed}
        <div class="brand" transition:fade={{ duration: 150 }}>
          <div class="brand-icon"><Languages size={22} /></div>
          <span>Translate</span>
        </div>
      {/if}

      <button
        class="collapse-toggle"
        class:centered={sidebarCollapsed}
        on:click={toggleSidebar}
        title={sidebarCollapsed ? "展开" : "收起"}
      >
        <div class="icon-wrapper" class:rotated={sidebarCollapsed}>
          <PanelLeftClose size={18} />
        </div>
      </button>
    </div>

    <nav class="side-nav">
      <button
        class="nav-item"
        on:click={() => (showHistory = true)}
        title="历史记录"
      >
        <div class="nav-icon"><HistoryIcon size={20} /></div>
        {#if !sidebarCollapsed}
          <span class="nav-text" transition:fade={{ duration: 100 }}
            >历史记录</span
          >
        {/if}
      </button>
    </nav>

    <div class="sidebar-footer">
      {#if !sidebarCollapsed}
        <div class="engine-box" transition:fade={{ duration: 100 }}>
          <!-- svelte-ignore a11y-label-has-associated-control -->
          <label>翻译引擎</label>
          <div class="engine-pills">
            <button
              class:active={activeEngine === "tencent"}
              on:click={() => updateAndSaveConfig("defaultEngine", "tencent")}
              >混元</button
            >
            <button
              class:active={activeEngine === "aliyun"}
              on:click={() => updateAndSaveConfig("defaultEngine", "aliyun")}
              >阿里</button
            >
          </div>
        </div>
      {/if}

      <div class="bottom-tools" class:column-layout={sidebarCollapsed}>
        <button
          class="tool-btn"
          on:click={() => updateAndSaveConfig("isDark", !isDark)}
          title="切换主题"
        >
          {#if isDark}<Sun size={18} />{:else}<Moon size={18} />{/if}
        </button>
        <button
          class="tool-btn"
          on:click={() => (showConfig = true)}
          title="设置"
        >
          <Settings size={18} />
        </button>
      </div>
    </div>
  </aside>

  <main class="main-content">
    {#if apiKeyMissing}
      <div class="api-key-banner" transition:fade={{ duration: 200 }}>
        <AlertCircle size={14} />
        <span>
          当前引擎（{activeEngine === "aliyun" ? "阿里云" : "混元"}）未配置凭据，
        </span>
        <button class="banner-link" on:click={() => (showConfig = true)}>
          前往设置
        </button>
      </div>
    {/if}
    <header class="workspace-header">
      <div class="lang-bar">
        <div class="select-wrapper">
          <select
            bind:value={source}
          >
            <option value="auto">{autoDetectLang}</option>
            {#each Object.entries(langs) as [code, name]}
              <option value={code}>{name}</option>
            {/each}
          </select>
        </div>

        <button
          class="swap-btn"
          on:click={() => ([source, target] = [target, source])}
          title="交换语言"
        >
          <ArrowLeftRight size={16} />
        </button>

        <div class="select-wrapper">
          <select
            bind:value={target}
          >
            {#each Object.entries(langs) as [code, name]}
              <option value={code}>{name}</option>
            {/each}
          </select>
        </div>
      </div>

      <div class="right-tools">
        <button
          class="mode-btn"
          class:active={autoTranslate}
          on:click={() => (autoTranslate = !autoTranslate)}
          title={autoTranslate ? "关闭自动翻译" : "开启自动翻译"}
        >
          <Zap size={13} />
          自动
        </button>
        <button
          class="mode-btn"
          class:active={clipboardWatch}
          on:click={toggleClipboardWatch}
          title={clipboardWatch ? "关闭剪贴板监听" : "开启剪贴板监听"}
        >
          <Clipboard size={13} />
          剪贴板
        </button>
        <button
          class="mode-btn"
          class:active={compareMode}
          on:click={() => updateAndSaveConfig("compareMode", !compareMode)}
          title="多引擎对照"
        >
          对照
        </button>
        <button
          class="translate-btn"
          on:click={translate}
          disabled={status === "翻译中..."}
        >
          <span>{status === "翻译中..." ? "翻译中" : "翻译"}</span>
          {#if status === "翻译中..."}
            <span class="loading-dots">...</span>
          {/if}
        </button>
      </div>
    </header>

    <div class="editor-container">
      <section class="editor-pane source">
        <textarea
          bind:this={inputEl}
          bind:value={input}
          placeholder="在此输入要翻译的文本..."
          spellcheck="false"
        ></textarea>
        <div class="pane-footer">
          <span class="char-count">{input.length} 字符</span>
          {#if input}
            <button
              class="clear-btn"
              on:click={() => handleSpeak(input, source === "auto" ? "en" : source)}
              title="朗读"
            >
              {#if speakingText === input}
                <Square size={12} fill="currentColor" />
              {:else}
                <Volume2 size={12} />
              {/if}
            </button>
            <button class="clear-btn" on:click={() => (input = "")}
              ><X size={12} /> 清空</button
            >
          {/if}
        </div>
      </section>

      <section class="editor-pane result" class:compare-mode={compareMode}>
        {#if compareMode}
          <div class="compare-engine-bar">
            <span class="engine-bar-label">对照引擎</span>
            <div class="engine-toggle-pills">
              {#each ["tencent", "aliyun"] as eng}
                <button
                  class="toggle-pill"
                  class:active={(compareEngines || []).includes(eng)}
                  on:click={() => toggleCompareEngine(eng)}
                >
                  {eng === "tencent" ? "混元" : "阿里"}
                </button>
              {/each}
            </div>
          </div>
          <div class="compare-grid">
            {#each (compareEngines || []) as eng}
              <div class="compare-card" class:refreshing={isProcessing}>
                <div class="compare-header">
                  <span class="compare-title">{eng === "tencent" ? "混元" : "阿里"}</span>
                  <div class="compare-header-right">
                    {#if isProcessing && compareOutputs?.[eng]?.text && !compareOutputs?.[eng]?.error}
                      <span class="compare-refreshing" title="正在更新">刷新中</span>
                    {/if}
                    {#if compareOutputs?.[eng]?.error}
                      <span class="compare-error">失败</span>
                    {/if}
                    <button
                      class="compare-copy-btn"
                      class:active={speakingText === compareOutputs?.[eng]?.text}
                      class:disabled={!compareOutputs?.[eng]?.text ||
                        compareOutputs?.[eng]?.error}
                      on:click={() =>
                        handleSpeak(compareOutputs?.[eng]?.text, target)}
                      title="朗读"
                    >
                      {#if speakingText === compareOutputs?.[eng]?.text}
                        <Square size={12} fill="currentColor" />
                      {:else}
                        <Volume2 size={12} />
                      {/if}
                    </button>
                    <button
                      class="compare-copy-btn"
                      class:success={copiedEngines[eng]}
                      class:disabled={!compareOutputs?.[eng]?.text || compareOutputs?.[eng]?.error}
                      on:click={() => handleCopyEngine(eng)}
                      title="复制此结果"
                    >
                      {#if copiedEngines[eng]}
                        <Check size={12} />
                      {:else}
                        <Copy size={12} />
                      {/if}
                    </button>
                  </div>
                </div>
                {#if isProcessing && !compareOutputs?.[eng]?.text && !compareOutputs?.[eng]?.error}
                  <div class="skeleton-line"></div>
                {:else}
                  <textarea
                    readonly
                    value={compareOutputs?.[eng]?.text || (compareOutputs?.[eng]?.error ? `错误：${compareOutputs[eng].error}` : "")}
                    placeholder="翻译结果..."
                    spellcheck="false"
                  ></textarea>
                {/if}
              </div>
            {:else}
              <div class="empty-compare">请至少选择一个对照引擎</div>
            {/each}
          </div>
        {:else}
          {#if isProcessing && !output}
            <div class="skeleton-block">
              <div class="skeleton-line"></div>
              <div class="skeleton-line short"></div>
              <div class="skeleton-line"></div>
            </div>
          {:else}
            <textarea
              readonly
              value={output}
              placeholder="翻译结果..."
              spellcheck="false"
            ></textarea>
          {/if}
        {/if}
        <div class="pane-footer">
          {#if !compareMode && output}
            <button
              class="action-btn"
              on:click={() => handleSpeak(output, target)}
              title="朗读"
            >
              {#if speakingText === output}
                <Square size={14} fill="currentColor" />
              {:else}
                <Volume2 size={14} />
              {/if}
              朗读
            </button>
            <button
              class="action-btn"
              on:click={retranslateOutput}
              title="将翻译结果作为新输入，交换语言后重新翻译"
            >
              <CornerDownLeft size={14} />
              再翻译
            </button>
            <button
              class="action-btn copy"
              on:click={handleCopy}
              class:success={copied}
            >
              {#if copied}<Check size={14} />{:else}<Copy size={14} />{/if}
              {copied ? "已复制" : "复制"}
            </button>
          {/if}
        </div>
      </section>
    </div>

    <footer class="app-status-bar">
      <div class="status-item">
        <span
          class="status-dot"
          class:processing={status === "翻译中..."}
          class:done={status === "完成"}
          class:error={status === "翻译失败"}
        ></span>
        {status}
      </div>
      <div class="status-item shortcut-hint">
        <span>Ctrl+Enter 发送 · Ctrl+J 交换 · Ctrl+Shift+H 历史</span>
      </div>
    </footer>
  </main>

  <History
    bind:show={showHistory}
    {history}
    on:select={handleHistorySelect}
    on:close={handleHistoryClose}
    on:clear={handleHistoryClear}
  />

  <Config bind:show={showConfig} {isDark} />
</div>

<style>
  /* 错误 Toast */
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
  .error-toast-retry {
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
  .error-toast-retry:hover {
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

  /* API Key 缺失提示横幅 */
  .api-key-banner {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    background: rgba(251, 191, 36, 0.12);
    border-bottom: 1px solid rgba(251, 191, 36, 0.3);
    color: var(--warning);
    font-size: 12px;
    font-weight: 500;
  }
  .light-mode .api-key-banner {
    background: rgba(251, 191, 36, 0.1);
  }
  .banner-link {
    background: transparent;
    border: none;
    color: var(--warning);
    text-decoration: underline;
    cursor: pointer;
    font-size: 12px;
    font-weight: 600;
    padding: 0;
  }
  .banner-link:hover {
    color: #d97706;
  }

  .right-tools {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .mode-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-sec);
    padding: 8px 12px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }
  .mode-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .mode-btn.active {
    border-color: var(--primary);
    color: var(--primary);
    background: var(--primary-soft);
  }

  .compare-grid {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
    overflow-y: auto;
  }
  .compare-card {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0; /* 允许内部 textarea 撑满剩余高度 */
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 8px 10px;
    background: rgba(0, 0, 0, 0.08);
    overflow: hidden;
    transition: opacity 0.2s;
  }
  .compare-card.refreshing {
    opacity: 0.85;
  }
  .compare-refreshing {
    color: var(--primary);
    font-weight: var(--fw-medium);
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 4px;
    background: var(--primary-bg, rgba(99, 102, 241, 0.12));
    animation: pulse 1.2s ease-in-out infinite;
  }
  @keyframes pulse {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; }
  }
  .compare-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 6px;
    font-size: 12px;
    color: var(--text-sec);
  }
  .compare-header-right {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 40px; /* 预留复制按钮空间，避免出现/消失时抖动 */
    justify-content: flex-end;
  }
  .compare-title {
    font-weight: 700;
    color: var(--text-main);
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .compare-error {
    color: var(--danger);
    font-weight: var(--fw-bold);
  }
  .compare-copy-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-sec);
    padding: 4px 8px;
    border-radius: 6px;
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.2s;
    min-width: 28px;
    height: 24px;
  }
  .compare-copy-btn.disabled {
    visibility: hidden; /* 保留占位，不触发布局抖动 */
    pointer-events: none;
  }
  .compare-copy-btn:hover {
    border-color: var(--text-sec);
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .compare-copy-btn.success {
    border-color: var(--success);
    color: var(--success);
    background: var(--success-soft);
  }
  .compare-copy-btn.active {
    border-color: var(--primary);
    color: var(--primary);
    background: var(--primary-soft);
  }

  .compare-card textarea {
    flex: 1;
    min-height: 0;
    margin-top: 4px;
    padding: 6px 0 0;
    border-top: 1px dashed var(--border);
    font-size: 14px;
  }

  .compare-engine-bar {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 8px;
    flex-shrink: 0;
  }
  .engine-bar-label {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-sec);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
  .engine-toggle-pills {
    display: flex;
    gap: 6px;
  }
  .toggle-pill {
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-sec);
    padding: 4px 12px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }
  .toggle-pill.active {
    border-color: var(--primary);
    color: var(--primary);
    background: var(--primary-soft);
  }
  .toggle-pill:hover {
    color: var(--text-main);
  }
  .empty-compare {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-sec);
    font-size: 13px;
    opacity: 0.6;
  }

  /* 加载骨架屏 */
  .skeleton-block {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 4px 0;
  }
  .skeleton-line {
    height: 16px;
    border-radius: 6px;
    background: linear-gradient(
      90deg,
      var(--bg-hover) 25%,
      var(--border) 50%,
      var(--bg-hover) 75%
    );
    background-size: 200% 100%;
    animation: shimmer 1.4s ease-in-out infinite;
  }
  .skeleton-line.short {
    width: 60%;
  }
  @keyframes shimmer {
    0% {
      background-position: 200% 0;
    }
    100% {
      background-position: -200% 0;
    }
  }
  /* 设计令牌（:root / .light-mode）与全局重置已迁移至全局 style.css，此处仅消费令牌 */

  .app-shell {
    display: flex;
    height: 100vh;
    background: var(--bg-base);
    color: var(--text-main);
    font-family: var(--font-sans);
    font-size: var(--fs-base);
    overflow: hidden;
  }

  /* --- 侧边栏 --- */
  .sidebar {
    width: var(--sidebar-w);
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    transition: width var(--t-slow) var(--ease-standard);
    z-index: var(--z-sidebar);
    flex-shrink: 0;
    backdrop-filter: blur(10px);
    -webkit-backdrop-filter: blur(10px);
  }

  .sidebar.collapsed {
    width: var(--sidebar-collapsed-w);
  }

  .sidebar-header {
    height: 64px;
    display: flex;
    align-items: center;
    justify-content: space-between; /* 默认两端对齐 */
    padding: 0 16px;
    position: relative;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: var(--sp-2);
    font-weight: var(--fw-bold);
    font-size: var(--fs-lg);
    letter-spacing: var(--tracking-tight);
    color: var(--text-main);
    white-space: nowrap;
    overflow: hidden;
  }
  .brand-icon {
    width: 34px;
    height: 34px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: var(--radius-md);
    background: var(--accent-grad);
    color: var(--text-inverse);
    box-shadow: var(--shadow-glow);
    flex-shrink: 0;
  }

  /* 切换按钮样式 */
  .collapse-toggle {
    background: transparent;
    border: none;
    color: var(--text-sec);
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
  }
  .collapse-toggle:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  /* 当侧边栏折叠时，按钮居中显示 */
  .collapse-toggle.centered {
    margin: 0 auto;
    width: 100%;
  }

  .icon-wrapper {
    display: flex;
    transition: transform var(--t-slow) var(--ease-spring);
  }
  /* 图标旋转动画：利用 CSS Transform 翻转 */
  .icon-wrapper.rotated {
    transform: rotate(180deg);
  }

  .side-nav {
    flex: 1;
    padding: var(--sp-4) var(--sp-3);
    display: flex;
    flex-direction: column;
    gap: var(--sp-1);
  }

  .nav-item {
    display: flex;
    align-items: center;
    padding: var(--sp-2) var(--sp-3);
    background: transparent;
    border: none;
    border-radius: var(--radius-md);
    color: var(--text-sec);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
    height: 44px;
    width: 100%;
    text-align: left;
  }
  .nav-item:hover {
    background: var(--bg-hover);
    color: var(--text-main);
  }
  .nav-item:active {
    background: var(--primary-soft);
    color: var(--primary);
  }
  .nav-icon {
    display: flex;
    justify-content: center;
    align-items: center;
    min-width: 24px;
  }
  .nav-text {
    margin-left: var(--sp-3);
    font-size: var(--fs-md);
    font-weight: var(--fw-medium);
    white-space: nowrap;
  }

  .sidebar-footer {
    border-top: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    gap: var(--sp-4);
    padding: var(--sp-4) var(--sp-3);
    transition: padding var(--t-slow) var(--ease-standard);
  }

  .engine-box label {
    font-size: var(--fs-xs);
    font-weight: var(--fw-semibold);
    color: var(--text-sec);
    margin-bottom: var(--sp-2);
    display: block;
    text-transform: uppercase;
    letter-spacing: var(--tracking-wider);
  }
  .engine-pills {
    display: flex;
    background: var(--bg-hover);
    border-radius: var(--radius-sm);
    padding: 3px;
  }
  .engine-pills button {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--text-sec);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    padding: var(--sp-1) var(--sp-2);
    border-radius: var(--radius-xs);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
  }
  .engine-pills button:hover {
    color: var(--text-main);
  }
  .engine-pills button.active {
    background: var(--bg-surface);
    color: var(--text-main);
    box-shadow: var(--shadow-sm);
    font-weight: var(--fw-semibold);
  }

  .bottom-tools {
    display: flex;
    gap: var(--sp-2);
    justify-content: space-between;
    transition: all var(--t-slow) var(--ease-standard);
  }

  /* 引擎选择框在折叠时的过渡 */
  .engine-box {
    overflow: hidden;
    transition:
      max-height var(--t-slow) var(--ease-standard),
      opacity var(--t-base) var(--ease-standard);
  }

  .sidebar.collapsed .sidebar-footer {
    padding: var(--sp-4) 0;
  }

  /* 当侧边栏折叠时，改变工具栏布局 */
  .sidebar.collapsed .bottom-tools {
    flex-direction: column;
    align-items: center;
    gap: var(--sp-3);
  }

  .tool-btn {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-sec);
    padding: var(--sp-2);
    border-radius: var(--radius-md);
    cursor: pointer;
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    transition: all var(--t-base) var(--ease-standard);
    min-width: 0;
  }

  .sidebar.collapsed .tool-btn {
    width: 40px;
    height: 40px;
    flex: none;
  }
  .tool-btn:hover {
    background: var(--bg-hover);
    color: var(--text-main);
    border-color: var(--border);
  }
  .tool-btn:active {
    transform: scale(0.94);
  }

  /* --- 主内容 --- */
  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    background: var(--bg-surface);
    position: relative;
    z-index: 1;
  }

  .workspace-header {
    height: var(--header-h);
    padding: 0 var(--sp-6);
    display: flex;
    align-items: center;
    justify-content: space-between;
    border-bottom: 1px solid var(--border);
    gap: var(--sp-4);
  }

  .lang-bar {
    display: flex;
    align-items: center;
    gap: var(--sp-3);
  }

  .select-wrapper {
    position: relative;
    display: inline-flex;
    align-items: center;
  }
  .select-wrapper select {
    appearance: none;
    -webkit-appearance: none;
    background: transparent;
    border: none;
    color: var(--text-main);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    padding: var(--sp-2) 26px var(--sp-2) var(--sp-3);
    border-radius: var(--radius-sm);
    outline: none;
    transition: background var(--t-base) var(--ease-standard),
      color var(--t-base) var(--ease-standard);
  }
  .select-wrapper select:hover {
    background: var(--bg-hover);
  }
  /* 自定义下拉箭头（替代被 appearance:none 移除的原生指示） */
  .select-wrapper::after {
    content: "";
    position: absolute;
    right: 10px;
    width: 7px;
    height: 7px;
    border-right: 2px solid var(--text-sec);
    border-bottom: 2px solid var(--text-sec);
    transform: rotate(45deg) translateY(-2px);
    pointer-events: none;
    transition: border-color var(--t-base) var(--ease-standard);
  }
  .select-wrapper:hover::after {
    border-color: var(--primary);
  }
  /* 主题化原生选项列表 */
  .select-wrapper select option {
    background: var(--bg-elevated);
    color: var(--text-main);
  }

  .swap-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    padding: var(--sp-2);
    border-radius: var(--radius-full);
    cursor: pointer;
    transition: all var(--t-base) var(--ease-standard);
    display: flex;
  }
  .swap-btn:hover {
    background: var(--bg-hover);
    color: var(--primary);
    transform: rotate(180deg);
  }

  .translate-btn {
    background: var(--accent-grad);
    color: var(--text-inverse);
    border: none;
    padding: var(--sp-2) var(--sp-6);
    border-radius: var(--radius-full);
    font-size: var(--fs-md);
    font-weight: var(--fw-semibold);
    cursor: pointer;
    box-shadow: var(--shadow-glow);
    transition: transform var(--t-fast) var(--ease-standard),
      box-shadow var(--t-base) var(--ease-standard),
      filter var(--t-base) var(--ease-standard);
    display: inline-flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .translate-btn:hover:not(:disabled) {
    transform: translateY(-1px);
    filter: brightness(1.08);
    box-shadow: 0 10px 28px -4px var(--accent-glow);
  }
  .translate-btn:active {
    transform: scale(0.96);
  }
  .translate-btn:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }
  /* 翻译中省略号动画 */
  .loading-dots {
    display: inline-block;
    letter-spacing: 2px;
    animation: dotsPulse 1.2s var(--ease-standard) infinite;
  }
  @keyframes dotsPulse {
    0%, 100% { opacity: 0.35; }
    50% { opacity: 1; }
  }

  .editor-container {
    flex: 1;
    display: flex;
    overflow: hidden;
  }

  .editor-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    padding: var(--sp-6);
    position: relative;
    transition: background var(--t-slow) var(--ease-standard);
    min-width: 0;
  }
  .editor-pane.source {
    border-right: 1px solid var(--border);
    flex: 0.9; /* 略缩小原文区域高度 */
  }
  .editor-pane.result {
    background: var(--bg-base); /* 结果区稍微深一点/浅一点区分 */
    /* 给结果内容更多空间 */
    padding: var(--sp-3) var(--sp-4);
    flex: 1.1; /* 略放大翻译结果区域高度 */
  }

  textarea {
    flex: 1;
    background: transparent;
    border: none;
    resize: none;
    outline: none;
    font-size: var(--fs-xl);
    line-height: var(--lh-relaxed);
    color: var(--text-main);
    padding: 0;
    font-family: inherit;
  }
  textarea::placeholder {
    color: var(--text-sec);
    opacity: 0.5;
  }

  .pane-footer {
    height: 30px;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--sp-2);
    margin-top: var(--sp-2);
  }
  /* 对照模式下隐藏 footer，让结果区域占满空间 */
  .editor-pane.result.compare-mode .pane-footer {
    height: 0;
    margin-top: 0;
    overflow: hidden;
  }
  /* 对照模式下让 compare-grid 占满整个空间 */
  .editor-pane.result.compare-mode {
    padding-bottom: var(--sp-3); /* 保持底部 padding，但移除 footer 空间 */
  }
  .char-count {
    font-size: var(--fs-sm);
    color: var(--text-sec);
    margin-right: auto;
    font-variant-numeric: tabular-nums;
  }

  .clear-btn {
    background: transparent;
    border: none;
    color: var(--text-sec);
    font-size: var(--fs-sm);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    padding: var(--sp-1);
    border-radius: var(--radius-xs);
    transition: color var(--t-base) var(--ease-standard),
      background var(--t-base) var(--ease-standard);
  }
  .clear-btn:hover {
    color: var(--text-main);
    background: var(--bg-hover);
  }

  .action-btn {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-main);
    padding: var(--sp-1) var(--sp-3);
    border-radius: var(--radius-sm);
    font-size: var(--fs-sm);
    font-weight: var(--fw-medium);
    cursor: pointer;
    display: flex;
    align-items: center;
    gap: var(--sp-1);
    transition: all var(--t-base) var(--ease-standard);
  }
  .action-btn:hover {
    border-color: var(--text-sec);
    background: var(--bg-hover);
  }
  .action-btn:active {
    transform: scale(0.96);
  }
  .action-btn.success {
    border-color: var(--success);
    color: var(--success);
    background: var(--success-soft);
  }

  /* 状态栏 */
  .app-status-bar {
    height: var(--statusbar-h);
    border-top: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 var(--sp-4);
    font-size: var(--fs-xs);
    color: var(--text-sec);
    background: var(--bg-sidebar);
  }
  .status-item {
    display: flex;
    align-items: center;
    gap: var(--sp-1);
  }
  .shortcut-hint {
    color: var(--text-muted);
  }
  .shortcut-hint :global(svg) {
    opacity: 0.7;
  }
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--success); /* Ready Green */
    box-shadow: 0 0 0 0 var(--success-soft);
  }
  .status-dot.done {
    background: var(--success);
  }
  .status-dot.processing {
    background: var(--warning);
    animation: pulse 1s infinite;
  }
  .status-dot.error {
    background: var(--danger);
  }

  @keyframes pulse {
    0% {
      opacity: 1;
    }
    50% {
      opacity: 0.4;
    }
    100% {
      opacity: 1;
    }
  }

  /* --- 响应式：窄窗口下紧凑布局 --- */
  @media (max-width: 720px) {
    .workspace-header {
      padding: 0 var(--sp-3);
    }
    .right-tools {
      gap: var(--sp-1);
    }
    .mode-btn {
      padding: var(--sp-1) var(--sp-2);
    }
    .mode-btn :global(svg) {
      display: none;
    }
    .editor-pane {
      padding: var(--sp-3);
    }
    .editor-pane.result {
      padding: var(--sp-2) var(--sp-3);
    }
    textarea {
      font-size: var(--fs-lg);
    }
    .shortcut-hint span {
      display: none;
    }
  }
</style>
