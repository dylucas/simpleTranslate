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
  EngineId,
  EngineTranslateResult,
  ErrorToast,
  HistoryEntry,
  TranslateErrorCode,
} from "./types";

export interface TranslationState {
  isProcessing: boolean;
  status: string;
  output: string;
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
  getCompareMode: () => boolean;
  getCompareEngines: () => EngineId[];
  getHistory: () => HistoryEntry[];
  setHistory: (updater: (history: HistoryEntry[]) => HistoryEntry[]) => void;
  setTarget: (target: string) => void;
}

export interface TranslateController {
  state: Writable<TranslationState>;
  translate(): Promise<void>;
  handleAutoTranslate(value: string): void;
  showError(error: unknown): void;
  dismissError(): void;
  retry(): void;
  setOutput(output: string): void;
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

export function createTranslateController(deps: TranslateControllerDeps): TranslateController {
  const state = writable<TranslationState>(initialState());
  let requestSequence = 0;
  let pendingRetry = false;
  let lastTranslatedInput = "";
  let autoTimer: ReturnType<typeof setTimeout> | null = null;
  let errorTimer: ReturnType<typeof setTimeout> | null = null;

  function showError(error: unknown): void {
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
    if (!output || isDuplicateRecent(deps.getHistory(), input, source, target)) return;
    const entry = createHistoryEntry({ input, output, source, target });
    deps.setHistory((history) => prependHistory(history, entry));
  }

  const unsubscribeEngineResult = deps.bridge.onEngineResult((payload) => {
    const current = get(state);
    if (!payload?.engine || payload.requestId !== current.activeRequestId) return;
    if (!current.compareLoadingEngines[payload.engine]) return;
    state.update((value) => ({
      ...value,
      compareOutputs: { ...value.compareOutputs, [payload.engine]: payload },
      compareLoadingEngines: { ...value.compareLoadingEngines, [payload.engine]: false },
    }));
  });

  async function translate(): Promise<void> {
    const input = deps.getInput();
    if (!input.trim()) return;
    if (autoTimer) {
      clearTimeout(autoTimer);
      autoTimer = null;
    }
    if (get(state).isProcessing) {
      pendingRetry = true;
      return;
    }

    const requestId = `${Date.now()}-${++requestSequence}`;
    const source = deps.getSource();
    const target = deps.getTarget();
    const compareMode = deps.getCompareMode();
    const engines = deps.getCompareEngines();
    lastTranslatedInput = input;
    pendingRetry = false;

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
        if (response.requestId !== get(state).activeRequestId) return;
        const results = response.results ?? {};
        const values = Object.values(results);
        const failed = values.filter((result) => result?.error || result?.errorCode);
        const preferred = results[deps.getActiveEngine()];
        const firstSuccess = values.find((result) => result?.text && !result.error);
        const output = preferred?.text || firstSuccess?.text || get(state).output;

        state.update((current) => ({
          ...current,
          output,
          compareOutputs: results,
          compareLoadingEngines: {},
          status: failed.length === values.length ? "翻译失败" : failed.length ? "部分引擎失败" : "完成",
        }));
        if (failed.length === values.length && failed[0]) showError(failed[0]);
        else addHistory(input, output, source, response.target || target);

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
        if (response.requestId !== get(state).activeRequestId) return;
        if (response.errorCode || response.error) {
          state.update((current) => ({ ...current, status: "翻译失败" }));
          showError(response);
        } else {
          deps.setTarget(response.target || target);
          state.update((current) => ({
            ...current,
            output: response.text,
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
      if (requestId !== get(state).activeRequestId) return;
      state.update((current) => ({ ...current, status: "翻译失败" }));
      showError(error);
    } finally {
      if (requestId === get(state).activeRequestId) {
        state.update((current) => ({ ...current, isProcessing: false, compareLoadingEngines: {} }));
      }
      const latest = deps.getInput();
      if (pendingRetry && latest.trim() && latest !== lastTranslatedInput) {
        pendingRetry = false;
        void translate();
      }
    }
  }

  function handleAutoTranslate(value: string): void {
    if (autoTimer) clearTimeout(autoTimer);
    if (!value.trim()) return;
    autoTimer = setTimeout(() => {
      if (value === deps.getInput()) void translate();
    }, AUTO_TRANSLATE_DELAY);
  }

  return {
    state,
    translate,
    handleAutoTranslate,
    showError,
    dismissError,
    retry: () => {
      dismissError();
      void translate();
    },
    setOutput: (output) => state.update((current) => ({ ...current, output })),
    restore: (entry) => state.update((current) => ({
      ...current,
      output: entry.output,
      compareOutputs: {},
      status: "已从历史恢复",
    })),
    destroy: () => {
      unsubscribeEngineResult();
      if (autoTimer) clearTimeout(autoTimer);
      if (errorTimer) clearTimeout(errorTimer);
    },
  };
}
