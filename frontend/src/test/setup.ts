import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/svelte";
import { afterEach } from "vitest";

afterEach(() => cleanup());

Object.defineProperty(window.navigator, "clipboard", {
  configurable: true,
  value: {
    writeText: async () => undefined,
    readText: async () => "",
  },
});
