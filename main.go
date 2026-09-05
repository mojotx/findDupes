package main

import (
	"os"

	"github.com/mojotx/findDupes/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
