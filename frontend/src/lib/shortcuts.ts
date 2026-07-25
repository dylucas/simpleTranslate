export const ARIA_SHORTCUTS = {
  translate: "Control+Enter Meta+Enter",
  focusInput: "Control+L Meta+L",
  clearInput: "Control+K Meta+K",
  swapLanguages: "Control+J Meta+J",
  toggleHistory: "Control+Shift+H Meta+Shift+H",
  toggleSettings: "Control+, Meta+,",
  toggleTheme: "Control+M Meta+M",
  closePanel: "Escape",
  cancelTranslation: "Escape",
} as const;

export interface ShortcutHandlers {
  onTranslate?: () => void;
  onCancel?: () => void;
  onFocusInput?: () => void;
  onClearInput?: () => void;
  onSwapLangs?: () => void;
  onToggleHistory?: () => void;
  onToggleSettings?: () => void;
  onToggleTheme?: () => void;
  onClosePanel?: () => void;
}

export interface ShortcutOptions {
  isPanelOpen?: () => boolean;
}

function isMod(event: KeyboardEvent): boolean {
  return event.ctrlKey || event.metaKey;
}

function keyMatches(event: KeyboardEvent, key: string): boolean {
  return event.key.toLowerCase() === key.toLowerCase();
}

function hasModifiers(event: KeyboardEvent, shift = false): boolean {
  return isMod(event) && event.shiftKey === shift && !event.altKey;
}

function shouldIgnore(event: KeyboardEvent): boolean {
  return event.defaultPrevented || event.repeat || event.isComposing || event.keyCode === 229;
}

export function createShortcutHandler(handlers: ShortcutHandlers, options: ShortcutOptions = {}) {
  return function handleKeydown(event: KeyboardEvent): void {
    if (shouldIgnore(event)) return;

    const panelOpen = options.isPanelOpen?.() ?? false;
    const invoke = (handler: (() => void) | undefined, allowWhenPanelOpen = false): void => {
      if (!handler || (panelOpen && !allowWhenPanelOpen)) return;
      event.preventDefault();
      handler();
    };

    if (hasModifiers(event) && event.key === "Enter") {
      invoke(handlers.onTranslate);
      return;
    }
    if (hasModifiers(event) && keyMatches(event, "l")) {
      invoke(handlers.onFocusInput, true);
      return;
    }
    if (hasModifiers(event) && keyMatches(event, "k")) {
      invoke(handlers.onClearInput);
      return;
    }
    if (hasModifiers(event) && keyMatches(event, "j")) {
      invoke(handlers.onSwapLangs);
      return;
    }
    if (hasModifiers(event, true) && keyMatches(event, "h")) {
      invoke(handlers.onToggleHistory, true);
      return;
    }
    if (hasModifiers(event) && keyMatches(event, ",")) {
      invoke(handlers.onToggleSettings, true);
      return;
    }
    if (hasModifiers(event) && keyMatches(event, "m")) {
      invoke(handlers.onToggleTheme, true);
      return;
    }
    if (event.key === "Escape") {
      if (panelOpen) invoke(handlers.onClosePanel, true);
      else invoke(handlers.onCancel);
    }
  };
}
