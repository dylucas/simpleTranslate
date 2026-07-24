import { get, writable, type Readable } from "svelte/store";
import { cloneConfig, DEFAULT_CONFIG, type DesktopBridge } from "./bridge";
import { langs } from "./languages";
import { isEngineId } from "./engines";
import type { BaiduDomain, CloudConfig, EngineId } from "./types";

const BAIDU_DOMAINS = new Set<BaiduDomain>([
  "general", "it", "finance", "machinery", "senimed", "novel",
  "academic", "aerospace", "wiki", "news", "law", "contract",
]);

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

function normalizeLanguage(value: string | undefined, allowAuto: boolean, fallback: string): string {
  let normalized = value?.trim().toLowerCase() ?? "";
  if (normalized === "ja") normalized = "jp";
  if (normalized === "ko") normalized = "kr";
  if (allowAuto && normalized === "auto") return normalized;
  return normalized in langs ? normalized : fallback;
}

export function normalizeConfig(input?: Partial<CloudConfig> | null): CloudConfig {
  const defaultEngine: EngineId = isEngineId(input?.defaultEngine) ? input.defaultEngine : "tencent";
  const compareEngines = input?.compareEngines?.filter(
    (engine): engine is EngineId => isEngineId(engine),
  );
  const inputDomain = input?.baidu?.domain;
  const domain: BaiduDomain = inputDomain && BAIDU_DOMAINS.has(inputDomain) ? inputDomain : "general";
  return {
    ...DEFAULT_CONFIG,
    ...input,
    version: 3,
    defaultEngine,
    tencent: { ...DEFAULT_CONFIG.tencent, ...input?.tencent },
    aliyun: { ...DEFAULT_CONFIG.aliyun, ...input?.aliyun },
    baidu: { ...DEFAULT_CONFIG.baidu, ...input?.baidu, domain },
    sourceLanguage: normalizeLanguage(input?.sourceLanguage, true, "auto"),
    targetLanguage: normalizeLanguage(input?.targetLanguage, false, "zh"),
    compareEngines: compareEngines?.length ? [...new Set(compareEngines)] : ["tencent", "aliyun", "baidu"],
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
  let loading: Promise<void> | null = null;

  function load(): Promise<void> {
    if (get(store).ready) return Promise.resolve();
    if (!loading) {
      loading = (async () => {
        try {
          persisted = normalizeConfig(await bridge.getConfig());
          store.set({ value: cloneConfig(persisted), ready: true, saving: false });
        } catch {
          store.set({ value: cloneConfig(DEFAULT_CONFIG), ready: true, saving: false });
          onError("读取设置失败，已使用默认配置");
        }
      })().finally(() => {
        loading = null;
      });
    }
    return loading;
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
    if (loading) await loading;
    const next = normalizeConfig(config);
    store.update((state) => ({ ...state, value: cloneConfig(next) }));
    await enqueue(next);
  }

  async function patch<K extends keyof CloudConfig>(key: K, value: CloudConfig[K]): Promise<void> {
    if (loading) await loading;
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
