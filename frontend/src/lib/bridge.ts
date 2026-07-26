import {
  GetConfig,
  AppendHistory,
  ClearHistory,
  ExportHistory,
  SaveConfig,
  QueryHistory,
  TestBaiduConnection,
  TestConnection,
  CancelTranslation,
  TranslateMulti,
  TranslateText,
} from "../../wailsjs/go/main/App";
import { ClipboardGetText, ClipboardSetText, EventsOn } from "../../wailsjs/runtime/runtime";
import { config as wailsConfig } from "../../wailsjs/go/models";
import type {
  CloudConfig,
  BaiduConfig,
  EngineId,
  EngineTranslateResult,
  HistoryEntry,
  HistoryPage,
  HistoryQuery,
  MultiTranslateRequest,
  MultiTranslateResult,
  ServiceConfig,
  TranslateRequest,
  TranslateResult,
} from "./types";

export interface DesktopBridge {
  readonly kind: "wails" | "mock";
  getConfig(): Promise<CloudConfig>;
  saveConfig(config: CloudConfig): Promise<void>;
  queryHistory(query: HistoryQuery): Promise<HistoryPage>;
  appendHistory(entry: HistoryEntry): Promise<boolean>;
  clearHistory(): Promise<void>;
  exportHistory(): Promise<boolean>;
  testConnection(engine: EngineId, service: ServiceConfig | BaiduConfig): Promise<void>;
  translateText(request: TranslateRequest): Promise<TranslateResult>;
  translateMulti(request: MultiTranslateRequest): Promise<MultiTranslateResult>;
  cancelTranslation(requestId: string): Promise<boolean>;
  onEngineResult(handler: (result: EngineTranslateResult) => void): () => void;
  getClipboardText(): Promise<string>;
  setClipboardText(text: string): Promise<void>;
}

export const DEFAULT_CONFIG: CloudConfig = {
  version: 3,
  tencent: { secretId: "", secretKey: "", region: "" },
  aliyun: { secretId: "", secretKey: "", region: "cn-hangzhou" },
  baidu: { appId: "", secretKey: "", domain: "general" },
  defaultEngine: "tencent",
  isDark: true,
  sidebarCollapsed: false,
  autoTranslate: true,
  sourceLanguage: "auto",
  targetLanguage: "zh",
  compareMode: false,
  compareEngines: ["tencent", "aliyun", "baidu"],
  clipboardWatch: false,
};

export function cloneConfig(config: CloudConfig): CloudConfig {
  return {
    ...config,
    tencent: { ...config.tencent },
    aliyun: { ...config.aliyun },
    baidu: { ...config.baidu },
    compareEngines: [...config.compareEngines],
  };
}

function createWailsBridge(): DesktopBridge {
  return {
    kind: "wails",
    getConfig: () => GetConfig() as Promise<CloudConfig>,
    saveConfig: (config) => SaveConfig(wailsConfig.CloudConfig.createFrom(config)),
    queryHistory: (query) => QueryHistory(query) as Promise<HistoryPage>,
    appendHistory: (entry) => AppendHistory(entry),
    clearHistory: () => ClearHistory(),
    exportHistory: () => ExportHistory(),
    testConnection: (engine, service) => engine === "baidu"
      ? TestBaiduConnection(service as BaiduConfig)
      : TestConnection(engine, service as ServiceConfig),
    translateText: (request) => TranslateText(request) as Promise<TranslateResult>,
    translateMulti: (request) => TranslateMulti(request) as Promise<MultiTranslateResult>,
    cancelTranslation: (requestId) => CancelTranslation(requestId),
    onEngineResult: (handler) =>
      EventsOn("translate:engine-result", (payload: EngineTranslateResult) => handler(payload)),
    getClipboardText: () => ClipboardGetText(),
    async setClipboardText(text) {
      if (!await ClipboardSetText(text)) throw new Error("clipboard write failed");
    },
  };
}

function mockTranslation(text: string, target: string): string {
  const examples: Record<string, string> = {
    hello: "你好",
    "simple translate": "简单翻译",
    "你好": "Hello",
  };
  return examples[text.trim().toLowerCase()] ?? `[${target.toUpperCase()}] ${text.trim()}`;
}

