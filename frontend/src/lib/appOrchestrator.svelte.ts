import { tick } from "svelte";
import { get } from "svelte/store";
import { createClipboardReader } from "./clipboard";
import { createConfigController, normalizeConfig, type ConfigController } from "./configController";
import { engineLabel, isEngineConfigured } from "./engines";
import { getSpeechLang } from "./languages";
import { createShortcutHandler } from "./shortcuts";
import { createSpeaker } from "./speech";
import { createTranslateController, type TranslateController, type TranslationState } from "./translateController";
import { MAX_INPUT_BYTES, truncateUtf8, utf8ByteLength } from "./textLimits";
import type { BaiduDomain, CloudConfig, EngineId, HistoryEntry } from "./types";
import type { DesktopBridge } from "./bridge";

export interface AppOrchestratorOptions {
  bridge: DesktopBridge;
  getInputElement: () => HTMLTextAreaElement | undefined;
  getLastPanelTrigger: () => HTMLElement | null;
  setLastPanelTrigger: (value: HTMLElement | null) => void;
}

export interface AppOrchestrator {
  readonly input: string;
  readonly source: string;
  readonly target: string;
  readonly showConfig: boolean;
  readonly showHistory: boolean;
  readonly copied: boolean;
  readonly copiedEngines: Partial<Record<EngineId, boolean>>;
  readonly speakingText: string | null;

  readonly config: CloudConfig;
  readonly configReady: boolean;
  readonly clipboardReadEnabled: boolean;
  readonly availableCompareEngines: EngineId[];
  readonly missingCompareEngines: EngineId[];
  readonly credentialsReady: boolean;
  readonly inputBytes: number;
  readonly canTranslate: boolean;
  readonly unavailableReason: string;
  readonly credentialMessage: string;
  readonly translation: TranslationState;

  readonly configController: ConfigController;
  readonly translateController: TranslateController;

  isConfigured(engine: EngineId): boolean;
  patchConfig<K extends keyof CloudConfig>(key: K, value: CloudConfig[K]): void;
  init(): Promise<void>;
  destroy(): void;
  requestTranslation(): void;
  setLanguage(kind: "source" | "target", value: string): void;
  setBaiduDomain(domain: BaiduDomain): void;
  swapLanguages(): void;
  toggleCompareEngine(engine: EngineId): void;
  speak(text: string, language: string): void;
  copyText(text: string, engine?: EngineId): Promise<void>;
  retranslate(): void;
  focusInputFromShortcut(): void;
  clearInput(): void;
  updateInput(value: string): void;
  toggleTheme(): void;
  openPanel(panel: "config" | "history"): void;
  closePanels(): void;
  restoreHistoryEntry(entry: HistoryEntry): void;
  handleGlobalKeydown(event: KeyboardEvent): void;
  handleWindowFocus(): void;
}

