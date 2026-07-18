import { fireEvent, render, screen } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import { DEFAULT_CONFIG } from "./bridge";
import Config from "./Config.svelte";

describe("settings draft", () => {
  it("tests draft credentials without saving them", async () => {
    const onSave = vi.fn(async () => undefined);
    const onTest = vi.fn(async () => undefined);
    render(Config, {
      open: true,
      config: { ...DEFAULT_CONFIG, tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" } },
      onClose: vi.fn(),
      onSave,
      onTest,
    });

    await fireEvent.click(screen.getByRole("button", { name: "测试腾讯混元连接" }));
    expect(onTest).toHaveBeenCalledWith("tencent", expect.objectContaining({ secretKey: "sk-test" }));
    expect(onSave).not.toHaveBeenCalled();
  });

  it("cancels without persisting edits", async () => {
    const onSave = vi.fn(async () => undefined);
    const onClose = vi.fn();
    render(Config, { open: true, config: DEFAULT_CONFIG, onClose, onSave, onTest: vi.fn(async () => undefined) });
    await fireEvent.input(screen.getByPlaceholderText("TokenHub API Key (sk-...)"), { target: { value: "draft" } });
    await fireEvent.click(screen.getByRole("button", { name: "取消" }));
    expect(onClose).toHaveBeenCalled();
    expect(onSave).not.toHaveBeenCalled();
  });
});
