import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import App from "./App.svelte";

describe("standalone app", () => {
  it("renders with the mock bridge instead of a blank Wails runtime error", async () => {
    render(App);
    expect(await screen.findByLabelText("应用侧边栏")).toBeInTheDocument();
    expect(screen.getByText("SimpleTranslate")).toBeInTheDocument();
    expect(screen.getByText("预览")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "翻译语言" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "默认翻译引擎" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "自动翻译" })).toHaveAttribute("aria-pressed", "true");
    expect(screen.queryByRole("button", { name: "收起侧边栏" })).not.toBeInTheDocument();
  });
});
