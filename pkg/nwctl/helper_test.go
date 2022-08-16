package nwctl_test

import (
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func ExitOnErr(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

func TestWriteFileWithMkdir(t *testing.T) {
	dir := t.TempDir()
	buf := []byte("foobar")

	t.Run("valid: new dir", func(t *testing.T) {
		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err := nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

	t.Run("valid: existing dir", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, "foo", "bar"), 750)
		ExitOnErr(t, err)

		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err = nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

	t.Run("valid: write multiple times", func(t *testing.T) {
		err := os.MkdirAll(filepath.Join(dir, "foo", "bar"), 750)
		ExitOnErr(t, err)

		path := filepath.Join(dir, "foo", "bar", "tmp.txt")
		err = nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)
		err = nwctl.WriteFileWithMkdir(path, buf)
		ExitOnErr(t, err)

		got, err := os.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, buf, got)
	})

}
