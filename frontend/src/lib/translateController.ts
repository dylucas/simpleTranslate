// 翻译控制器：封装翻译流程的状态与逻辑，与组件解耦。
//
// 持有所有翻译相关状态（isProcessing/status/output/compareOutputs/errorToast 等），
// 通过依赖注入读取宿主组件的输入文本、语言对、引擎配置，
// 通过回调通知宿主更新历史记录与目标语言。
//
// 使用方式：
//   const ctrl = createTranslateController({
//     getInput: () => input,
//     getSource: () => source,
//     getTarget: () => target,
//     getActiveEngine: () => activeEngine,
//     getCompareMode: () => compareMode,
//     getCompareEngines: () => compareEngines,
//     getHistory: () => history,
//     setHistory: (updater) => { history = updater(history); },
//     setTarget: (t) => { target = t; },
//   });
//   $: state = $ctrl.state;
//   await ctrl.translate();

import { writable, get, type Writable } from "svelte/store";
import { TranslateText, TranslateMulti } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { formatAutoDetect, DEFAULT_COMPARE_ENGINES } from "./languages";
import {
  isDuplicateRecent,
  createHistoryEntry,
  prependHistory,
  HISTORY_LIMIT,
} from "./history";
import {
  formatErrorToToast,
  isRetryable,
  shouldShowSettingsButton,
  ErrorCodes,
} from "./errors";
import type {
  HistoryEntry,
  EngineTranslateResult,
  ErrorToast,
  TranslateErrorCode,
} from "./types";

// 翻译状态集合：所有由翻译流程驱动、UI 消费的状态
export interface TranslationState {
  isProcessing: boolean;
  status: string;
  output: string;
  compareOutputs: Record<string, EngineTranslateResult>;
  compareLoadingEngines: Record<string, boolean>;
  errorToast: ErrorToast | null;
  autoDetectLang: string;
  lastDetectedLang: string;
}

// 宿主组件需提供的依赖：读取输入状态 + 回写历史/目标语言
export interface TranslateControllerDeps {
  getInput: () => string;
  getSource: () => string;
  getTarget: () => string;
  getActiveEngine: () => string;
  getCompareMode: () => boolean;
  getCompareEngines: () => string[];
  getHistory: () => HistoryEntry[];
  setHistory: (updater: (h: HistoryEntry[]) => HistoryEntry[]) => void;
  setTarget: (t: string) => void;
}

// 自动翻译防抖延迟
const AUTO_TRANSLATE_DELAY = 700;
// 错误提示自动消失延迟
const ERROR_TOAST_DURATION = 4000;

// 初始状态
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
  };
}

export interface TranslateController {
  state: Writable<TranslationState>;
  translate: () => Promise<void>;
  handleAutoTranslate: (val: string) => void;
  showError: (err: unknown) => void;
  dismissError: () => void;
  retry: () => void;
  destroy: () => void;
}

