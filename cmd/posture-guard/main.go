package main

import (
	"fmt"
	"os"

	"github.com/MatheusDSantossi/posture-guard/internal/cli"
)

var version = "0.3.3"

func main() {
	cli.Version = version
	rootCmd, err := cli.BuildRootCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
