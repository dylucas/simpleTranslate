// 语音朗读工具：封装 Web Speech API，跟踪当前朗读的文本以便切换停止。
//
// 使用方式：
//   const speaker = createSpeaker(getSpeechLang);
//   speaker.speak(text, langCode, { onChange: (cur) => ... });
//   speaker.stop();
//   speaker.current(); // 当前正在朗读的文本（用于 UI 高亮）
//
// 与组件状态解耦：speaker 内部用回调通知 UI 更新高亮状态。

export interface SpeakOptions {
  onChange?: (text: string | null) => void;
}

export interface Speaker {
  speak: (text: string, langCode: string, opts?: SpeakOptions) => void;
  stop: () => void;
  current: () => string | null;
}

export function createSpeaker(getSpeechLang: (code: string) => string): Speaker {
  let speakingText: string | null = null;
  let activeUtterance: SpeechSynthesisUtterance | null = null;

  function stop(): void {
    activeUtterance = null;
    if (typeof window !== "undefined" && window.speechSynthesis) {
      window.speechSynthesis.cancel();
    }
    speakingText = null;
  }

  function speak(text: string, langCode: string, opts?: SpeakOptions): void {
    const onChange = opts?.onChange;
    if (!text) return;
    if (typeof window === "undefined" || !window.speechSynthesis) return;

    // 点击正在朗读的文本则停止
    if (speakingText === text) {
      stop();
      onChange?.(null);
      return;
    }

    stop();
    const u = new SpeechSynthesisUtterance(text);
    activeUtterance = u;
    u.lang = getSpeechLang(langCode);
    const finish = () => {
      if (activeUtterance !== u) return;
      activeUtterance = null;
      speakingText = null;
      onChange?.(null);
    };
    u.onend = finish;
    u.onerror = finish;
    speakingText = text;
    onChange?.(text);
    window.speechSynthesis.speak(u);
  }

  return {
    speak,
    stop,
    current: () => speakingText,
  };
}
