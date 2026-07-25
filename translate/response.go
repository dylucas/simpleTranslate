package translate

import (
	"fmt"
	"io"
)

const maxAPIResponseBytes = 2 << 20

func readAPIResponse(body io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, maxAPIResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxAPIResponseBytes {
		return nil, fmt.Errorf("API 响应超过 %d 字节限制", maxAPIResponseBytes)
	}
	return data, nil
}
