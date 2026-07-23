import { afterEach, describe, expect, it, vi } from "vitest";
import { createHistoryPersistence } from "./historyPersistence";
import type { HistoryEntry } from "./types";

function entry(id: number): HistoryEntry {
  return { id, input: `input-${id}`, output: `output-${id}`, source: "en", target: "zh", time: "now" };
}

afterEach(() => vi.useRealTimers());

describe("history persistence", () => {
  it("debounces changes and writes only the latest snapshot", async () => {
    vi.useFakeTimers();
    const save = vi.fn(async () => undefined);
    const persistence = createHistoryPersistence(save, vi.fn(), 100);

    persistence.schedule([entry(1)]);
    persistence.schedule([entry(2)]);
    await vi.advanceTimersByTimeAsync(99);
    expect(save).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);

    expect(save).toHaveBeenCalledOnce();
    expect(save).toHaveBeenCalledWith([entry(2)]);
  });

  it("serializes writes so an older snapshot cannot overwrite a newer one", async () => {
    vi.useFakeTimers();
    let finishFirst: (() => void) | undefined;
    const saved: HistoryEntry[][] = [];
    const save = vi.fn(async (entries: HistoryEntry[]) => {
      saved.push(entries);
      if (saved.length === 1) await new Promise<void>((resolve) => { finishFirst = resolve; });
    });
    const persistence = createHistoryPersistence(save, vi.fn(), 100);

    persistence.schedule([entry(1)]);
    await vi.advanceTimersByTimeAsync(100);
    persistence.schedule([entry(2)]);
    await vi.advanceTimersByTimeAsync(100);
    expect(save).toHaveBeenCalledOnce();

    finishFirst?.();
    await persistence.flush();

    expect(saved).toEqual([[entry(1)], [entry(2)]]);
  });

  it("flushes a pending debounced snapshot immediately", async () => {
    vi.useFakeTimers();
    const save = vi.fn(async () => undefined);
    const persistence = createHistoryPersistence(save, vi.fn(), 100);
    persistence.schedule([entry(1)]);

    await persistence.flush();

    expect(save).toHaveBeenCalledWith([entry(1)]);
    await vi.advanceTimersByTimeAsync(100);
    expect(save).toHaveBeenCalledOnce();
  });
});