export function createAppOrchestrator(options: AppOrchestratorOptions): AppOrchestrator {
  let input = $state("");
  let source = $state("auto");
  let target = $state("zh");
  let showConfig = $state(false);
  let showHistory = $state(false);
  let suppressAuto = $state(false);
  let skipAutoForInput: string | null = null;
  let copied = $state(false);
  let copiedEngines = $state<Partial<Record<EngineId, boolean>>>({});
  let speakingText = $state<string | null>(null);
  let queuedErrors: string[] = [];
  let controller: TranslateController;
  const copyFeedbackTimers = new Map<EngineId | "output", ReturnType<typeof setTimeout>>();

  const configController = createConfigController(options.bridge, (message) => {
    if (controller) controller.showError(message);
    else queuedErrors.push(message);
  });
  let configState = $state(get(configController));
  $effect(() => configController.subscribe((v) => { configState = v; }));
  let config = $derived(configState.value);
  let configReady = $derived(configState.ready);
  let clipboardReadEnabled = $derived(configReady && config.clipboardWatch);

  $effect(() => {
    if (!configReady) return;
    source = config.sourceLanguage;
    target = config.targetLanguage;
  });

  function persistConfig(operation: Promise<void>): void {
    void operation.catch(() => undefined);
  }

  function patchConfig<K extends keyof CloudConfig>(key: K, value: CloudConfig[K]): void {
    persistConfig(configController.patch(key, value));
  }

  controller = createTranslateController({
    bridge: options.bridge,
    getInput: () => input,
    getSource: () => source,
    getTarget: () => target,
    getActiveEngine: () => config.defaultEngine,
    getBaiduDomain: () => config.baidu.domain,
    getCompareMode: () => config.compareMode,
    getCompareEngines: () => availableCompareEngines,
    appendHistory: (entry) => options.bridge.appendHistory(entry),
    setTarget: (next) => {
      if (target === next) return;
      skipAutoForInput = input;
      target = next;
      persistConfig(configController.patch("targetLanguage", next));
    },
  });
  const translationStore = controller.state;
  let translation = $state(get(translationStore));
  $effect(() => translationStore.subscribe((v) => { translation = v; }));

  function isConfigured(engine: EngineId): boolean {
    return isEngineConfigured(config, engine);
  }

  let availableCompareEngines = $derived(config.compareEngines.filter(isConfigured));
  let missingCompareEngines = $derived(config.compareEngines.filter((engine: EngineId) => !isConfigured(engine)));
  let credentialsReady = $derived(config.compareMode
    ? availableCompareEngines.length > 0
    : isConfigured(config.defaultEngine));
  let inputBytes = $derived(utf8ByteLength(input));
  let canTranslate = $derived(Boolean(input.trim()) && inputBytes <= MAX_INPUT_BYTES && credentialsReady);
  let unavailableReason = $derived(!input.trim()
    ? "请输入文本"
    : inputBytes > MAX_INPUT_BYTES
      ? `原文不能超过 ${MAX_INPUT_BYTES} 个 UTF-8 字节`
    : !credentialsReady
      ? "请先配置翻译凭据"
      : "");
  let credentialMessage = $derived.by(() => {
    if (!config.compareMode && !isConfigured(config.defaultEngine)) {
      return `${engineLabel(config.defaultEngine, true)}尚未配置凭据`;
    }
    if (config.compareMode && missingCompareEngines.length) {
      const names = missingCompareEngines.map((engine: EngineId) => engineLabel(engine)).join("、");
      return availableCompareEngines.length ? `${names}未配置，将仅使用可用引擎` : "所选对照引擎均未配置凭据";
    }
    return "";
  });

  const speaker = createSpeaker(getSpeechLang);
  const clipboardReader = createClipboardReader({
    getText: () => options.bridge.getClipboardText(),
    isBusy: () => translation.isProcessing,
    onText: (text) => {
      suppressAuto = true;
      input = text;
      setTimeout(() => {
        suppressAuto = false;
        requestTranslation();
      }, 0);
    },
  });

  $effect(() => {
    if (clipboardReadEnabled) void clipboardReader.read();
    else clipboardReader.cancel();
  });

  let autoTranslateRoute = $derived.by(() => {
    const domain = config.defaultEngine === "baidu" || (config.compareMode && availableCompareEngines.includes("baidu"))
      ? config.baidu.domain
      : "";
    return `${source}\u0000${target}\u0000${domain}`;
  });

  $effect(() => {
    const value = input;
    const route = autoTranslateRoute;
    if (skipAutoForInput === value) {
      controller.cancelAutoTranslate();
      skipAutoForInput = null;
    } else if (route && config.autoTranslate && !suppressAuto && credentialsReady) {
      controller.handleAutoTranslate(value);
    } else {
      controller.cancelAutoTranslate();
    }
  });

  function openPanel(panel: "config" | "history"): void {
    if (panel === "config" && !configReady) return;
    options.setLastPanelTrigger(document.activeElement instanceof HTMLElement ? document.activeElement : null);
    showConfig = panel === "config";
    showHistory = panel === "history";
  }

  function closePanels(): void {
    showConfig = false;
    showHistory = false;
    void tick().then(() => options.getLastPanelTrigger()?.focus());
  }

  function requestTranslation(): void {
    if (translation.isProcessing) {
      controller.cancel();
      return;
    }
    if (!credentialsReady) {
      controller.showError({ errorCode: "credentials", error: "请先配置可用的翻译凭据" });
      return;
    }
    void controller.translate();
  }

  function setLanguage(kind: "source" | "target", value: string): void {
    if (kind === "source") source = value;
    else target = value;
    persistConfig(configController.patch(kind === "source" ? "sourceLanguage" : "targetLanguage", value));
  }

  function setBaiduDomain(domain: BaiduDomain): void {
    if (domain === config.baidu.domain) return;
    const next = configController.snapshot();
    next.baidu = { ...next.baidu, domain };
    persistConfig(configController.save(next));
  }

  function swapLanguages(): void {
    if (!configReady) return;
    const resolved = source === "auto" ? translation.lastDetectedLang : source;
    if (!resolved) return;
    const nextSource = target;
    const nextTarget = resolved;
    source = nextSource;
    target = nextTarget;
    persistConfig(configController.save({ ...configController.snapshot(), sourceLanguage: nextSource, targetLanguage: nextTarget }));
  }

  function toggleCompareEngine(engine: EngineId): void {
    const selected = [...config.compareEngines];
    const next = selected.includes(engine)
      ? selected.length === 1 ? selected : selected.filter((item) => item !== engine)
      : [...selected, engine];
    persistConfig(configController.patch("compareEngines", next));
  }

  function speak(text: string, language: string): void {
    speaker.speak(text, language, { onChange: (value) => (speakingText = value) });
  }

  function markCopied(engine?: EngineId): void {
    const key = engine ?? "output";
    const previous = copyFeedbackTimers.get(key);
    if (previous) clearTimeout(previous);
    if (engine) copiedEngines[engine] = true;
    else copied = true;
    copyFeedbackTimers.set(key, setTimeout(() => {
      copyFeedbackTimers.delete(key);
      if (engine) copiedEngines[engine] = false;
      else copied = false;
    }, 1600));
  }

  async function copyText(text: string, engine?: EngineId): Promise<void> {
    if (!text) return;
    try {
      await options.bridge.setClipboardText(text);
      clipboardReader.setBaseline(text);
      markCopied(engine);
    } catch {
      controller.showError("复制失败，请重试");
    }
  }

  function retranslate(): void {
    if (!translation.output || translation.isProcessing) return;
    const nextSource = target;
    let nextTarget = source === "auto" ? (translation.lastDetectedLang || "zh") : source;
    if (nextSource === nextTarget) nextTarget = nextSource === "zh" ? "en" : "zh";
    suppressAuto = true;
    input = translation.output;
    controller.setOutput("");
    source = nextSource;
    target = nextTarget;
    persistConfig(configController.save({ ...configController.snapshot(), sourceLanguage: nextSource, targetLanguage: nextTarget }));
    setTimeout(() => {
      suppressAuto = false;
      requestTranslation();
    }, 0);
  }

  function focusInputFromShortcut(): void {
    showConfig = false;
    showHistory = false;
    void tick().then(() => options.getInputElement()?.focus());
  }

  function clearInput(): void {
    input = "";
    controller.clear();
  }

  function updateInput(value: string): void {
    const next = truncateUtf8(value);
    input = next;
    if (next !== value) controller.showError({ errorCode: "invalid_input", error: `原文不能超过 ${MAX_INPUT_BYTES} 个 UTF-8 字节` });
  }

  function toggleTheme(): void {
    if (showConfig) return;
    persistConfig(configController.patch("isDark", !config.isDark));
  }

  const shortcuts = createShortcutHandler({
    onTranslate: requestTranslation,
    onCancel: () => controller.cancel(),
    onFocusInput: focusInputFromShortcut,
    onClearInput: clearInput,
    onSwapLangs: swapLanguages,
    onToggleHistory: () => showHistory ? closePanels() : openPanel("history"),
    onToggleSettings: () => showConfig ? closePanels() : openPanel("config"),
    onToggleTheme: toggleTheme,
    onClosePanel: closePanels,
  }, {
    isPanelOpen: () => showConfig || showHistory,
  });

  function handleGlobalKeydown(event: KeyboardEvent): void {
    shortcuts(event);
  }

  function handleWindowFocus(): void {
    if (clipboardReadEnabled) void clipboardReader.read();
  }

  function restoreHistoryEntry(entry: HistoryEntry): void {
    const route = normalizeConfig({
      sourceLanguage: entry.source,
      targetLanguage: entry.target,
    });
    const restored = {
      ...entry,
      source: route.sourceLanguage,
      target: route.targetLanguage,
    };
    skipAutoForInput = restored.input;
    input = restored.input;
    source = restored.source;
    target = restored.target;
    persistConfig(configController.save({
      ...configController.snapshot(),
      sourceLanguage: restored.source,
      targetLanguage: restored.target,
    }));
    controller.restore(restored);
    closePanels();
    void tick().then(() => {
      if (skipAutoForInput === restored.input) skipAutoForInput = null;
    });
  }

  async function init(): Promise<void> {
    await configController.load();
    for (const message of queuedErrors) controller.showError(message);
    queuedErrors = [];
  }

  function destroy(): void {
    controller.destroy();
    clipboardReader.cancel();
    speaker.stop();
    for (const timer of copyFeedbackTimers.values()) clearTimeout(timer);
    copyFeedbackTimers.clear();
  }

  return {
    get input() { return input; },
    get source() { return source; },
    get target() { return target; },
    get showConfig() { return showConfig; },
    get showHistory() { return showHistory; },
    get copied() { return copied; },
    get copiedEngines() { return copiedEngines; },
    get speakingText() { return speakingText; },
    get config() { return config; },
    get configReady() { return configReady; },
    get clipboardReadEnabled() { return clipboardReadEnabled; },
    get availableCompareEngines() { return availableCompareEngines; },
    get missingCompareEngines() { return missingCompareEngines; },
    get credentialsReady() { return credentialsReady; },
    get inputBytes() { return inputBytes; },
    get canTranslate() { return canTranslate; },
    get unavailableReason() { return unavailableReason; },
    get credentialMessage() { return credentialMessage; },
    get translation() { return translation; },
    get configController() { return configController; },
    get translateController() { return controller; },
    isConfigured,
    patchConfig,
    init,
    destroy,
    requestTranslation,
    setLanguage,
    setBaiduDomain,
    swapLanguages,
    toggleCompareEngine,
    speak,
    copyText,
    retranslate,
    focusInputFromShortcut,
    clearInput,
    updateInput,
    toggleTheme,
    openPanel,
    closePanels,
    restoreHistoryEntry,
    handleGlobalKeydown,
    handleWindowFocus,
  };
}
