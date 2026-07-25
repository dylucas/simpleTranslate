import { get, writable, type Writable } from "svelte/store";
import type { DesktopBridge } from "./bridge";
import { formatAutoDetect } from "./languages";
import { createHistoryEntry, isDuplicateRecent, prependHistory } from "./history";
import {
  ErrorCodes,
  formatErrorToToast,
  isRetryable,
  shouldShowSettingsButton,
  type AnyErrorInput,
} from "./errors";
import type {
  BaiduDomain,
  EngineId,
  EngineTranslateResult,
  ErrorToast,
  HistoryEntry,
  TranslateErrorCode,
} from "./types";
import { MAX_INPUT_BYTES, utf8ByteLength } from "./textLimits";

export interface TranslationState {
  isProcessing: boolean;
  status: string;
  output: string;
  notice: string;
  compareOutputs: Partial<Record<EngineId, EngineTranslateResult>>;
  compareLoadingEngines: Partial<Record<EngineId, boolean>>;
  errorToast: ErrorToast | null;
  autoDetectLang: string;
  lastDetectedLang: string;
  activeRequestId: string;
}

export interface TranslateControllerDeps {
  bridge: DesktopBridge;
  getInput: () => string;
  getSource: () => string;
  getTarget: () => string;
  getActiveEngine: () => EngineId;
  getBaiduDomain: () => BaiduDomain;
  getCompareMode: () => boolean;
  getCompareEngines: () => EngineId[];
  appendHistory?: (entry: HistoryEntry) => Promise<boolean>;
  getHistory?: () => HistoryEntry[];
  setHistory?: (updater: (history: HistoryEntry[]) => HistoryEntry[]) => void;
  setTarget: (target: string) => void;
}

export interface TranslateController {
  state: Writable<TranslationState>;
  translate(): Promise<void>;
  cancel(): void;
  handleAutoTranslate(value: string): void;
  cancelAutoTranslate(): void;
  showError(error: unknown): void;
  dismissError(): void;
  retry(): void;
  setOutput(output: string): void;
  clear(): void;
  restore(entry: HistoryEntry): void;
  destroy(): void;
}

const AUTO_TRANSLATE_DELAY = 700;
const ERROR_TOAST_DURATION = 5000;

function initialState(): TranslationState {
  return {
    isProcessing: false,
    status: "准备就绪",
    output: "",
    notice: "",
    compareOutputs: {},
    compareLoadingEngines: {},
    errorToast: null,
    autoDetectLang: "自动识别",
    lastDetectedLang: "",
    activeRequestId: "",
  };
}

function asErrorInput(error: unknown): AnyErrorInput {
  if (typeof error === "string" || error instanceof Error || error == null) return error;
  if (typeof error === "object") {
    const value = error as { error?: unknown; errorCode?: unknown };
    return {
      error: typeof value.error === "string" ? value.error : undefined,
      errorCode: typeof value.errorCode === "string" ? value.errorCode : undefined,
    };
  }
  return undefined;
}

function errorCode(error: AnyErrorInput): TranslateErrorCode {
  if (error && typeof error === "object" && "errorCode" in error) {
    const code = error.errorCode;
    if (typeof code === "string") return code as TranslateErrorCode;
  }
  return ErrorCodes.Unknown;
}

function isSuccessfulResult(result: EngineTranslateResult | undefined): result is EngineTranslateResult {
  return Boolean(result?.text?.trim()) && !result?.error && !result?.errorCode;
}

