import { afterEach, describe, expect, it, vi } from "vitest";
import { createClipboardWatcher } from "./clipboard";

afterEach(() => vi.useRealTimers());

describe("clipboard watcher", () => {
  it("uses the current clipboard as a baseline and emits later changes", async () => {
    vi.useFakeTimers();
    const onText = vi.fn();
    const getText = vi.fn()
      .mockResolvedValueOnce("existing")
      .mockResolvedValueOnce("new text");
    const watcher = createClipboardWatcher({ getText, onText, intervalMs: 100 });

    watcher.start();
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(100);

    expect(onText).toHaveBeenCalledOnce();
    expect(onText).toHaveBeenCalledWith("new text");
    watcher.stop();
  });

  it("ignores a clipboard read that finishes after the watcher is stopped", async () => {
    vi.useFakeTimers();
    let finishPoll: ((text: string) => void) | undefined;
    const getText = vi.fn()
      .mockResolvedValueOnce("existing")
      .mockImplementationOnce(() => new Promise<string>((resolve) => { finishPoll = resolve; }));
    const onText = vi.fn();
    const watcher = createClipboardWatcher({ getText, onText, intervalMs: 100 });

    watcher.start();
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(100);
    expect(getText).toHaveBeenCalledTimes(2);

    watcher.stop();
    finishPoll?.("late text");
    await Promise.resolve();

    expect(onText).not.toHaveBeenCalled();
  });

  it("does not let an in-flight read overwrite an explicit baseline", async () => {
    vi.useFakeTimers();
    let finishPoll: ((text: string) => void) | undefined;
    const getText = vi.fn()
      .mockResolvedValueOnce("existing")
      .mockImplementationOnce(() => new Promise<string>((resolve) => { finishPoll = resolve; }));
    const onText = vi.fn();
    const watcher = createClipboardWatcher({ getText, onText, intervalMs: 100 });

    watcher.start();
    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(100);
    watcher.setBaseline("copied by app");
    finishPoll?.("existing");
    await Promise.resolve();

    expect(onText).not.toHaveBeenCalled();
    watcher.stop();
  });
});
