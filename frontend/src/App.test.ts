import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import App from "./App.svelte";
import { ARIA_SHORTCUTS } from "./lib/shortcuts";

describe("standalone app", () => {
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
});
