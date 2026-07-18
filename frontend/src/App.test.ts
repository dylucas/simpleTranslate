import { render, screen } from "@testing-library/svelte";
import { describe, expect, it } from "vitest";
import App from "./App.svelte";

describe("standalone app", () => {
  it("renders with the mock bridge instead of a blank Wails runtime error", async () => {
    render(App);
    expect(await screen.findByLabelText("应用侧边栏")).toBeInTheDocument();
    expect(screen.getByText("SimpleTranslate")).toBeInTheDocument();
    expect(screen.getByText("预览")).toBeInTheDocument();
  });
});
