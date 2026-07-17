import { writable } from "svelte/store";
import { GetConfig, SaveConfig } from "../../wailsjs/go/main/App";
import { config as wailsConfig } from "../../wailsjs/go/models";

type ServiceConfig = wailsConfig.ServiceConfig;

// CloudConfig 与后端 config.CloudConfig 对应（wails 自动生成版本同步）
export interface CloudConfig {
  isDark: boolean;
  sidebarCollapsed: boolean;
  defaultEngine: string;
  compareMode: boolean;
  compareEngines: string[];
  pickBest: boolean;
  clipboardWatch: boolean;
  tencent: ServiceConfig;
  aliyun: ServiceConfig;
}

// 默认配置：首次启动或后端配置丢失时使用
const DEFAULT_CONFIG: CloudConfig = {
  isDark: true,
  sidebarCollapsed: false,
  defaultEngine: "tencent",
  compareMode: false,
  compareEngines: ["tencent", "aliyun"],
  pickBest: false,
  clipboardWatch: false,
  tencent: { secretId: "", secretKey: "", region: "" } as ServiceConfig,
  aliyun: { secretId: "", secretKey: "", region: "cn-hangzhou" } as ServiceConfig,
};

export const configStore = writable<CloudConfig>(DEFAULT_CONFIG);

// initConfig 从后端加载配置并写入 store
export const initConfig = async (): Promise<void> => {
  try {
    const cfg = await GetConfig();
    if (cfg) {
      // 后端字段缺省时回填默认值，避免 undefined 进入响应式状态
      const normalized: CloudConfig = {
        ...DEFAULT_CONFIG,
        ...cfg,
        tencent: cfg.tencent || DEFAULT_CONFIG.tencent,
        aliyun: cfg.aliyun || DEFAULT_CONFIG.aliyun,
      };
      if (normalized.isDark === undefined) normalized.isDark = true;
      if (normalized.compareMode === undefined) normalized.compareMode = false;
      if (normalized.clipboardWatch === undefined) normalized.clipboardWatch = false;
      if (!Array.isArray(normalized.compareEngines) || normalized.compareEngines.length === 0) {
        normalized.compareEngines = ["tencent", "aliyun"];
      }
      configStore.set(normalized);
    }
  } catch (e) {
    console.error("初始化配置失败:", e);
  }
};

// updateAndSaveConfig 修改单个字段后立即同步到后端
// 返回 Promise，便于调用方感知保存是否成功
export const updateAndSaveConfig = async (
  partialKey: keyof CloudConfig,
  value: unknown
): Promise<void> => {
  let next: CloudConfig | undefined;
  configStore.update((curr) => {
    next = { ...curr, [partialKey]: value } as CloudConfig;
    return next;
  });
  try {
    if (next) {
      await SaveConfig(next as any);
    }
  } catch (e) {
    console.error("保存配置失败:", partialKey, e);
    throw e; // 向上抛出，调用方可决定如何提示
  }
};
