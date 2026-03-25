package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wposter",
	Short: "WPoster - A CLI tool for posting content",
	Long: `WPoster is a command-line tool for posting content to various platforms.
It provides a simple interface to manage and publish content from the terminal.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Welcome to WPoster! Use --help to see available commands.")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
