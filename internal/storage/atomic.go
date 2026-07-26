package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// WriteFileAtomic writes data to a temporary file and then replaces path.
// Keeping the temporary file in the destination directory makes the rename
// atomic on the filesystems supported by the desktop application.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return fmt.Errorf("set file permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("flush temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace destination file: %w", err)
	}
	if err := syncDirectory(dir); err != nil {
		return fmt.Errorf("flush destination directory: %w", err)
	}
	return nil
}

func syncDirectory(path string) error {
	// Windows does not support syncing an opened directory through os.File.
	// Rename durability is delegated to the platform implementation there.
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := dir.Sync(); err != nil {
		dir.Close()
		return err
	}
	return dir.Close()
}
