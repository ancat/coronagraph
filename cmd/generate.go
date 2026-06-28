package cmd

import (
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate supporting files",
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.PersistentFlags().StringVar(&generateConfigPath, "config", "config.yml", "path to coronagraph config file")
	generateCmd.AddCommand(generateBundlesCmd)
	generateCmd.AddCommand(generateShimsCmd)
}
