package main

import (
	"github.com/hrk091/nwctl/pkg/common"
	"os"
)

func main() {
	cmd := NewRootCmd()
	if err := cmd.Execute(); err != nil {
		common.ShowStackTrace(os.Stderr, err)
		os.Exit(1)
	}
}
