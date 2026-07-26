import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { describe, expect, it, vi } from "vitest";
import History from "./History.svelte";
import { MAX_INPUT_BYTES } from "./textLimits";
import type { HistoryEntry } from "./types";

interface HistoryProps {
  open: boolean;
  queryHistory: (query: { query: string; offset: number; limit: number }) => Promise<{ entries: HistoryEntry[]; total: number; allTotal: number; hasMore: boolean }>;
  onClose: () => void;
  onClear: () => Promise<void>;
  onExport: () => Promise<boolean>;
  onError: (message: string) => void;
  onSelect: (entry: HistoryEntry) => void;
}

const history: HistoryEntry[] = [
  { id: 2, input: "hello world", output: "你好，世界", source: "auto", target: "zh", time: "07/18 20:10" },
  { id: 1, input: "再见", output: "goodbye", source: "zh", target: "en", time: "07/18 20:00" },
];

function renderHistory(overrides: Partial<HistoryProps> = {}) {
  const source = "history" in overrides ? (overrides as unknown as { history: HistoryEntry[] }).history : history;
  const props: HistoryProps = {
    open: true,
    queryHistory: vi.fn(async ({ query, offset, limit }) => {
      const needle = query.toLowerCase();
      const filtered = needle ? source.filter((item) => item.input.toLowerCase().includes(needle) || item.output.toLowerCase().includes(needle)) : source;
      return { entries: filtered.slice(offset, offset + limit), total: filtered.length, allTotal: source.length, hasMore: offset + limit < filtered.length };
    }),
    onClose: vi.fn(),
    onClear: vi.fn(async () => undefined),
    onExport: vi.fn(async () => true),
    onError: vi.fn(),
    onSelect: vi.fn(),
    ...overrides,
  };
  render(History, props);
  return props;
}

