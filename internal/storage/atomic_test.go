package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFileAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "data.json")
	for _, content := range []string{"first", "second"} {
		if err := WriteFileAtomic(path, []byte(content), 0600); err != nil {
			t.Fatalf("WriteFileAtomic 失败: %v", err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取写入结果失败: %v", err)
		}
		if string(got) != content {
			t.Fatalf("content = %q, want %q", got, content)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat 失败: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
}
