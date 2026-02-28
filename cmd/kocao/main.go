package main

import (
	"os"

	"github.com/withakay/kocao/internal/controlplanecli"
)

func main() {
	os.Exit(controlplanecli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
