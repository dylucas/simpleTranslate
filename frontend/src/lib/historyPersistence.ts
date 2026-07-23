import type { HistoryEntry } from "./types";

export interface HistoryPersistence {
  schedule(entries: HistoryEntry[]): void;
  flush(): Promise<void>;
}

function cloneHistory(entries: HistoryEntry[]): HistoryEntry[] {
  return entries.map((entry) => ({ ...entry }));
}

export function createHistoryPersistence(
  save: (entries: HistoryEntry[]) => Promise<void>,
  onError: () => void,
  delayMs = 400,
): HistoryPersistence {
  let timer: ReturnType<typeof setTimeout> | null = null;
  let scheduled: HistoryEntry[] | null = null;
  let queued: HistoryEntry[] | null = null;
  let draining: Promise<void> | null = null;

  function drain(): Promise<void> {
    if (draining) return draining;
    draining = (async () => {
      while (queued) {
        const next = queued;
        queued = null;
        try {
          await save(next);
        } catch {
          onError();
        }
      }
    })().finally(() => {
      draining = null;
    });
    return draining;
  }

  function enqueueScheduled(): Promise<void> {
    if (timer) clearTimeout(timer);
    timer = null;
    if (scheduled) {
      queued = scheduled;
      scheduled = null;
    }
    return queued ? drain() : draining ?? Promise.resolve();
  }

  function schedule(entries: HistoryEntry[]): void {
    scheduled = cloneHistory(entries);
    if (timer) clearTimeout(timer);
    timer = setTimeout(() => {
      timer = null;
      void enqueueScheduled();
    }, delayMs);
  }

  return {
    schedule,
    flush: enqueueScheduled,
  };
}
