import { MAX_INPUT_BYTES, utf8ByteLength } from "./textLimits";

const DEFAULT_MAX_LEN = MAX_INPUT_BYTES;

export interface ClipboardReaderOptions {
  getText: () => Promise<string>;
  onText: (text: string) => void;
  isBusy?: () => boolean;
  maxTextLength?: number;
}

export interface ClipboardReader {
  read: () => Promise<void>;
  cancel: () => void;
  setBaseline: (text: string) => void;
}

export function createClipboardReader(opts: ClipboardReaderOptions): ClipboardReader {
  const {
    getText,
    onText,
    isBusy = () => false,
    maxTextLength = DEFAULT_MAX_LEN,
  } = opts;

  let activeRead: Promise<void> | null = null;
  let generation = 0;
  let baselineRevision = 0;
  let lastText = "";

  function setBaseline(text: string = ""): void {
    baselineRevision += 1;
    lastText = text || "";
  }

  function read(): Promise<void> {
    if (activeRead) return activeRead;

    const readGeneration = generation;
    const readBaselineRevision = baselineRevision;
    const task = (async () => {
      try {
        const text = await getText();
        if (readGeneration !== generation || readBaselineRevision !== baselineRevision) return;
        if (
          text &&
          text !== lastText &&
          text.trim().length > 0 &&
          utf8ByteLength(text) <= maxTextLength &&
          !isBusy()
        ) {
          lastText = text;
          onText(text);
        }
      } catch {
        // Clipboard access can fail when the platform denies it; retry on the next activation.
      } finally {
        if (readGeneration === generation) activeRead = null;
      }
    })();

    activeRead = task;
    return task;
  }

  function cancel(): void {
    generation += 1;
    activeRead = null;
  }

  return { read, cancel, setBaseline };
}
