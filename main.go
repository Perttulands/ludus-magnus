package main

import (
	"os"

	"github.com/Perttulands/ludus-magnus/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
