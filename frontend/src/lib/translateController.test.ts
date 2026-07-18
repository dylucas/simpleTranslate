import { get } from "svelte/store";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createMockBridge, type DesktopBridge } from "./bridge";
import { createTranslateController } from "./translateController";
import type { EngineTranslateResult, HistoryEntry, MultiTranslateResult } from "./types";

function createHarness(bridge: DesktopBridge) {
  let input = "hello";
  let history: HistoryEntry[] = [];
  const controller = createTranslateController({
    bridge,
    getInput: () => input,
    getSource: () => "en",
    getTarget: () => "zh",
    getActiveEngine: () => "tencent",
    getCompareMode: () => false,
    getCompareEngines: () => ["tencent", "aliyun"],
    getHistory: () => history,
    setHistory: (update) => { history = update(history); },
    setTarget: vi.fn(),
  });
  return { controller, setInput: (value: string) => { input = value; }, getHistory: () => history };
}

afterEach(() => vi.useRealTimers());

describe("translate controller", () => {
  it("debounces automatic translation", async () => {
    vi.useFakeTimers();
    const bridge = createMockBridge();
    const spy = vi.spyOn(bridge, "translateText");
    const harness = createHarness(bridge);

    harness.controller.handleAutoTranslate("hello");
    await vi.advanceTimersByTimeAsync(699);
    expect(spy).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(spy).toHaveBeenCalledOnce();
    harness.controller.destroy();
  });

  it("cancels a pending automatic request after manual translation", async () => {
    vi.useFakeTimers();
    const bridge = createMockBridge();
    const spy = vi.spyOn(bridge, "translateText");
    const harness = createHarness(bridge);

    harness.controller.handleAutoTranslate("hello");
    await harness.controller.translate();
    await vi.advanceTimersByTimeAsync(700);
    expect(spy).toHaveBeenCalledOnce();
    harness.controller.destroy();
  });

  it("ignores engine events from an older request", async () => {
    let eventHandler: ((result: EngineTranslateResult) => void) | undefined;
    let finish: ((result: MultiTranslateResult) => void) | undefined;
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      onEngineResult(handler) { eventHandler = handler; return () => undefined; },
      translateMulti: () => new Promise((resolve) => { finish = resolve; }),
    };
    let history: HistoryEntry[] = [];
    const controller = createTranslateController({
      bridge,
      getInput: () => "hello",
      getSource: () => "en",
      getTarget: () => "zh",
      getActiveEngine: () => "tencent",
      getCompareMode: () => true,
      getCompareEngines: () => ["tencent"],
      getHistory: () => history,
      setHistory: (update) => { history = update(history); },
      setTarget: vi.fn(),
    });

    const pending = controller.translate();
    const requestId = get(controller.state).activeRequestId;
    eventHandler?.({ requestId: "stale", engine: "tencent", text: "old" });
    expect(get(controller.state).compareOutputs.tencent).toBeUndefined();
    finish?.({ requestId, source: "en", autoSrc: "en", target: "zh", results: {
      tencent: { requestId, engine: "tencent", text: "你好" },
    } });
    await pending;
    expect(get(controller.state).output).toBe("你好");
    controller.destroy();
  });

  it("keeps a successful result when another engine fails", async () => {
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      async translateMulti(request) {
        return { requestId: request.requestId, source: "en", autoSrc: "en", target: "zh", results: {
          tencent: { requestId: request.requestId, engine: "tencent", text: "你好" },
          aliyun: { requestId: request.requestId, engine: "aliyun", text: "", error: "timeout", errorCode: "timeout" },
        } };
      },
    };
    let history: HistoryEntry[] = [];
    const controller = createTranslateController({
      bridge,
      getInput: () => "hello",
      getSource: () => "en",
      getTarget: () => "zh",
      getActiveEngine: () => "tencent",
      getCompareMode: () => true,
      getCompareEngines: () => ["tencent", "aliyun"],
      getHistory: () => history,
      setHistory: (update) => { history = update(history); },
      setTarget: vi.fn(),
    });
    await controller.translate();
    expect(get(controller.state).status).toBe("部分引擎失败");
    expect(get(controller.state).output).toBe("你好");
    expect(history).toHaveLength(1);
    controller.destroy();
  });
});
