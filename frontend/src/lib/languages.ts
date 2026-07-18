// 语种相关常量与映射，统一供前端组件复用，避免散落定义造成不一致。

// 应用内部语种代码 → 展示名（用于下拉框与历史记录展示）
export const langs: Record<string, string> = {
  zh: "中文",
  en: "英文",
  jp: "日语",
  kr: "韩语",
  fr: "法语",
  de: "德语",
  ru: "俄语",
  es: "西语",
};

// 语种代码 → Web Speech API 的 BCP-47 语言标签
export const langMap: Record<string, string> = {
  zh: "zh-CN",
  en: "en-US",
  jp: "ja-JP",
  kr: "ko-KR",
  fr: "fr-FR",
  de: "de-DE",
  ru: "ru-RU",
  es: "es-ES",
};

// getSpeechLang 返回 Web Speech API 期望的语言标签，未知代码兜底为 en-US
export function getSpeechLang(code: string): string {
  return langMap[code] || "en-US";
}

// formatAutoDetect 拼接自动识别下拉框显示文本
export function formatAutoDetect(detectedLang: string): string {
  const name = langs[detectedLang] || detectedLang;
  return `自动 (${name})`;
}
