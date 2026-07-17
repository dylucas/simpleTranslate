import { writable } from 'svelte/store';
import { GetConfig, SaveConfig } from "../../wailsjs/go/main/App";

// 初始状态包含 isDark
export const configStore = writable({
    isDark: true,
    sidebarCollapsed: false, // 新增：持久化侧边栏状态
    defaultEngine: 'tencent',
    // Multi-engine compare
    compareMode: false,
    compareEngines: ['tencent', 'aliyun'],
    // 剪贴板监听
    clipboardWatch: false,
    tencent: { secretId: "", secretKey: "", region: "" },
    aliyun: { secretId: "", secretKey: "", region: "cn-hangzhou" }
});

// 初始化
export const initConfig = async () => {
    try {
        const cfg = await GetConfig();
        if (cfg) {
            // 如果后端返回的配置里没有 isDark，给个默认值
            if (cfg.isDark === undefined) cfg.isDark = true;
            if (cfg.compareMode === undefined) cfg.compareMode = false;
            if (cfg.clipboardWatch === undefined) cfg.clipboardWatch = false;
            if (!Array.isArray(cfg.compareEngines) || cfg.compareEngines.length === 0) cfg.compareEngines = ['tencent', 'aliyun'];
            // @ts-ignore
            configStore.set(cfg);
        }
    } catch (e) {
        console.error("初始化配置失败:", e);
    }
};

// 快捷保存函数：用于只修改单个属性（如主题）时同步到后端
// 返回 Promise，便于调用方感知保存是否成功
export const updateAndSaveConfig = async (partialKey, value) => {
    let next;
    configStore.update(curr => {
        next = { ...curr, [partialKey]: value };
        return next;
    });
    try {
        // @ts-ignore
        await SaveConfig(next);
    } catch (e) {
        console.error("保存配置失败:", partialKey, e);
        throw e; // 向上抛出，调用方可决定如何提示
    }
};