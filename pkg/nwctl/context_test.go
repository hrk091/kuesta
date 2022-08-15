package nwctl_test

import (
	"bytes"
	"context"
	"github.com/hrk091/nwctl/pkg/nwctl"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestWriterFromContext(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, os.Stdout, nwctl.WriterFromContext(ctx))
}

func TestWriterFromContext_WithWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	ctx := context.Background()
	ctx = nwctl.WithWriter(ctx, buf)
	assert.Equal(t, buf, nwctl.WriterFromContext(ctx))
}
