import {
  GetConfig,
  LoadHistory,
  SaveConfig,
  SaveHistory,
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
  loadHistory(): Promise<HistoryEntry[]>;
  saveHistory(history: HistoryEntry[]): Promise<void>;
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
    loadHistory: () => LoadHistory() as Promise<HistoryEntry[]>,
    saveHistory: (history) => SaveHistory(history),
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
    async loadHistory() {
      return history.map((item) => ({ ...item }));
    },
    async saveHistory(next) {
      history = next.map((item) => ({ ...item }));
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
