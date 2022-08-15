package nwctl

import (
	"context"
	"io"
	"os"
)

type _keyWriter struct{}

func WithWriter(parent context.Context, w io.Writer) context.Context {
	return context.WithValue(parent, _keyWriter{}, w)
}

func WriterFromContext(ctx context.Context) io.Writer {
	v, ok := ctx.Value(_keyWriter{}).(io.Writer)
	if !ok {
		return os.Stdout
	} else {
		return v
	}
}
