package translate

import "testing"

// TestNormalizeLangCode 验证模型返回的各种语种代码/名称归一化
func TestNormalizeLangCode(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"ja", "jp"},
		{"JA", "jp"},
		{"ko", "kr"},
		{"zh-cn", "zh"},
		{"en-us", "en"},
		{"english", "en"},
		{"中文", "zh"},
		{"汉语", "zh"},
		{"英语", "en"},
		{"`zh`", "zh"},
		{"\"en\"", "en"},
		{"fr", "fr"},
		// 未知代码原样返回（小写、去标点后）
		{"xx", "xx"},
	}
	for _, c := range cases {
		got := normalizeLangCode(c.in)
		if got != c.want {
			t.Errorf("normalizeLangCode(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestLangCodeToName 验证内部代码到中文名的映射覆盖
func TestLangCodeToName(t *testing.T) {
	wantKeys := []string{"zh", "en", "jp", "ja", "kr", "ko", "fr", "de", "ru", "es"}
	for _, k := range wantKeys {
		if _, ok := langCodeToName[k]; !ok {
			t.Errorf("langCodeToName 缺少键 %q", k)
		}
	}
	if langCodeToName["zh"] != "中文" {
		t.Errorf("langCodeToName[zh] 期望 中文，得到 %q", langCodeToName["zh"])
	}
}

// TestNameToLangCode_Reverse 验证中文名到代码映射与正向一致
func TestNameToLangCode_Reverse(t *testing.T) {
	cases := map[string]string{
		"中文":   "zh",
		"英语":   "en",
		"日语":   "jp",
		"韩语":   "kr",
		"法语":   "fr",
		"德语":   "de",
		"俄语":   "ru",
		"西班牙语": "es",
	}
	for name, code := range cases {
		got, ok := nameToLangCode[name]
		if !ok {
			t.Errorf("nameToLangCode 缺少 %q", name)
			continue
		}
		if got != code {
			t.Errorf("nameToLangCode[%q] = %q, want %q", name, got, code)
		}
	}
}

// TestDetectLanguage_EmptyText 空文本应返回错误
func TestDetectLanguage_EmptyText(t *testing.T) {
	_, err := DetectLanguage("")
	if err == nil {
		t.Error("空文本应返回错误，得到 nil")
	}
}

// TestDetectLanguage_Whitespace 仅空白应返回错误
func TestDetectLanguage_Whitespace(t *testing.T) {
	_, err := DetectLanguage("   \n\t  ")
	if err == nil {
		t.Error("纯空白文本应返回错误，得到 nil")
	}
}
