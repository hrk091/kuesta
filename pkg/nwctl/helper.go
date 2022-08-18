package nwctl

import (
	"github.com/pkg/errors"
	"os"
	"path/filepath"
)

func WriteFileWithMkdir(path string, buf []byte) error {
	dir, _ := filepath.Split(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err = os.MkdirAll(dir, 0750); err != nil {
			return errors.WithStack(err)
		}
	}
	if err := os.WriteFile(path, buf, 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
