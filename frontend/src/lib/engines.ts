import type { CloudConfig, EngineId } from "./types";

export const ENGINE_IDS: EngineId[] = ["tencent", "aliyun", "baidu"];

export function isEngineId(value: unknown): value is EngineId {
  return value === "tencent" || value === "aliyun" || value === "baidu";
}

export function engineLabel(engine: EngineId, full = false): string {
  if (engine === "tencent") return full ? "腾讯混元" : "混元";
  if (engine === "aliyun") return full ? "阿里云 MT" : "阿里云";
  return full ? "百度翻译" : "百度";
}

export function isEngineConfigured(config: CloudConfig, engine: EngineId): boolean {
  if (engine === "tencent") return Boolean(config.tencent.secretKey.trim());
  if (engine === "aliyun") {
    return Boolean(config.aliyun.secretId.trim() && config.aliyun.secretKey.trim());
  }
  return Boolean(config.baidu.appId.trim() && config.baidu.secretKey.trim());
}
