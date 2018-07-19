package main

import (
	"os"

	"github.com/zhengyuxin/perfbeat/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
