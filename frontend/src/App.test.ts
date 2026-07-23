import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import App from "./App.svelte";
import { DEFAULT_CONFIG, desktopBridge } from "./lib/bridge";
import { ARIA_SHORTCUTS } from "./lib/shortcuts";

describe("standalone app", () => {
  beforeEach(async () => {
    vi.restoreAllMocks();
    await desktopBridge.saveConfig(DEFAULT_CONFIG);
    await desktopBridge.saveHistory([]);
  });

  afterEach(() => vi.restoreAllMocks());

  it("renders with the mock bridge instead of a blank Wails runtime error", async () => {
    render(App);
    expect(await screen.findByLabelText("应用侧边栏")).toBeInTheDocument();
    expect(screen.getByText("SimpleTranslate")).toBeInTheDocument();
    expect(screen.getByText("预览")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "翻译语言" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "默认翻译引擎" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "自动翻译" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("textbox", { name: "原文" })).toHaveAttribute("aria-keyshortcuts", ARIA_SHORTCUTS.focusInput);
    expect(screen.getByRole("button", { name: "识别语言后即可交换" })).toHaveAttribute("aria-keyshortcuts", ARIA_SHORTCUTS.swapLanguages);
    expect(screen.getByRole("button", { name: "翻译" })).toHaveAttribute("aria-keyshortcuts", ARIA_SHORTCUTS.translate);
    expect(screen.getByRole("button", { name: "打开历史记录" })).toHaveAttribute("aria-keyshortcuts", ARIA_SHORTCUTS.toggleHistory);
    expect(screen.getByRole("button", { name: "切换深浅主题" })).toHaveAttribute("aria-keyshortcuts", ARIA_SHORTCUTS.toggleTheme);
    expect(screen.getByRole("button", { name: "打开设置" })).toHaveAttribute("aria-keyshortcuts", ARIA_SHORTCUTS.toggleSettings);
    expect(screen.queryByRole("button", { name: "收起侧边栏" })).not.toBeInTheDocument();
  });

  it("normalizes and persists the language route restored from history", async () => {
    await desktopBridge.saveHistory([{
      id: 1,
      input: "legacy input",
      output: "legacy output",
      source: "ja",
      target: "ko",
      time: "2026-07-22 19:00",
    }]);
    const saveConfig = vi.spyOn(desktopBridge, "saveConfig");
    render(App);

    await fireEvent.click(await screen.findByRole("button", { name: "打开历史记录" }));
    const entryText = await screen.findByText("legacy input");
    await fireEvent.click(entryText.closest("button")!);

    expect(screen.getByRole("textbox", { name: "原文" })).toHaveValue("legacy input");
    expect(screen.getByRole("textbox", { name: "译文" })).toHaveValue("legacy output");
    await waitFor(() => expect(saveConfig).toHaveBeenCalledWith(expect.objectContaining({
      sourceLanguage: "jp",
      targetLanguage: "kr",
    })));
  });

  it("restores history without automatically translating it again", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    await desktopBridge.saveHistory([{
      id: 2,
      input: "saved input",
      output: "saved output",
      source: "en",
      target: "zh",
      time: "2026-07-23 10:00",
    }]);
    const translateText = vi.spyOn(desktopBridge, "translateText");
    render(App);

    await fireEvent.click(await screen.findByRole("button", { name: "打开历史记录" }));
    await fireEvent.click((await screen.findByText("saved input")).closest("button")!);
    await new Promise((resolve) => setTimeout(resolve, 800));

    expect(translateText).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox", { name: "译文" })).toHaveValue("saved output");
  });

  it("copies translations through the desktop bridge", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      autoTranslate: false,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    const setClipboardText = vi.spyOn(desktopBridge, "setClipboardText");
    render(App);

    await fireEvent.input(await screen.findByRole("textbox", { name: "原文" }), { target: { value: "hello" } });
    await fireEvent.click(screen.getByRole("button", { name: /^翻译$/ }));
    await fireEvent.click(await screen.findByRole("button", { name: "复制译文" }));

    expect(setClipboardText).toHaveBeenCalledWith("你好");
  });

  it("exposes an immediate cancel action for an in-flight translation", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      autoTranslate: false,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    let finish: (() => void) | undefined;
    vi.spyOn(desktopBridge, "translateText").mockImplementation((request) => new Promise((resolve) => {
      finish = () => resolve({
        requestId: request.requestId,
        source: request.source,
        autoSrc: request.source,
        target: request.target,
        text: "late result",
      });
    }));
    const cancelTranslation = vi.spyOn(desktopBridge, "cancelTranslation").mockResolvedValue(true);
    render(App);

    await fireEvent.input(await screen.findByRole("textbox", { name: "原文" }), { target: { value: "hello" } });
    await fireEvent.click(screen.getByRole("button", { name: /^翻译$/ }));
    const cancelButton = await screen.findByRole("button", { name: "取消翻译" });
    expect(cancelButton).toHaveAttribute("aria-busy", "true");
    await fireEvent.click(cancelButton);

    expect(cancelTranslation).toHaveBeenCalledOnce();
    expect(screen.getByText("已取消")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "取消翻译" })).not.toBeInTheDocument();
    finish?.();
  });

  it("clears the translation together with the source text", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      autoTranslate: false,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    render(App);

    await fireEvent.input(await screen.findByRole("textbox", { name: "原文" }), { target: { value: "hello" } });
    await fireEvent.click(screen.getByRole("button", { name: /^翻译$/ }));
    expect(await screen.findByRole("textbox", { name: "译文" })).toHaveValue("你好");

    await fireEvent.click(screen.getByRole("button", { name: "清空原文" }));

    expect(screen.getByRole("textbox", { name: "原文" })).toHaveValue("");
    expect(screen.queryByRole("textbox", { name: "译文" })).not.toBeInTheDocument();
    expect(screen.getByText("等待翻译")).toBeInTheDocument();
  });

  it("does not change persisted settings behind an open settings draft", async () => {
    const saveConfig = vi.spyOn(desktopBridge, "saveConfig");
    render(App);
    await fireEvent.click(await screen.findByRole("button", { name: "打开设置" }));

    window.dispatchEvent(new KeyboardEvent("keydown", {
      key: "m",
      ctrlKey: true,
      bubbles: true,
      cancelable: true,
    }));

    expect(document.querySelector(".app-shell")).not.toHaveClass("light-mode");
    expect(saveConfig).not.toHaveBeenCalled();
  });

  it("rolls language selectors back when persistence fails", async () => {
    vi.spyOn(desktopBridge, "saveConfig").mockRejectedValueOnce(new Error("disk full"));
    render(App);
    const target = await screen.findByRole("combobox", { name: "目标语言" });

    await fireEvent.change(target, { target: { value: "fr" } });

    await waitFor(() => expect(target).toHaveValue("zh"));
    expect(screen.getByRole("alert")).toHaveTextContent("设置保存失败");
  });

  it("automatically retranslates when the language route changes", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    const translateText = vi.spyOn(desktopBridge, "translateText");
    render(App);
    const input = await screen.findByRole("textbox", { name: "原文" });
    const target = screen.getByRole("combobox", { name: "目标语言" });

    await fireEvent.input(input, { target: { value: "hello" } });
    await fireEvent.click(screen.getByRole("button", { name: /^翻译$/ }));
    await waitFor(() => expect(translateText).toHaveBeenCalledOnce());
    await fireEvent.change(target, { target: { value: "fr" } });

    await waitFor(() => expect(translateText).toHaveBeenCalledTimes(2), { timeout: 1800 });
    expect(translateText.mock.calls[1][0].target).toBe("fr");
  });

  it("does not retranslate after the backend adjusts the target language", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    const translateText = vi.spyOn(desktopBridge, "translateText").mockImplementation(async (request) => ({
      requestId: request.requestId,
      source: request.source,
      autoSrc: "zh",
      target: "en",
      text: "Hello",
    }));
    render(App);

    await fireEvent.input(await screen.findByRole("textbox", { name: "原文" }), { target: { value: "你好" } });
    await fireEvent.click(screen.getByRole("button", { name: /^翻译$/ }));
    await waitFor(() => expect(screen.getByRole("combobox", { name: "目标语言" })).toHaveValue("en"));
    await new Promise((resolve) => setTimeout(resolve, 800));

    expect(translateText).toHaveBeenCalledOnce();
  });

  it("cancels a pending automatic translation when the mode is disabled", async () => {
    await desktopBridge.saveConfig({
      ...DEFAULT_CONFIG,
      tencent: { ...DEFAULT_CONFIG.tencent, secretKey: "sk-test" },
    });
    const translateText = vi.spyOn(desktopBridge, "translateText");
    render(App);
    const autoTranslate = await screen.findByRole("button", { name: "自动翻译" });

    await fireEvent.input(screen.getByRole("textbox", { name: "原文" }), { target: { value: "hello" } });
    await fireEvent.click(autoTranslate);
    await waitFor(() => expect(autoTranslate).toHaveAttribute("aria-pressed", "false"));
    await new Promise((resolve) => setTimeout(resolve, 800));

    expect(translateText).not.toHaveBeenCalled();
  });
});
