// Package main 是 Forge 的入口。
package main

import (
	"os"

	"github.com/kkkldpz/forge/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
