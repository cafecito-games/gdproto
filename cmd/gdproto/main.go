// Package main is the entry point for the gdproto command-line tool.
package main

import (
	"os"

	"github.com/cafecito-games/gdproto/internal/cli"
)

func main() {
	os.Exit(cli.Execute(os.Args[1:], os.Stdout, os.Stderr))
}
