package nwctl

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func CollectPartialDeviceConfig(dir, device string) ([]string, error) {
	var files []string
	walkDirFunc := func(path string, info fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir: %w", err)
		}
		if !info.IsDir() {
			return nil
		}
		if info.Name() != DirComputed {
			return nil
		}

		p := filepath.Join(path, fmt.Sprintf("%s.cue", device))
		if _, err := os.Stat(p); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return fmt.Errorf("check if file exists: %w", err)
		}
		files = append(files, p)
		return filepath.SkipDir
	}

	if err := filepath.WalkDir(dir, walkDirFunc); err != nil {
		return nil, err
	}
	return files, nil
}