// createTranslateController 构造翻译控制器实例。
// 内部订阅 translate:engine-result 事件实现逐引擎流式渲染，
// 在 destroy() 时取消订阅并清理定时器。
export function createTranslateController(
  deps: TranslateControllerDeps
): TranslateController {
  const state = writable<TranslationState>(initialState());

  // 请求序列号与重试标记：用于丢弃过期响应、在途请求期间输入变化自动重试
  let translateSeq = 0;
  let pendingRetry = false;
  let lastTranslatedInput = "";
  let autoTimer: ReturnType<typeof setTimeout> | null = null;
  let errorTimer: ReturnType<typeof setTimeout> | null = null;

  // showError 展示错误提示，4 秒后自动消失。支持结构化错误与字符串。
  function showError(err: unknown): void {
    const msg = formatErrorToToast(err as any);
    const code: TranslateErrorCode =
      err && typeof err === "object" && "errorCode" in err
        ? ((err as any).errorCode as TranslateErrorCode) || ErrorCodes.Unknown
        : ErrorCodes.Unknown;
    if (errorTimer) clearTimeout(errorTimer);
    state.update((s) => ({
      ...s,
      errorToast: {
        msg,
        code,
        canRetry: isRetryable(code),
        showSettings: shouldShowSettingsButton(code),
        ts: Date.now(),
      },
    }));
    errorTimer = setTimeout(() => {
      state.update((s) =>
        s.errorToast && s.errorToast.msg === msg
          ? { ...s, errorToast: null }
          : s
      );
    }, ERROR_TOAST_DURATION);
  }

  function dismissError(): void {
    if (errorTimer) clearTimeout(errorTimer);
    state.update((s) => ({ ...s, errorToast: null }));
  }

  // addHistory 去重后 prepend，通过宿主回调写入
  function addHistory(
    inputText: string,
    outputText: string,
    src: string,
    tgt: string
  ): void {
    const history = deps.getHistory();
    if (isDuplicateRecent(history, inputText, src, tgt)) return;
    const entry = createHistoryEntry({
      input: inputText,
      output: outputText,
      source: src,
      target: tgt,
    });
    deps.setHistory((h) => prependHistory(h, entry, HISTORY_LIMIT));
  }

  // 订阅逐引擎完成事件：仅处理仍在 loading 的引擎，过期事件忽略
  const unsubscribeEngineResult = EventsOn(
    "translate:engine-result",
    (payload: EngineTranslateResult) => {
      if (!payload || !payload.engine) return;
      const cur = get(state);
      if (!cur.compareLoadingEngines[payload.engine]) return;
      state.update((s) => ({
        ...s,
        compareOutputs: {
          ...s.compareOutputs,
          [payload.engine]: {
            engine: payload.engine,
            text: payload.text || "",
            error: payload.error,
            errorCode: payload.errorCode as TranslateErrorCode | undefined,
          },
        },
        compareLoadingEngines: {
          ...s.compareLoadingEngines,
          [payload.engine]: false,
        },
      }));
    }
  );

  async function translate(): Promise<void> {
    const input = deps.getInput();
    if (!input.trim()) return;

    const cur = get(state);
    // 已有请求在途：标记需重试
    if (cur.isProcessing) {
      pendingRetry = true;
      return;
    }

    // 捕获本次请求快照，用于响应返回时判断是否过期
    const seq = ++translateSeq;
    const reqInput = input;
    const reqSource = deps.getSource();
    const reqTarget = deps.getTarget();
    const reqCompare = deps.getCompareMode();
    lastTranslatedInput = reqInput;
    pendingRetry = false;

    state.update((s) => ({ ...s, isProcessing: true, status: "翻译中..." }));

    try {
      let res: any;
      const engines = reqCompare
        ? Array.isArray(deps.getCompareEngines())
          ? deps.getCompareEngines()
          : [...DEFAULT_COMPARE_ENGINES]
        : [];

      if (reqCompare) {
        // 初始化逐引擎 loading，清空旧结果
        state.update((s) => ({
          ...s,
          compareLoadingEngines: engines.reduce(
            (acc: Record<string, boolean>, e: string) => ({
              ...acc,
              [e]: true,
            }),
            {}
          ),
          compareOutputs: engines.reduce(
            (acc: Record<string, EngineTranslateResult>, e: string) => ({
              ...acc,
              [e]: { engine: e, text: "" },
            }),
            {}
          ),
        }));
        res = await TranslateMulti(reqInput, reqSource, reqTarget, engines);
      } else {
        res = await TranslateText(
          reqInput,
          reqSource,
          reqTarget,
          deps.getActiveEngine()
        );
      }

      // 丢弃过期响应
      if (seq !== translateSeq) return;

      if (reqCompare) {
        const results =
          (res.results as Record<string, EngineTranslateResult>) || {};
        // 用最终返回值兜底覆盖（事件可能尚未到达），清空所有 loading
        state.update((s) => ({
          ...s,
          compareOutputs: results,
          compareLoadingEngines: engines.reduce(
            (acc: Record<string, boolean>, e: string) => ({
              ...acc,
              [e]: false,
            }),
            {}
          ),
        }));

        const allErrored =
          Object.keys(results).length > 0 &&
          Object.values(results).every((r) => r.errorCode || r.error);
        const anyErrored = Object.values(results).some(
          (r) => r.errorCode || r.error
        );
        const preferredEngine = deps.getActiveEngine() || engines[0];
        const output = results?.[preferredEngine]?.text || "";

        if (allErrored) {
          const firstErr =
            Object.values(results).find((r) => r.errorCode || r.error) || res;
          state.update((s) => ({
            ...s,
            output,
            status: "翻译失败",
          }));
          showError(firstErr);
        } else {
          state.update((s) => ({
            ...s,
            output,
            status: anyErrored ? "部分引擎失败" : "完成",
          }));
          addHistory(reqInput, output, reqSource, deps.getTarget());
        }
      } else {
        if (res.errorCode) {
          state.update((s) => ({
            ...s,
            output: "",
            compareOutputs: {},
            status: "翻译失败",
          }));
          showError(res);
        } else {
          // 仅当用户未在请求期间手动改 target 时，同步后端兜底后的 target
          if (reqTarget === deps.getTarget()) {
            deps.setTarget(res.target || reqTarget);
          }
          state.update((s) => ({
            ...s,
            output: res.text,
            compareOutputs: {},
            status: "完成",
          }));
          addHistory(reqInput, res.text, reqSource, deps.getTarget());
        }
      }

      // auto 模式下同步后端识别到的语种（单引擎与对照统一处理）
      if (reqSource === "auto") {
        const autoSrc = (res.autoSrc as string) || "";
        const hasError = reqCompare
          ? false // 对照模式部分成功也算识别成功
          : !!res.errorCode;
        if (autoSrc && !hasError) {
          state.update((s) => ({
            ...s,
            lastDetectedLang: autoSrc,
            autoDetectLang: formatAutoDetect(autoSrc),
          }));
        }
      }
    } catch (e) {
      if (seq !== translateSeq) return;
      state.update((s) => ({ ...s, status: "翻译失败" }));
      showError(e);
      console.error(e);
    } finally {
      state.update((s) => ({
        ...s,
        isProcessing: false,
        compareLoadingEngines: {},
      }));
      // 请求期间输入又变化，重新触发
      const curInput = deps.getInput();
      if (
        pendingRetry &&
        curInput.trim() &&
        curInput !== lastTranslatedInput
      ) {
        pendingRetry = false;
        translate();
      }
    }
  }

  // handleAutoTranslate 自动翻译防抖：输入变化后延迟触发
  function handleAutoTranslate(val: string): void {
    if (autoTimer) clearTimeout(autoTimer);
    if (!val || !val.trim()) return;
    autoTimer = setTimeout(() => {
      if (val === deps.getInput()) translate();
    }, AUTO_TRANSLATE_DELAY);
  }

  // retry 用户点击错误提示的重试按钮
  function retry(): void {
    dismissError();
    translate();
  }

  function destroy(): void {
    unsubscribeEngineResult();
    if (autoTimer) clearTimeout(autoTimer);
    if (errorTimer) clearTimeout(errorTimer);
  }

  return { state, translate, handleAutoTranslate, showError, dismissError, retry, destroy };
}
