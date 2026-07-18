import { get, writable, type Readable } from "svelte/store";
import { cloneConfig, DEFAULT_CONFIG, type DesktopBridge } from "./bridge";
import type { CloudConfig, EngineId } from "./types";

export interface ConfigState {
  value: CloudConfig;
  ready: boolean;
  saving: boolean;
}

export interface ConfigController extends Readable<ConfigState> {
  load(): Promise<void>;
  patch<K extends keyof CloudConfig>(key: K, value: CloudConfig[K]): Promise<void>;
  save(config: CloudConfig): Promise<void>;
  snapshot(): CloudConfig;
}

export function normalizeConfig(input?: Partial<CloudConfig> | null): CloudConfig {
  const defaultEngine: EngineId = input?.defaultEngine === "aliyun" ? "aliyun" : "tencent";
  const compareEngines = input?.compareEngines?.filter(
    (engine): engine is EngineId => engine === "tencent" || engine === "aliyun",
  );
  return {
    ...DEFAULT_CONFIG,
    ...input,
    version: 2,
    defaultEngine,
    tencent: { ...DEFAULT_CONFIG.tencent, ...input?.tencent },
    aliyun: { ...DEFAULT_CONFIG.aliyun, ...input?.aliyun },
    sourceLanguage: input?.sourceLanguage || "auto",
    targetLanguage: input?.targetLanguage || "zh",
    compareEngines: compareEngines?.length ? [...new Set(compareEngines)] : ["tencent", "aliyun"],
  };
}

export function createConfigController(
  bridge: DesktopBridge,
  onError: (message: string) => void,
): ConfigController {
  const store = writable<ConfigState>({
    value: cloneConfig(DEFAULT_CONFIG),
    ready: false,
    saving: false,
  });
  let persisted = cloneConfig(DEFAULT_CONFIG);
  let queued: CloudConfig | null = null;
  let draining: Promise<void> | null = null;

  async function load(): Promise<void> {
    try {
      persisted = normalizeConfig(await bridge.getConfig());
      store.set({ value: cloneConfig(persisted), ready: true, saving: false });
    } catch {
      store.set({ value: cloneConfig(DEFAULT_CONFIG), ready: true, saving: false });
      onError("读取设置失败，已使用默认配置");
    }
  }

  async function drain(): Promise<void> {
    store.update((state) => ({ ...state, saving: true }));
    try {
      while (queued) {
        const next = queued;
        queued = null;
        await bridge.saveConfig(next);
        persisted = cloneConfig(next);
      }
    } catch {
      queued = null;
      store.set({ value: cloneConfig(persisted), ready: true, saving: false });
      onError("设置保存失败，已恢复上次配置");
      throw new Error("config save failed");
    } finally {
      draining = null;
      store.update((state) => ({ ...state, saving: false }));
    }
  }

  function enqueue(config: CloudConfig): Promise<void> {
    queued = cloneConfig(normalizeConfig(config));
    if (!draining) draining = drain();
    return draining;
  }

  async function save(config: CloudConfig): Promise<void> {
    const next = normalizeConfig(config);
    store.update((state) => ({ ...state, value: cloneConfig(next) }));
    await enqueue(next);
  }

  async function patch<K extends keyof CloudConfig>(key: K, value: CloudConfig[K]): Promise<void> {
    const next = cloneConfig(get(store).value);
    next[key] = value;
    await save(next);
  }

  return {
    subscribe: store.subscribe,
    load,
    patch,
    save,
    snapshot: () => cloneConfig(get(store).value),
  };
}
