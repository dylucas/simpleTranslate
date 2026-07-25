package translate

import (
	"bytes"
	"testing"
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
