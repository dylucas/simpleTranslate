import { get } from "svelte/store";
import { afterEach, describe, expect, it, vi } from "vitest";
import { createMockBridge, type DesktopBridge } from "./bridge";
import { createTranslateController } from "./translateController";
import type { EngineTranslateResult, HistoryEntry, MultiTranslateResult } from "./types";

function createHarness(bridge: DesktopBridge, compareMode = false) {
  let input = "hello";
  let target = "zh";
  let history: HistoryEntry[] = [];
  const controller = createTranslateController({
    bridge,
    getInput: () => input,
    getSource: () => "en",
    getTarget: () => target,
    getActiveEngine: () => "tencent",
    getCompareMode: () => compareMode,
    getCompareEngines: () => ["tencent", "aliyun"],
    getHistory: () => history,
    setHistory: (update) => { history = update(history); },
    setTarget: (value) => { target = value; },
  });
  return {
    controller,
    setInput: (value: string) => { input = value; },
    setTarget: (value: string) => { target = value; },
    getHistory: () => history,
  };
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

  it("cancels a pending automatic request when automatic translation is disabled", async () => {
    vi.useFakeTimers();
    const bridge = createMockBridge();
    const spy = vi.spyOn(bridge, "translateText");
    const harness = createHarness(bridge);

    harness.controller.handleAutoTranslate("hello");
    harness.controller.cancelAutoTranslate();
    await vi.advanceTimersByTimeAsync(700);

    expect(spy).not.toHaveBeenCalled();
    harness.controller.destroy();
  });

  it("cancels an active request without losing the previous output", async () => {
    let finish: ((value: Awaited<ReturnType<DesktopBridge["translateText"]>>) => void) | undefined;
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      cancelTranslation: vi.fn(async () => true),
      translateText: () => new Promise((resolve) => { finish = resolve; }),
    };
    const harness = createHarness(bridge);
    harness.controller.setOutput("previous output");

    const pending = harness.controller.translate();
    const requestId = get(harness.controller.state).activeRequestId;
    harness.controller.cancel();

    expect(bridge.cancelTranslation).toHaveBeenCalledWith(requestId);
    expect(get(harness.controller.state)).toMatchObject({
      isProcessing: false,
      status: "已取消",
      output: "previous output",
      activeRequestId: "",
      compareLoadingEngines: {},
    });

    finish?.({ requestId, source: "en", autoSrc: "en", target: "zh", text: "late result" });
    await pending;
    expect(get(harness.controller.state).output).toBe("previous output");
    expect(harness.getHistory()).toHaveLength(0);
    harness.controller.destroy();
  });

  it("cancels the previous request before translating the latest auto input", async () => {
    vi.useFakeTimers();
    let finishFirst: ((value: Awaited<ReturnType<DesktopBridge["translateText"]>>) => void) | undefined;
    const calls: string[] = [];
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      cancelTranslation: vi.fn(async () => true),
      async translateText(request) {
        calls.push(request.text);
        if (calls.length === 1) {
          return new Promise((resolve) => { finishFirst = resolve; });
        }
        return {
          requestId: request.requestId,
          source: request.source,
          autoSrc: request.source,
          target: request.target,
          text: "latest output",
        };
      },
    };
    const harness = createHarness(bridge);
    const first = harness.controller.translate();
    const firstRequestId = get(harness.controller.state).activeRequestId;
    harness.setInput("latest");
    harness.controller.handleAutoTranslate("latest");

    await vi.advanceTimersByTimeAsync(700);
    await vi.waitFor(() => expect(calls).toEqual(["hello", "latest"]));
    expect(bridge.cancelTranslation).toHaveBeenCalledWith(firstRequestId);
    expect(get(harness.controller.state).output).toBe("latest output");

    finishFirst?.({ requestId: firstRequestId, source: "en", autoSrc: "en", target: "zh", text: "stale output" });
    await first;
    expect(get(harness.controller.state).output).toBe("latest output");
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

  it("ignores streaming events after request parameters change", async () => {
    let eventHandler: ((result: EngineTranslateResult) => void) | undefined;
    let finish: ((result: MultiTranslateResult) => void) | undefined;
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      onEngineResult(handler) { eventHandler = handler; return () => undefined; },
      translateMulti: () => new Promise((resolve) => { finish = resolve; }),
    };
    const harness = createHarness(bridge, true);
    const pending = harness.controller.translate();
    const requestId = get(harness.controller.state).activeRequestId;

    harness.setTarget("fr");
    eventHandler?.({ requestId, engine: "tencent", text: "stale stream" });
    expect(get(harness.controller.state).compareOutputs.tencent).toBeUndefined();

    finish?.({ requestId, source: "en", autoSrc: "en", target: "zh", results: {
      tencent: { requestId, engine: "tencent", text: "stale final" },
    } });
    await pending;
    harness.controller.destroy();
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

  it("discards an in-flight response and restarts when request parameters change", async () => {
    let finishFirst: ((value: Awaited<ReturnType<DesktopBridge["translateText"]>>) => void) | undefined;
    const calls: string[] = [];
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      async translateText(request) {
        calls.push(request.target);
        if (calls.length === 1) {
          return new Promise((resolve) => { finishFirst = resolve; });
        }
        return {
          requestId: request.requestId,
          source: request.source,
          autoSrc: request.source,
          target: request.target,
          text: "bonjour",
        };
      },
    };
    const harness = createHarness(bridge);

    const first = harness.controller.translate();
    const firstRequestId = get(harness.controller.state).activeRequestId;
    harness.setTarget("fr");
    finishFirst?.({
      requestId: firstRequestId,
      source: "en",
      autoSrc: "en",
      target: "zh",
      text: "stale",
    });
    await first;

    await vi.waitFor(() => expect(calls).toEqual(["zh", "fr"]));
    await vi.waitFor(() => expect(get(harness.controller.state).isProcessing).toBe(false));
    expect(get(harness.controller.state).output).toBe("bonjour");
    expect(harness.getHistory()).toHaveLength(1);
    harness.controller.destroy();
  });

  it("does not treat an empty compare response as a successful translation", async () => {
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      async translateMulti(request) {
        return {
          requestId: request.requestId,
          source: request.source,
          autoSrc: request.source,
          target: request.target,
          results: {},
        };
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
    controller.setOutput("previous result");

    await controller.translate();

    expect(get(controller.state).status).toBe("翻译失败");
    expect(get(controller.state).errorToast?.msg).toContain("未返回结果");
    expect(history).toHaveLength(0);
    controller.destroy();
  });

  it("ignores an in-flight response after the controller is destroyed", async () => {
    let finish: ((value: Awaited<ReturnType<DesktopBridge["translateText"]>>) => void) | undefined;
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      translateText: () => new Promise((resolve) => { finish = resolve; }),
    };
    const harness = createHarness(bridge);
    const pending = harness.controller.translate();
    const requestId = get(harness.controller.state).activeRequestId;

    harness.controller.destroy();
    finish?.({ requestId, source: "en", autoSrc: "en", target: "zh", text: "late result" });
    await pending;

    expect(get(harness.controller.state).output).toBe("");
    expect(harness.getHistory()).toHaveLength(0);
  });

  it("clears output and invalidates an in-flight response", async () => {
    let finish: ((value: Awaited<ReturnType<DesktopBridge["translateText"]>>) => void) | undefined;
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      translateText: () => new Promise((resolve) => { finish = resolve; }),
    };
    const harness = createHarness(bridge);
    const pending = harness.controller.translate();
    const requestId = get(harness.controller.state).activeRequestId;

    harness.controller.setOutput("old output");
    harness.controller.clear();
    finish?.({ requestId, source: "en", autoSrc: "en", target: "zh", text: "late result" });
    await pending;

    expect(get(harness.controller.state)).toMatchObject({
      isProcessing: false,
      output: "",
      compareOutputs: {},
      status: "准备就绪",
      activeRequestId: "",
    });
    expect(harness.getHistory()).toHaveLength(0);
    harness.controller.destroy();
  });
});
