package translate

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

const maxAPIResponseBytes = 2 << 20

// bufPool 复用 bytes.Buffer，避免每次 HTTP 响应读取与阿里云响应解码都重新分配。
// 桌面翻译应用请求频繁，缓冲池可显著降低 GC 压力。
var bufPool = sync.Pool{
	New: func() interface{} { return new(bytes.Buffer) },
}

func readAPIResponse(body io.Reader) ([]byte, error) {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if _, err := io.Copy(buf, io.LimitReader(body, maxAPIResponseBytes+1)); err != nil {
		return nil, err
	}
	data := buf.Bytes()
	if len(data) > maxAPIResponseBytes {
		return nil, fmt.Errorf("API 响应超过 %d 字节限制", maxAPIResponseBytes)
	}
	// 必须拷贝：buf 归还后会复用，调用方需要独立字节数组
	out := make([]byte, len(data))
	copy(out, data)
	return out, nil
}

// decodeThroughBuffer 将 src 序列化后再反序列化到 dst，使用复用缓冲区避免中间 []byte 分配。
// 用于阿里云 SDK 返回的 TeaResponse → 业务结构体转换，替代 json.Marshal + json.Unmarshal。
func decodeThroughBuffer(src interface{}, dst interface{}) error {
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufPool.Put(buf)
	if err := json.NewEncoder(buf).Encode(src); err != nil {
		return err
	}
	return json.NewDecoder(buf).Decode(dst)
}
