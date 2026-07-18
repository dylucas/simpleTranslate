// 剪贴板监听器：封装轮询、变化检测与回调触发，与 Svelte 响应式状态解耦。
//
// 使用方式：
//   const watcher = createClipboardWatcher({
//     getText: () => ClipboardGetText(),         // Wails 提供的读取剪贴板函数
//     onText: (text) => { input = text; ... },   // 命中新文本时的回调
//     isBusy: () => isProcessing,                 // 当前是否处理中（true 时跳过触发）
//     intervalMs: 1500,                          // 轮询间隔
//     maxTextLength: 5000,                       // 超长文本过滤阈值
//   });
//   watcher.start();  // 启动并记录基线
//   watcher.stop();   // 停止轮询
//   watcher.setBaseline(text); // 显式更新基线

const DEFAULT_INTERVAL = 1500;
const DEFAULT_MAX_LEN = 5000;

export interface ClipboardWatcherOptions {
  getText: () => Promise<string>;
  onText: (text: string) => void;
  isBusy?: () => boolean;
  intervalMs?: number;
  maxTextLength?: number;
}

export interface ClipboardWatcher {
  start: () => void;
  stop: () => void;
  setBaseline: (text: string) => void;
  isRunning: () => boolean;
}

export function createClipboardWatcher(opts: ClipboardWatcherOptions): ClipboardWatcher {
  const {
    getText,
    onText,
    isBusy = () => false,
    intervalMs = DEFAULT_INTERVAL,
    maxTextLength = DEFAULT_MAX_LEN,
  } = opts;

  let timer: ReturnType<typeof setInterval> | null = null;
  let lastText = "";

  function setBaseline(text: string = ""): void {
    lastText = text || "";
  }

  async function poll(): Promise<void> {
    try {
      const text = await getText();
      if (
        text &&
        text !== lastText &&
        text.trim().length > 0 &&
        text.length < maxTextLength &&
        !isBusy()
      ) {
        lastText = text;
        onText(text);
      }
    } catch {
      // 读取失败静默忽略，下次重试
    }
  }

  function start(): void {
    if (timer) return;
    // 先记录当前剪贴板内容作为基线，避免开启即触发
    getText()
      .then((text) => setBaseline(text))
      .catch(() => {});
    timer = setInterval(poll, intervalMs);
  }

  function stop(): void {
    if (timer) {
      clearInterval(timer);
      timer = null;
    }
  }

  return { start, stop, setBaseline, isRunning: () => timer !== null };
}
