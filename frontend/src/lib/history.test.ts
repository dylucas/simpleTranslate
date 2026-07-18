import { describe, expect, it } from "vitest";
import { createHistoryEntry, isDuplicateRecent, prependHistory } from "./history";

describe("history helpers", () => {
  it("deduplicates the latest language pair and enforces the limit", () => {
    const first = createHistoryEntry({ input: "hello", output: "你好", source: "en", target: "zh", id: 1, time: "now" });
    expect(isDuplicateRecent([first], "hello", "en", "zh")).toBe(true);
    const next = createHistoryEntry({ input: "world", output: "世界", source: "en", target: "zh", id: 2, time: "now" });
    expect(prependHistory([first], next, 1)).toEqual([next]);
  });
});