describe("history drawer", () => {
  it("shows localized language routes and filters source and translated text", async () => {
    renderHistory();

    expect(await screen.findByText("2 条记录")).toBeInTheDocument();
    expect(screen.getByText("自动识别")).toBeInTheDocument();
    expect(screen.getAllByText("中文")).toHaveLength(2);

    const search = screen.getByRole("textbox", { name: "搜索翻译记录" });
    await waitFor(() => expect(search).toHaveFocus());
    await fireEvent.input(search, { target: { value: "goodbye" } });
    expect(await screen.findByText("1 条匹配记录", {}, { timeout: 1000 })).toBeInTheDocument();
    expect(await screen.findByText("再见")).toBeInTheDocument();
    expect(screen.queryByText("hello world")).not.toBeInTheDocument();

    await fireEvent.input(search, { target: { value: "missing" } });
    expect(await screen.findByText("0 条匹配记录", {}, { timeout: 1000 })).toBeInTheDocument();
    expect(await screen.findByText("没有匹配记录")).toBeInTheDocument();
    await fireEvent.click(screen.getByText("清除搜索", { selector: "button" }));
    expect(await screen.findByText("2 条记录", {}, { timeout: 1000 })).toBeInTheDocument();
  });

  it("truncates search input to the backend UTF-8 byte limit", async () => {
    const queryHistory = vi.fn(async (_query: { query: string; offset: number; limit: number }) => (
      { entries: [], total: 0, allTotal: 0, hasMore: false }
    ));
    renderHistory({ queryHistory });
    await waitFor(() => expect(queryHistory).toHaveBeenCalledOnce());

    const search = screen.getByRole("textbox", { name: "搜索翻译记录" });
    await fireEvent.input(search, { target: { value: "中".repeat(MAX_INPUT_BYTES) } });
    await waitFor(() => expect(queryHistory.mock.calls.length).toBeGreaterThan(1), { timeout: 1000 });

    const submitted = queryHistory.mock.calls.at(-1)?.[0]?.query ?? "";
    expect(new TextEncoder().encode(submitted)).toHaveLength(MAX_INPUT_BYTES);
    expect(search).toHaveValue(submitted);
  });

  it("selects entries and keeps destructive clearing behind confirmation", async () => {
    const props = renderHistory();
    const entryButton = (await screen.findByText("hello world")).closest("button");
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

  it("keeps whole-history actions available when a search has no matches", async () => {
    const props = renderHistory();
    expect(await screen.findByText("2 条记录")).toBeInTheDocument();

    await fireEvent.input(screen.getByRole("textbox", { name: "搜索翻译记录" }), { target: { value: "missing" } });
    expect(await screen.findByText("0 条匹配记录", {}, { timeout: 1000 })).toBeInTheDocument();

    const exportButton = screen.getByRole("button", { name: "导出历史记录" });
    const clearButton = screen.getByRole("button", { name: "清空历史记录" });
    expect(exportButton).toBeEnabled();
    expect(clearButton).toBeEnabled();
    await fireEvent.click(exportButton);
    expect(props.onExport).toHaveBeenCalledOnce();
  });

  it("allows clearing a history file that failed to load", async () => {
    const onError = vi.fn();
    const onClear = vi.fn(async () => undefined);
    renderHistory({
      queryHistory: vi.fn(async () => { throw new Error("corrupt history"); }),
      onError,
      onClear,
    });

    await waitFor(() => expect(onError).toHaveBeenCalledWith("历史记录加载失败"));
    expect(screen.getByRole("button", { name: "导出历史记录" })).toBeDisabled();
    const clearButton = screen.getByRole("button", { name: "清空历史记录" });
    expect(clearButton).toBeEnabled();

    await fireEvent.click(clearButton);
    await fireEvent.click(screen.getByRole("button", { name: "确认清空" }));
    expect(onClear).toHaveBeenCalledOnce();
  });

  it("renders legacy entries that have duplicate ids", async () => {
    renderHistory({
      queryHistory: async () => ({ entries: [
        { ...history[0], id: 1 },
        { ...history[1], id: 1 },
      ], total: 2, allTotal: 2, hasMore: false }),
    });

    expect(await screen.findByText("hello world")).toBeInTheDocument();
    expect(await screen.findByText("再见")).toBeInTheDocument();
  });

  it("loads ten entries at a time and releases them when closed", async () => {
    const many = Array.from({ length: 12 }, (_, id) => ({ ...history[0], id, input: `input-${id}` }));
    const queryHistory = vi.fn(async ({ offset, limit }) => ({
      entries: many.slice(offset, offset + limit), total: many.length, allTotal: many.length, hasMore: offset + limit < many.length,
    }));
    const { rerender } = render(History, {
      open: true, queryHistory, onClose: vi.fn(), onClear: vi.fn(async () => undefined),
      onExport: vi.fn(async () => true), onError: vi.fn(), onSelect: vi.fn(),
    });

    expect(await screen.findByText("input-0")).toBeInTheDocument();
    expect(screen.queryByText("input-10")).not.toBeInTheDocument();
    expect(queryHistory).toHaveBeenCalledWith({ query: "", offset: 0, limit: 10 });
    await fireEvent.click(screen.getByRole("button", { name: "加载更多" }));
    expect(await screen.findByText("input-10")).toBeInTheDocument();

    await rerender({ open: false, queryHistory, onClose: vi.fn(), onClear: vi.fn(async () => undefined), onExport: vi.fn(async () => true), onError: vi.fn(), onSelect: vi.fn() });
    expect(screen.queryByText("input-0")).not.toBeInTheDocument();
  });

  it("ignores an in-flight page after history is cleared", async () => {
    let finishLoadMore: ((page: { entries: HistoryEntry[]; total: number; allTotal: number; hasMore: boolean }) => void) | undefined;
    const initial = { ...history[0], input: "initial-entry" };
    const late = { ...history[1], input: "late-old-entry" };
    const queryHistory = vi.fn()
      .mockResolvedValueOnce({ entries: [initial], total: 2, allTotal: 2, hasMore: true })
      .mockImplementationOnce(() => new Promise((resolve) => { finishLoadMore = resolve; }));
    const onClear = vi.fn(async () => undefined);
    renderHistory({ queryHistory, onClear });

    expect(await screen.findByText("initial-entry")).toBeInTheDocument();
    await fireEvent.click(screen.getByRole("button", { name: "加载更多" }));
    await fireEvent.click(screen.getByRole("button", { name: "清空历史记录" }));
    await fireEvent.click(screen.getByRole("button", { name: "确认清空" }));
    expect(onClear).toHaveBeenCalledOnce();

    finishLoadMore?.({ entries: [late], total: 2, allTotal: 2, hasMore: false });
    await waitFor(() => expect(screen.getByText("0 条记录")).toBeInTheDocument());
    expect(screen.queryByText("late-old-entry")).not.toBeInTheDocument();
  });
});
