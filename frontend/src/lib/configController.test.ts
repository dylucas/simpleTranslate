import { get } from "svelte/store";
import { describe, expect, it, vi } from "vitest";
import { createMockBridge, DEFAULT_CONFIG, type DesktopBridge } from "./bridge";
import { createConfigController } from "./configController";
import type { CloudConfig } from "./types";

describe("config controller", () => {
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
