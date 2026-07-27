import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import { DEFAULT_CONFIG } from "./bridge";
import Config from "./Config.svelte";

describe("settings draft", () => {
  it("disables browser autofill for service credentials", () => {
    const { container } = render(Config, {
      open: true,
      config: DEFAULT_CONFIG,
      onClose: vi.fn(),
      onSave: vi.fn(async () => undefined),
      onTest: vi.fn(async () => undefined),
    });

    const inputs = [...container.querySelectorAll("input")];
    expect(inputs).toHaveLength(6);
    for (const input of inputs) expect(input).toHaveAttribute("autocomplete", "off");
  });

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

  it("tests Baidu draft credentials and keeps the selected domain", async () => {
    const onSave = vi.fn(async () => undefined);
    const onTest = vi.fn(async () => undefined);
    render(Config, {
      open: true,
      config: {
        ...DEFAULT_CONFIG,
        baidu: { appId: "baidu-app", secretKey: "baidu-key", domain: "general" },
      },
      onClose: vi.fn(),
      onSave,
      onTest,
    });

    await fireEvent.change(screen.getByRole("combobox", { name: "百度翻译领域" }), { target: { value: "it" } });
    await fireEvent.click(screen.getByRole("button", { name: "测试百度翻译连接" }));
    expect(onTest).toHaveBeenCalledWith("baidu", {
      appId: "baidu-app",
      secretKey: "baidu-key",
      domain: "it",
    });
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

  it("discards a connection result after the tested credentials change", async () => {
    let finishTest: (() => void) | undefined;
    const onTest = vi.fn(() => new Promise<void>((resolve) => { finishTest = resolve; }));
    render(Config, {
      open: true,
      config: { ...DEFAULT_CONFIG, tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-old" } },
      onClose: vi.fn(),
      onSave: vi.fn(async () => undefined),
      onTest,
    });

    await fireEvent.click(screen.getByRole("button", { name: "测试腾讯混元连接" }));
    await fireEvent.input(screen.getByPlaceholderText("TokenHub API Key (sk-...)"), { target: { value: "sk-new" } });
    finishTest?.();

    await waitFor(() => expect(screen.getByRole("button", { name: "测试腾讯混元连接" })).toBeEnabled());
    expect(screen.queryByText("连接成功")).not.toBeInTheDocument();
  });

  it("prevents edits and closing while a save is in progress", async () => {
    let finishSave: (() => void) | undefined;
    const onClose = vi.fn();
    const onSave = vi.fn(() => new Promise<void>((resolve) => { finishSave = resolve; }));
    render(Config, {
      open: true,
      config: DEFAULT_CONFIG,
      onClose,
      onSave,
      onTest: vi.fn(async () => undefined),
    });

    await fireEvent.click(screen.getByRole("button", { name: "保存配置" }));

    expect((screen.getByRole("main") as HTMLElement & { inert: boolean }).inert).toBe(true);
    expect(screen.getByRole("button", { name: "取消" })).toBeDisabled();
    for (const button of screen.getAllByRole("button", { name: "关闭偏好设置" })) {
      expect(button).toBeDisabled();
    }
    expect(onClose).not.toHaveBeenCalled();

    const escapedShortcut = vi.fn();
    window.addEventListener("keydown", escapedShortcut);
    const shortcut = new KeyboardEvent("keydown", {
      key: ",",
      ctrlKey: true,
      bubbles: true,
      cancelable: true,
    });
    screen.getByRole("dialog", { name: "偏好设置" }).dispatchEvent(shortcut);
    window.removeEventListener("keydown", escapedShortcut);
    expect(escapedShortcut).not.toHaveBeenCalled();

    const tab = new KeyboardEvent("keydown", { key: "Tab", bubbles: true, cancelable: true });
    screen.getByRole("dialog", { name: "偏好设置" }).dispatchEvent(tab);
    expect(tab.defaultPrevented).toBe(true);

    finishSave?.();
    await waitFor(() => expect(onClose).toHaveBeenCalledOnce());
  });
});
