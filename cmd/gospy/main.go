package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gospy",
	Short: "A reconnaissance platform for web targets",
	Long:  `gospy is a reconnaissance platform whose first capability is screenshot generation.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
