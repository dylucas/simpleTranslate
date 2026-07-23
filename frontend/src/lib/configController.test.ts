import { get } from "svelte/store";
import { describe, expect, it, vi } from "vitest";
import { createMockBridge, DEFAULT_CONFIG, type DesktopBridge } from "./bridge";
import { createConfigController, normalizeConfig } from "./configController";
import type { CloudConfig } from "./types";

describe("config controller", () => {
  it("normalizes language aliases and rejects unsupported persisted values", () => {
    expect(normalizeConfig({ sourceLanguage: " JA ", targetLanguage: "ko" })).toMatchObject({
      sourceLanguage: "jp",
      targetLanguage: "kr",
    });
    expect(normalizeConfig({ sourceLanguage: "invalid", targetLanguage: "" })).toMatchObject({
      sourceLanguage: "auto",
      targetLanguage: "zh",
    });
  });

  it("coalesces queued writes and keeps the latest snapshot", async () => {
    const saved: CloudConfig[] = [];
    let releaseFirst: (() => void) | undefined;
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      saveConfig: vi.fn(async (config) => {
        saved.push(config);
        if (saved.length === 1) await new Promise<void>((resolve) => (releaseFirst = resolve));
      }),
    };
    const controller = createConfigController(bridge, vi.fn());
    await controller.load();

    const first = controller.patch("isDark", false);
    const second = controller.patch("sidebarCollapsed", true);
    expect(saved).toHaveLength(1);
    releaseFirst?.();
    await Promise.all([first, second]);

    expect(saved).toHaveLength(2);
    expect(saved[1].isDark).toBe(false);
    expect(saved[1].sidebarCollapsed).toBe(true);
  });

  it("waits for loading before applying a patch", async () => {
    let finishLoad: ((config: CloudConfig) => void) | undefined;
    const saveConfig = vi.fn(async () => undefined);
    const bridge: DesktopBridge = {
      ...createMockBridge(),
      getConfig: () => new Promise((resolve) => { finishLoad = resolve; }),
      saveConfig,
    };
    const controller = createConfigController(bridge, vi.fn());
    const loading = controller.load();
    const patching = controller.patch("isDark", false);

    expect(saveConfig).not.toHaveBeenCalled();
    finishLoad?.({
      ...DEFAULT_CONFIG,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "preserve-me" },
    });
    await loading;
    await patching;

    expect(saveConfig).toHaveBeenCalledWith(expect.objectContaining({
      isDark: false,
      tencent: expect.objectContaining({ secretKey: "preserve-me" }),
    }));
  });

  it("rolls back to the last persisted config after a save failure", async () => {
    const onError = vi.fn();
    const bridge: DesktopBridge = {
      ...createMockBridge(DEFAULT_CONFIG),
      saveConfig: vi.fn(async () => { throw new Error("disk full"); }),
    };
    const controller = createConfigController(bridge, onError);
    await controller.load();

    await expect(controller.patch("isDark", false)).rejects.toThrow();
    expect(get(controller).value.isDark).toBe(true);
    expect(onError).toHaveBeenCalledOnce();
  });
});
