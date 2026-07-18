import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import History from "./History.svelte";
import type { HistoryEntry } from "./types";

interface HistoryProps {
  open: boolean;
  history: HistoryEntry[];
  onClose: () => void;
  onClear: () => void;
  onSelect: (entry: HistoryEntry) => void;
}

const history: HistoryEntry[] = [
  { id: 2, input: "hello world", output: "你好，世界", source: "auto", target: "zh", time: "07/18 20:10" },
  { id: 1, input: "再见", output: "goodbye", source: "zh", target: "en", time: "07/18 20:00" },
];

function renderHistory(overrides: Partial<HistoryProps> = {}) {
  const props: HistoryProps = {
    open: true,
    history,
    onClose: vi.fn(),
    onClear: vi.fn(),
    onSelect: vi.fn(),
    ...overrides,
  };
  render(History, props);
  return props;
}

describe("history drawer", () => {
  it("shows localized language routes and filters source and translated text", async () => {
    renderHistory();

    expect(screen.getByText("2 条记录")).toBeInTheDocument();
    expect(screen.getByText("自动识别")).toBeInTheDocument();
    expect(screen.getAllByText("中文")).toHaveLength(2);

    const search = screen.getByRole("textbox", { name: "搜索翻译记录" });
    await waitFor(() => expect(search).toHaveFocus());
    await fireEvent.input(search, { target: { value: "goodbye" } });
    expect(screen.getByText("1 / 2 条记录")).toBeInTheDocument();
    expect(screen.getByText("再见")).toBeInTheDocument();
    expect(screen.queryByText("hello world")).not.toBeInTheDocument();

    await fireEvent.input(search, { target: { value: "missing" } });
    expect(screen.getByText("0 / 2 条记录")).toBeInTheDocument();
    expect(screen.getByText("没有匹配记录")).toBeInTheDocument();
    await fireEvent.click(screen.getByText("清除搜索", { selector: "button" }));
    expect(screen.getByText("2 条记录")).toBeInTheDocument();
  });

  it("selects entries and keeps destructive clearing behind confirmation", async () => {
    const props = renderHistory();
    const entryButton = screen.getByText("hello world").closest("button");
    expect(entryButton).not.toBeNull();
    await fireEvent.click(entryButton!);
    expect(props.onSelect).toHaveBeenCalledWith(history[0]);

    const clearButton = screen.getByRole("button", { name: "清空历史记录" });
    await fireEvent.click(clearButton);
    const cancelButton = screen.getByRole("button", { name: "取消" });
    await waitFor(() => expect(cancelButton).toHaveFocus());
    await fireEvent.keyDown(screen.getByRole("dialog", { name: "历史记录" }), { key: "Escape" });
    expect(screen.queryByRole("button", { name: "确认清空" })).not.toBeInTheDocument();
    expect(props.onClose).not.toHaveBeenCalled();
    await waitFor(() => expect(clearButton).toHaveFocus());

    await fireEvent.click(clearButton);
    await fireEvent.click(screen.getByRole("button", { name: "确认清空" }));
    expect(props.onClear).toHaveBeenCalledOnce();
  });

  it("closes on Escape when no confirmation is active", async () => {
    const props = renderHistory();
    await fireEvent.keyDown(screen.getByRole("dialog", { name: "历史记录" }), { key: "Escape" });
    expect(props.onClose).toHaveBeenCalledOnce();
  });
});
