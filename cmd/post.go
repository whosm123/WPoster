package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var postCmd = &cobra.Command{
	Use:   "post [content]",
	Short: "Post content",
	Long: `Post content to the configured platform.
You can provide content directly as an argument or use flags for more control.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		content := args[0]
		fmt.Printf("Posting: %s\n", content)
		fmt.Println("Content posted successfully!")
	},
}

func init() {
	rootCmd.AddCommand(postCmd)
}
