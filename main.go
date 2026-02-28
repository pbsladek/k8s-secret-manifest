package main

import (
	"fmt"
	"os"

	"github.com/pbsladek/k8s-secret-manifest/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
