import type { BaiduDomain } from "./types";

export const BAIDU_DOMAIN_OPTIONS = [
  { value: "general", label: "通用" },
  { value: "it", label: "信息技术" },
  { value: "finance", label: "金融财经" },
  { value: "machinery", label: "机械制造" },
  { value: "senimed", label: "生物医药" },
  { value: "novel", label: "网络文学" },
  { value: "academic", label: "学术论文" },
  { value: "aerospace", label: "航空航天" },
  { value: "wiki", label: "人文社科" },
  { value: "news", label: "新闻资讯" },
  { value: "law", label: "法律法规" },
  { value: "contract", label: "合同" },
] as const satisfies ReadonlyArray<{ value: BaiduDomain; label: string }>;

export const BAIDU_DOMAINS = new Set<BaiduDomain>(
  BAIDU_DOMAIN_OPTIONS.map(({ value }) => value),
);
