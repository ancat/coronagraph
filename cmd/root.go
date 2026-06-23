package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cg",
	Short: "Coronagraph proxy and key management",
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}