// 历史记录工具函数：去重、构造条目、上限裁剪。
// 与 App.svelte 解耦，便于在测试或其他组件中复用。
import type { HistoryEntry } from "./types";

// 历史记录最大条数，与后端持久化上限保持一致
export const HISTORY_LIMIT = 200;

// formatHistoryTime 统一历史记录的时间格式（月/日 时:分）
export function formatHistoryTime(date: Date = new Date()): string {
  return date.toLocaleString([], {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

// isDuplicateRecent 判断新条目是否与最近一条重复（相同输入 + 语种对）
// 返回 true 表示应跳过写入。
export function isDuplicateRecent(
  history: HistoryEntry[] | undefined,
  input: string,
  src: string,
  tgt: string
): boolean {
  if (!history || history.length === 0) return false;
  const top = history[0];
  return top.input === input && top.source === src && top.target === tgt;
}

// createHistoryEntryOptions 构造历史条目的入参
export interface CreateHistoryEntryOptions {
  input: string;
  output: string;
  source: string;
  target: string;
  id?: number;
  time?: string;
}

// createHistoryEntry 构造一条历史记录，含时间戳与格式化时间
export function createHistoryEntry(opts: CreateHistoryEntryOptions): HistoryEntry {
  const { input, output, source, target } = opts;
  const id = opts.id ?? Date.now();
  const time = opts.time ?? formatHistoryTime();
  return { id, input, output, source, target, time };
}

// prependHistory 在历史记录头部插入新条目，按上限裁剪。
// 重复条目由调用方先用 isDuplicateRecent 判断后再调用此函数。
export function prependHistory(
  history: HistoryEntry[],
  entry: HistoryEntry,
  limit: number = HISTORY_LIMIT
): HistoryEntry[] {
  return [entry, ...history].slice(0, limit);
}
