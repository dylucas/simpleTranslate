package translate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	maxAPIResponseBytes             = 2 << 20
	maxPooledAPIResponseBufferBytes = 256 << 10
	maxAPIErrorExcerptBytes         = 1 << 10
)

var errAPIResponseTooLarge = fmt.Errorf("API 响应超过 %d 字节限制", maxAPIResponseBytes)

// bufPool 复用 bytes.Buffer，避免每次 HTTP 响应读取与阿里云响应解码都重新分配。
// 桌面翻译应用请求频繁，缓冲池可显著降低 GC 压力。
var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func acquireAPIResponseBuffer() *bytes.Buffer {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func releaseAPIResponseBuffer(buf *bytes.Buffer) {
	clear(buf.Bytes())
	buf.Reset()
	if buf.Cap() <= maxPooledAPIResponseBufferBytes {
		bufPool.Put(buf)
	}
}

func readAPIResponse(body io.Reader) ([]byte, error) {
	buf := acquireAPIResponseBuffer()
	defer releaseAPIResponseBuffer(buf)
	if _, err := io.Copy(buf, io.LimitReader(body, maxAPIResponseBytes+1)); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	if len(data) > maxAPIResponseBytes {
		return nil, errAPIResponseTooLarge
	}
	// 必须拷贝：buf 归还后会复用，调用方需要独立字节数组
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

func utf8Prefix(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end]
}

func apiErrorExcerpt(value string) string {
	value = strings.TrimSpace(strings.ToValidUTF8(value, "\uFFFD"))
	if len(value) <= maxAPIErrorExcerptBytes {
		return value
	}
	return utf8Prefix(value, maxAPIErrorExcerptBytes) + "..."
}

type limitedBufferWriter struct {
	buffer *bytes.Buffer
	limit  int
}

func (w *limitedBufferWriter) Write(data []byte) (int, error) {
	if len(data) > w.limit-w.buffer.Len() {
		return 0, errAPIResponseTooLarge
	}
	return w.buffer.Write(data)
}

// decodeThroughBuffer 将 src 序列化后再反序列化到 dst，使用复用缓冲区避免中间 []byte 分配。
// 用于阿里云 SDK 返回的 TeaResponse → 业务结构体转换，替代 json.Marshal + json.Unmarshal。
func decodeThroughBuffer(src interface{}, dst interface{}) error {
	buf := acquireAPIResponseBuffer()
	defer releaseAPIResponseBuffer(buf)
	writer := &limitedBufferWriter{buffer: buf, limit: maxAPIResponseBytes}
	if err := json.NewEncoder(writer).Encode(src); err != nil {
		return err
	}
	return json.NewDecoder(buf).Decode(dst)
}
