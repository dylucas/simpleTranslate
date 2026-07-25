import { describe, expect, it } from "vitest";
import { MAX_INPUT_BYTES, truncateUtf8, utf8ByteLength } from "./textLimits";

describe("UTF-8 text limits", () => {
  it("counts ASCII and multibyte characters by encoded bytes", () => {
    expect(utf8ByteLength("abc")).toBe(3);
    expect(utf8ByteLength("中文")).toBe(6);
  });

  it("truncates without splitting a multibyte character", () => {
    const value = "a".repeat(MAX_INPUT_BYTES - 2) + "中";
    const truncated = truncateUtf8(value);
    expect(utf8ByteLength(truncated)).toBe(MAX_INPUT_BYTES - 2);
    expect(truncated.endsWith("中")).toBe(false);
  });
});
