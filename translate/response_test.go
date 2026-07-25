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
