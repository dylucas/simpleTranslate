import { afterEach, describe, expect, it, vi } from "vitest";
import { createSpeaker } from "./speech";

class MockUtterance {
  lang = "";
  onend: (() => void) | null = null;
  onerror: (() => void) | null = null;

  constructor(readonly text: string) {}
}

afterEach(() => vi.unstubAllGlobals());

describe("speech controller", () => {
  it("ignores completion events from a cancelled utterance", () => {
    const spoken: MockUtterance[] = [];
    vi.stubGlobal("SpeechSynthesisUtterance", MockUtterance);
    vi.stubGlobal("speechSynthesis", {
      cancel: vi.fn(),
      speak: vi.fn((utterance: MockUtterance) => spoken.push(utterance)),
    });
    const changes: Array<string | null> = [];
    const speaker = createSpeaker((code) => code);

    speaker.speak("first", "en", { onChange: (value) => changes.push(value) });
    const firstFinish = spoken[0].onend;
    speaker.speak("second", "zh", { onChange: (value) => changes.push(value) });
    firstFinish?.();

    expect(speaker.current()).toBe("second");
    expect(changes).toEqual(["first", "second"]);

    spoken[1].onend?.();
    expect(speaker.current()).toBeNull();
    expect(changes).toEqual(["first", "second", null]);
  });
});
