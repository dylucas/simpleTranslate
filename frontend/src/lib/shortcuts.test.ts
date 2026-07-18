import { describe, expect, it, vi } from "vitest";
import { ARIA_SHORTCUTS, createShortcutHandler, type ShortcutHandlers } from "./shortcuts";

function press(handler: (event: KeyboardEvent) => void, key: string, init: KeyboardEventInit = {}) {
  const event = new KeyboardEvent("keydown", { key, cancelable: true, ...init });
  handler(event);
  return event;
}

function createHandlers(): Required<ShortcutHandlers> {
  return {
    onTranslate: vi.fn(),
    onFocusInput: vi.fn(),
    onClearInput: vi.fn(),
    onSwapLangs: vi.fn(),
    onToggleHistory: vi.fn(),
    onToggleSettings: vi.fn(),
    onToggleTheme: vi.fn(),
    onClosePanel: vi.fn(),
  };
}

describe("keyboard shortcuts", () => {
  it("exposes platform-specific ARIA shortcut values", () => {
    expect(ARIA_SHORTCUTS.translate).toBe("Control+Enter Meta+Enter");
    expect(ARIA_SHORTCUTS.toggleHistory).toBe("Control+Shift+H Meta+Shift+H");
    expect(ARIA_SHORTCUTS.toggleSettings).toBe("Control+, Meta+,");
    expect(ARIA_SHORTCUTS.closePanel).toBe("Escape");
  });

  it("dispatches the supported Ctrl/Cmd shortcuts", () => {
    const handlers = createHandlers();
    const shortcut = createShortcutHandler(handlers);

    expect(press(shortcut, "Enter", { ctrlKey: true }).defaultPrevented).toBe(true);
    press(shortcut, "L", { metaKey: true });
    press(shortcut, "k", { ctrlKey: true });
    press(shortcut, "j", { metaKey: true });
    press(shortcut, "H", { ctrlKey: true, shiftKey: true });
    press(shortcut, ",", { metaKey: true });
    press(shortcut, "m", { ctrlKey: true });
    press(shortcut, "Escape");

    expect(handlers.onTranslate).toHaveBeenCalledOnce();
    expect(handlers.onFocusInput).toHaveBeenCalledOnce();
    expect(handlers.onClearInput).toHaveBeenCalledOnce();
    expect(handlers.onSwapLangs).toHaveBeenCalledOnce();
    expect(handlers.onToggleHistory).toHaveBeenCalledOnce();
    expect(handlers.onToggleSettings).toHaveBeenCalledOnce();
    expect(handlers.onToggleTheme).toHaveBeenCalledOnce();
    expect(handlers.onClosePanel).not.toHaveBeenCalled();
  });

  it("ignores repeats, IME composition, handled events, and extra modifiers", () => {
    const handlers = createHandlers();
    const shortcut = createShortcutHandler(handlers);
    const handled = new KeyboardEvent("keydown", { key: "k", ctrlKey: true, cancelable: true });
    handled.preventDefault();

    shortcut(handled);
    press(shortcut, "Enter", { ctrlKey: true, repeat: true });
    press(shortcut, "Enter", { ctrlKey: true, isComposing: true });
    press(shortcut, "k", { ctrlKey: true, shiftKey: true });
    press(shortcut, "m", { ctrlKey: true, altKey: true });

    for (const handler of Object.values(handlers)) expect(handler).not.toHaveBeenCalled();
  });

  it("allows navigation shortcuts while a panel is open and blocks destructive actions", () => {
    const handlers = createHandlers();
    const shortcut = createShortcutHandler(handlers, { isPanelOpen: () => true });

    press(shortcut, "Enter", { ctrlKey: true });
    press(shortcut, "k", { ctrlKey: true });
    press(shortcut, "j", { ctrlKey: true });
    press(shortcut, "l", { ctrlKey: true });
    press(shortcut, "h", { ctrlKey: true, shiftKey: true });
    press(shortcut, ",", { ctrlKey: true });
    press(shortcut, "m", { ctrlKey: true });
    press(shortcut, "Escape");

    expect(handlers.onTranslate).not.toHaveBeenCalled();
    expect(handlers.onClearInput).not.toHaveBeenCalled();
    expect(handlers.onSwapLangs).not.toHaveBeenCalled();
    expect(handlers.onFocusInput).toHaveBeenCalledOnce();
    expect(handlers.onToggleHistory).toHaveBeenCalledOnce();
    expect(handlers.onToggleSettings).toHaveBeenCalledOnce();
    expect(handlers.onToggleTheme).toHaveBeenCalledOnce();
    expect(handlers.onClosePanel).toHaveBeenCalledOnce();
  });
});
