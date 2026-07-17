// 全局快捷键分发器：将按键事件按 modifier+key 映射到回调。
// 与组件状态解耦，App.svelte 只需注入 handlers 对象即可。
//
// 使用方式：
//   const handler = createShortcutHandler({
//     onTranslate: () => translate(),
//     onFocusInput: () => inputEl?.focus(),
//     onClearInput: () => (input = ""),
//     onSwapLangs: () => ([source, target] = [target, source]),
//     onToggleHistory: () => (showHistory = !showHistory),
//     onToggleTheme: () => updateAndSaveConfig("isDark", !$configStore.isDark),
//     onClosePanel: () => { if (showHistory) showHistory = false; },
//   });
//   <svelte:window on:keydown={handler} />

export interface ShortcutHandlers {
  onTranslate?: () => void;
  onFocusInput?: () => void;
  onClearInput?: () => void;
  onSwapLangs?: () => void;
  onToggleHistory?: () => void;
  onToggleTheme?: () => void;
  onClosePanel?: () => void;
}

// isMod 检测 Ctrl/Cmd 修饰键（macOS 用 Cmd，其他平台用 Ctrl）
function isMod(e: KeyboardEvent): boolean {
  return e.ctrlKey || e.metaKey;
}

// keyMatches 同时匹配大小写形式（避免 caps lock 影响）
function keyMatches(e: KeyboardEvent, key: string): boolean {
  const lower = e.key.toLowerCase();
  return lower === key.toLowerCase();
}

export function createShortcutHandler(handlers: ShortcutHandlers) {
  return function handleKeydown(e: KeyboardEvent): void {
    // Ctrl/Cmd + Enter：发送翻译
    if (isMod(e) && e.key === "Enter") {
      e.preventDefault();
      handlers.onTranslate?.();
      return;
    }
    // Ctrl/Cmd + L：聚焦输入
    if (isMod(e) && keyMatches(e, "l")) {
      e.preventDefault();
      handlers.onFocusInput?.();
      return;
    }
    // Ctrl/Cmd + K：清空输入
    if (isMod(e) && keyMatches(e, "k")) {
      e.preventDefault();
      handlers.onClearInput?.();
      return;
    }
    // Ctrl/Cmd + J：交换源/目标
    if (isMod(e) && keyMatches(e, "j")) {
      e.preventDefault();
      handlers.onSwapLangs?.();
      return;
    }
    // Ctrl/Cmd + Shift + H：切换历史面板
    if (isMod(e) && e.shiftKey && keyMatches(e, "h")) {
      e.preventDefault();
      handlers.onToggleHistory?.();
      return;
    }
    // Ctrl/Cmd + M：切换主题
    if (isMod(e) && keyMatches(e, "m")) {
      e.preventDefault();
      handlers.onToggleTheme?.();
      return;
    }
    // Esc：关闭面板
    if (e.key === "Escape") {
      handlers.onClosePanel?.();
    }
  };
}