export function createTranslateController(deps: TranslateControllerDeps): TranslateController {
  const state = writable<TranslationState>(initialState());
  let requestSequence = 0;
  let destroyed = false;
  let autoTimer: ReturnType<typeof setTimeout> | null = null;
  let errorTimer: ReturnType<typeof setTimeout> | null = null;
  let activeRequestKey: string | null = null;
  let previousCompareOutputs: Partial<Record<EngineId, EngineTranslateResult>> | null = null;
  const cancelledRequestIds = new Set<string>();

  function currentRequestKey(): string {
    const compareMode = deps.getCompareMode();
    return JSON.stringify([
      deps.getInput(),
      deps.getSource(),
      deps.getTarget(),
      compareMode,
      deps.getActiveEngine(),
      deps.getBaiduDomain(),
      compareMode ? deps.getCompareEngines() : [],
    ]);
  }

  function showError(error: unknown): void {
    if (destroyed) return;
    const input = asErrorInput(error);
    const code = errorCode(input);
    const msg = formatErrorToToast(input);
    if (errorTimer) clearTimeout(errorTimer);
    state.update((current) => ({
      ...current,
      errorToast: {
        msg,
        code,
        canRetry: isRetryable(code),
        showSettings: shouldShowSettingsButton(code),
        ts: Date.now(),
      },
    }));
    errorTimer = setTimeout(() => dismissError(), ERROR_TOAST_DURATION);
  }

  function dismissError(): void {
    if (errorTimer) clearTimeout(errorTimer);
    errorTimer = null;
    state.update((current) => ({ ...current, errorToast: null }));
  }

  function addHistory(input: string, output: string, source: string, target: string): void {
    if (!output) return;
    const entry = createHistoryEntry({ input, output, source, target });
    if (deps.appendHistory) {
      void deps.appendHistory(entry).catch(() => showError("历史记录保存失败"));
      return;
    }
    const history = deps.getHistory?.() ?? [];
    if (!isDuplicateRecent(history, input, source, target)) deps.setHistory?.((current) => prependHistory(current, entry));
  }

  const unsubscribeEngineResult = deps.bridge.onEngineResult((payload) => {
    if (destroyed) return;
    const current = get(state);
    if (!payload?.engine || payload.requestId !== current.activeRequestId) return;
    if (activeRequestKey !== currentRequestKey()) return;
    if (!current.compareLoadingEngines[payload.engine]) return;
    state.update((value) => ({
      ...value,
      compareOutputs: { ...value.compareOutputs, [payload.engine]: payload },
      compareLoadingEngines: { ...value.compareLoadingEngines, [payload.engine]: false },
    }));
  });

  async function translate(): Promise<void> {
    if (destroyed) return;
    const input = deps.getInput();
    if (!input.trim()) return;
    if (utf8ByteLength(input) > MAX_INPUT_BYTES) {
      showError({ errorCode: ErrorCodes.InvalidInput, error: `原文不能超过 ${MAX_INPUT_BYTES} 个 UTF-8 字节` });
      return;
    }
    if (autoTimer) {
      clearTimeout(autoTimer);
      autoTimer = null;
    }
    if (get(state).isProcessing) return;

    const requestId = `${Date.now()}-${++requestSequence}`;
    const source = deps.getSource();
    const target = deps.getTarget();
    const compareMode = deps.getCompareMode();
    const engines = deps.getCompareEngines();
    const requestKey = currentRequestKey();
    activeRequestKey = requestKey;
    let responseApplied = false;
    previousCompareOutputs = get(state).compareOutputs;

    state.update((current) => ({
      ...current,
      isProcessing: true,
      status: "翻译中...",
      activeRequestId: requestId,
      compareLoadingEngines: compareMode
        ? Object.fromEntries(engines.map((engine) => [engine, true]))
        : {},
      compareOutputs: compareMode ? {} : current.compareOutputs,
    }));

    try {
      if (compareMode) {
        const response = await deps.bridge.translateMulti({
          requestId,
          text: input,
          source,
          target,
          engines,
        });
        if (destroyed) return;
        if (response.requestId !== get(state).activeRequestId) return;
        if (requestKey !== currentRequestKey()) return;
        responseApplied = true;
        const results = response.results ?? {};
        const values = Object.values(results);
        const failed = values.filter((result) => result?.error || result?.errorCode);
        const preferred = results[deps.getActiveEngine()];
        const successful = values.filter(isSuccessfulResult);
        const firstSuccess = successful[0];
        const output = isSuccessfulResult(preferred) ? preferred.text : firstSuccess?.text || get(state).output;
        const allFailed = successful.length === 0;
        const allCancelled = values.length > 0 && values.every((result) => result?.errorCode === ErrorCodes.Cancelled);
        const hasFailures = successful.length < engines.length;

        state.update((current) => ({
          ...current,
          output,
          notice: isSuccessfulResult(preferred) ? preferred.notice ?? "" : firstSuccess?.notice ?? "",
          compareOutputs: results,
          compareLoadingEngines: {},
          status: allCancelled ? "已取消" : allFailed ? "翻译失败" : hasFailures ? "部分引擎失败" : "完成",
        }));
        if (allFailed && !allCancelled) showError(failed[0] ?? "翻译服务未返回结果");
        else if (!allCancelled) addHistory(input, output, source, response.target || target);

        if (source === "auto" && response.autoSrc) {
          state.update((current) => ({
            ...current,
            lastDetectedLang: response.autoSrc,
            autoDetectLang: formatAutoDetect(response.autoSrc),
          }));
        }
      } else {
        const response = await deps.bridge.translateText({
          requestId,
          text: input,
          source,
          target,
          engine: deps.getActiveEngine(),
        });
        if (destroyed) return;
        if (response.requestId !== get(state).activeRequestId) return;
        if (requestKey !== currentRequestKey()) return;
        responseApplied = true;
        if (response.errorCode === ErrorCodes.Cancelled) {
          state.update((current) => ({ ...current, status: "已取消" }));
        } else if (response.errorCode || response.error || !response.text.trim()) {
          state.update((current) => ({ ...current, status: "翻译失败" }));
          showError(response.errorCode || response.error ? response : "翻译服务未返回结果");
        } else {
          deps.setTarget(response.target || target);
          state.update((current) => ({
            ...current,
            output: response.text,
            notice: response.notice ?? "",
            compareOutputs: {},
            status: "完成",
          }));
          addHistory(input, response.text, source, response.target || target);
          if (source === "auto" && response.autoSrc) {
            state.update((current) => ({
              ...current,
              lastDetectedLang: response.autoSrc,
              autoDetectLang: formatAutoDetect(response.autoSrc),
            }));
          }
        }
      }
    } catch (error) {
      if (destroyed) return;
      if (requestId !== get(state).activeRequestId) return;
      if (requestKey !== currentRequestKey()) return;
      responseApplied = true;
      state.update((current) => ({ ...current, status: "翻译失败" }));
      showError(error);
    } finally {
      if (!destroyed) {
        const wasCancelled = cancelledRequestIds.delete(requestId);
        if (requestId === get(state).activeRequestId) {
          state.update((current) => ({ ...current, isProcessing: false, compareLoadingEngines: {} }));
          activeRequestKey = null;
          previousCompareOutputs = null;
        }
        const latest = deps.getInput();
        if (!wasCancelled && !responseApplied && latest.trim() && requestKey !== currentRequestKey()) {
          void translate();
        }
      }
    }
  }

  function handleAutoTranslate(value: string): void {
    if (destroyed) return;
    if (autoTimer) clearTimeout(autoTimer);
    if (!value.trim()) return;
    autoTimer = setTimeout(() => {
      autoTimer = null;
      if (value !== deps.getInput()) return;
      if (get(state).isProcessing) cancel();
      void translate();
    }, AUTO_TRANSLATE_DELAY);
  }

  function cancelAutoTranslate(): void {
    if (autoTimer) clearTimeout(autoTimer);
    autoTimer = null;
  }

  function cancel(): void {
    if (destroyed) return;
    const current = get(state);
    const requestId = current.activeRequestId;
    if (!current.isProcessing || !requestId) return;

    const compareOutputs = previousCompareOutputs ?? current.compareOutputs;
    previousCompareOutputs = null;
    cancelledRequestIds.add(requestId);
    activeRequestKey = null;
    state.update((value) => ({
      ...value,
      isProcessing: false,
      status: "已取消",
      activeRequestId: "",
      compareOutputs,
      compareLoadingEngines: {},
    }));
    void deps.bridge.cancelTranslation(requestId).catch(() => undefined);
  }

  return {
    state,
    translate,
    cancel,
    handleAutoTranslate,
    cancelAutoTranslate,
    showError,
    dismissError,
    retry: () => {
      dismissError();
      void translate();
    },
    setOutput: (output) => {
      if (!destroyed) state.update((current) => ({ ...current, output, notice: "" }));
    },
    clear: () => {
      if (destroyed) return;
      cancel();
      cancelAutoTranslate();
      if (errorTimer) clearTimeout(errorTimer);
      errorTimer = null;
      activeRequestKey = null;
      previousCompareOutputs = null;
      state.set(initialState());
    },
    restore: (entry) => {
      if (destroyed) return;
      state.update((current) => ({
        ...current,
        output: entry.output,
        notice: "",
        compareOutputs: {},
        autoDetectLang: "自动识别",
        lastDetectedLang: "",
        status: "已从历史恢复",
      }));
    },
    destroy: () => {
      cancel();
      destroyed = true;
      unsubscribeEngineResult();
      if (autoTimer) clearTimeout(autoTimer);
      if (errorTimer) clearTimeout(errorTimer);
      activeRequestKey = null;
      previousCompareOutputs = null;
    },
  };
}