export function createMockBridge(initial: CloudConfig = DEFAULT_CONFIG): DesktopBridge {
  let config = cloneConfig(initial);
  let history: HistoryEntry[] = [];
  const cancelledRequests = new Set<string>();
  const handlers = new Set<(result: EngineTranslateResult) => void>();

  return {
    kind: "mock",
    async getConfig() {
      return cloneConfig(config);
    },
    async saveConfig(next) {
      config = cloneConfig(next);
    },
    async queryHistory(request) {
      const needle = request.query.trim().toLowerCase();
      const filtered = needle
        ? history.filter((item) => item.input.toLowerCase().includes(needle) || item.output.toLowerCase().includes(needle))
        : history;
      const offset = Math.max(0, request.offset);
      const limit = Math.min(50, Math.max(1, request.limit));
      return {
        entries: filtered.slice(offset, offset + limit).map((item) => ({ ...item })),
        total: filtered.length,
        allTotal: history.length,
        hasMore: offset + limit < filtered.length,
      };
    },
    async appendHistory(entry) {
      const latest = history[0];
      if (latest && latest.input === entry.input && latest.source === entry.source && latest.target === entry.target) return false;
      history = [{ ...entry }, ...history].slice(0, 200);
      return true;
    },
    async clearHistory() {
      history = [];
    },
    async exportHistory() {
      return history.length > 0;
    },
    async testConnection(engine, service) {
      const valid = engine === "baidu"
        ? Boolean((service as BaiduConfig).appId.trim() && service.secretKey.trim())
        : engine === "tencent"
          ? Boolean(service.secretKey.trim())
          : Boolean((service as ServiceConfig).secretId.trim() && service.secretKey.trim());
      if (!valid) throw new Error("请先填写完整凭据");
    },
    async translateText(request) {
      if (cancelledRequests.has(request.requestId)) {
        cancelledRequests.delete(request.requestId);
        return {
          requestId: request.requestId,
          source: request.source,
          autoSrc: request.source === "auto" ? "en" : request.source,
          target: request.target,
          text: "",
          error: "请求已取消",
          errorCode: "cancelled",
        };
      }
      return {
        requestId: request.requestId,
        source: request.source,
        autoSrc: request.source === "auto" ? "en" : request.source,
        target: request.target,
        text: mockTranslation(request.text, request.target),
      };
    },
    async translateMulti(request) {
      if (cancelledRequests.has(request.requestId)) {
        cancelledRequests.delete(request.requestId);
        return {
          requestId: request.requestId,
          source: request.source,
          autoSrc: request.source === "auto" ? "en" : request.source,
          target: request.target,
          results: Object.fromEntries(request.engines.map((engine) => [engine, {
            requestId: request.requestId,
            engine,
            text: "",
            error: "请求已取消",
            errorCode: "cancelled",
          }])) as Partial<Record<EngineId, EngineTranslateResult>>,
        };
      }
      const results: Partial<Record<EngineId, EngineTranslateResult>> = {};
      for (const engine of request.engines) {
        const result: EngineTranslateResult = {
          requestId: request.requestId,
          engine,
          text: mockTranslation(request.text, request.target),
        };
        results[engine] = result;
        queueMicrotask(() => handlers.forEach((handler) => handler(result)));
      }
      return {
        requestId: request.requestId,
        source: request.source,
        autoSrc: request.source === "auto" ? "en" : request.source,
        target: request.target,
        results,
      };
    },
    async cancelTranslation(requestId) {
      if (!requestId) return false;
      cancelledRequests.add(requestId);
      return true;
    },
    onEngineResult(handler) {
      handlers.add(handler);
      return () => handlers.delete(handler);
    },
    async getClipboardText() {
      return "";
    },
    async setClipboardText(text) {
      if (!navigator.clipboard?.writeText) throw new Error("clipboard unavailable");
      await navigator.clipboard.writeText(text);
    },
  };
}

function hasWailsRuntime(): boolean {
  return typeof window !== "undefined" && "go" in window && "runtime" in window;
}

export const desktopBridge: DesktopBridge = hasWailsRuntime()
  ? createWailsBridge()
  : createMockBridge();
