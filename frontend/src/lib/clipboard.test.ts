import { afterEach, describe, expect, it, vi } from "vitest";
import { createClipboardReader } from "./clipboard";

afterEach(() => vi.useRealTimers());

describe("clipboard reader", () => {
  it("reads once without starting background polling", async () => {
    vi.useFakeTimers();
    const getText = vi.fn().mockResolvedValue("current text");
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText });

    await reader.read();
    await vi.advanceTimersByTimeAsync(10_000);

    expect(getText).toHaveBeenCalledOnce();
    expect(onText).toHaveBeenCalledOnce();
    expect(onText).toHaveBeenCalledWith("current text");
  });

  it("coalesces concurrent reads", async () => {
    let finishRead: ((text: string) => void) | undefined;
    const getText = vi.fn(() => new Promise<string>((resolve) => { finishRead = resolve; }));
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText });

    const first = reader.read();
    const second = reader.read();
    expect(getText).toHaveBeenCalledOnce();
    expect(second).toBe(first);

    finishRead?.("new text");
    await first;
    expect(onText).toHaveBeenCalledWith("new text");
  });

  it("ignores a clipboard read that finishes after cancellation", async () => {
    let finishRead: ((text: string) => void) | undefined;
    const getText = vi.fn(() => new Promise<string>((resolve) => { finishRead = resolve; }));
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText });

    const pending = reader.read();
    reader.cancel();
    finishRead?.("late text");
    await pending;

    expect(onText).not.toHaveBeenCalled();
  });

  it("does not let an in-flight read overwrite an explicit baseline", async () => {
    let finishRead: ((text: string) => void) | undefined;
    const getText = vi.fn(() => new Promise<string>((resolve) => { finishRead = resolve; }));
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText });

    const pending = reader.read();
    reader.setBaseline("copied by app");
    finishRead?.("older clipboard text");
    await pending;

    expect(onText).not.toHaveBeenCalled();
  });

  it("uses the baseline and last handled text to prevent feedback and duplicates", async () => {
    const getText = vi.fn()
      .mockResolvedValueOnce("copied by app")
      .mockResolvedValueOnce("new text")
      .mockResolvedValueOnce("new text");
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText });
    reader.setBaseline("copied by app");

    await reader.read();
    await reader.read();
    await reader.read();

    expect(getText).toHaveBeenCalledTimes(3);
    expect(onText).toHaveBeenCalledOnce();
    expect(onText).toHaveBeenCalledWith("new text");
  });

  it("rejects blank and oversized clipboard text", async () => {
    const getText = vi.fn()
      .mockResolvedValueOnce("   ")
      .mockResolvedValueOnce("中".repeat(2001));
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText, maxTextLength: 6000 });

    await reader.read();
    await reader.read();

    expect(onText).not.toHaveBeenCalled();
  });

  it("retries text skipped while translation is busy", async () => {
    let busy = true;
    const getText = vi.fn().mockResolvedValue("new text");
    const onText = vi.fn();
    const reader = createClipboardReader({ getText, onText, isBusy: () => busy });

    await reader.read();
    busy = false;
    await reader.read();

    expect(getText).toHaveBeenCalledTimes(2);
    expect(onText).toHaveBeenCalledOnce();
    expect(onText).toHaveBeenCalledWith("new text");
  });
});
