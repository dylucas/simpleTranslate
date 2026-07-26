package translate

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestReadAPIResponseLimit(t *testing.T) {
	exact := bytes.Repeat([]byte("x"), maxAPIResponseBytes)
	data, err := readAPIResponse(bytes.NewReader(exact))
	if err != nil || len(data) != maxAPIResponseBytes {
		t.Fatalf("exact limit: len=%d err=%v", len(data), err)
	}
	if _, err := readAPIResponse(bytes.NewReader(append(exact, 'x'))); err == nil {
		t.Fatal("response above limit should fail")
	}
}

func TestReleaseAPIResponseBufferClearsContents(t *testing.T) {
	buf := new(bytes.Buffer)
	buf.Write(bytes.Repeat([]byte{0x7f}, maxPooledAPIResponseBufferBytes+1))
	contents := buf.Bytes()

	releaseAPIResponseBuffer(buf)

	if buf.Len() != 0 {
		t.Fatalf("released buffer length = %d, want 0", buf.Len())
	}
	for i, value := range contents {
		if value != 0 {
			t.Fatalf("released buffer byte %d = %d, want 0", i, value)
		}
	}
}

func TestDecodeThroughBufferLimit(t *testing.T) {
	var decoded map[string]string
	if err := decodeThroughBuffer(map[string]string{"value": "ok"}, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["value"] != "ok" {
		t.Fatalf("decoded value = %q", decoded["value"])
	}

	err := decodeThroughBuffer(
		map[string]string{"value": strings.Repeat("x", maxAPIResponseBytes)},
		&decoded,
	)
	if !errors.Is(err, errAPIResponseTooLarge) {
		t.Fatalf("oversized SDK response error = %v", err)
	}
}

func TestAPIErrorExcerptIsBoundedAndValidUTF8(t *testing.T) {
	excerpt := apiErrorExcerpt(strings.Repeat("中", maxAPIErrorExcerptBytes))
	if !utf8.ValidString(excerpt) {
		t.Fatalf("excerpt is not valid UTF-8: %q", excerpt)
	}
	if len(excerpt) > maxAPIErrorExcerptBytes+len("...") || !strings.HasSuffix(excerpt, "...") {
		t.Fatalf("unexpected excerpt length or suffix: len=%d suffix=%q", len(excerpt), excerpt[len(excerpt)-3:])
	}

	invalid := apiErrorExcerpt(string([]byte{'a', 0xff, 'b'}))
	if !utf8.ValidString(invalid) {
		t.Fatalf("invalid input was not normalized: %q", invalid)
	}
}

// BenchmarkReadAPIResponse 测量 HTTP 响应读取的分配开销（优化后使用 sync.Pool 复用 Buffer）。
func BenchmarkReadAPIResponse(b *testing.B) {
	payload := bytes.Repeat([]byte(`{"text":"result"}`), 200) // ~3KB
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = readAPIResponse(bytes.NewReader(payload))
	}
}

// BenchmarkReadAPIResponse_Large 测量大响应（~100KB）下的分配开销。
func BenchmarkReadAPIResponse_Large(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 100<<10)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = readAPIResponse(bytes.NewReader(payload))
	}
}

// BenchmarkDecodeThroughBuffer 测量阿里云响应解码的分配开销（优化后复用缓冲区）。
func BenchmarkDecodeThroughBuffer(b *testing.B) {
	src := map[string]interface{}{
		"body": map[string]interface{}{
			"Code":      "200",
			"Message":   "ok",
			"RequestId": "abc-123",
			"Data": map[string]interface{}{
				"Translated": "hello world",
				"WordCount":  "2",
			},
		},
		"headers":    map[string]string{"Content-Type": "application/json"},
		"statusCode": 200,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var dst struct {
			Body       interface{}       `json:"body"`
			Headers    map[string]string `json:"headers"`
			StatusCode int               `json:"statusCode"`
		}
		_ = decodeThroughBuffer(src, &dst)
	}
}
