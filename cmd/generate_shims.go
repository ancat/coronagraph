package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"coronagraph/bundle"
	"coronagraph/config"
	"coronagraph/shim"
)

var generateShimsCmd = &cobra.Command{
	Use:   "shims",
	Short: "Write command shims to ~/.cg/bin",
	Long: `Write wrapper scripts for gh, gem, and bundle under ~/.cg/bin.

Each shim points HTTP clients at the coronagraph proxy (port from config),
sets placeholder credentials (coronagraph injects real ones), and uses
~/.cg/bundle.pem for TLS verification.

Run "cg generate bundles" first. Then prepend ~/.cg/bin to PATH:

  export PATH="$HOME/.cg/bin:$PATH"`,
	RunE: runGenerateShims,
}

func runGenerateShims(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(generateConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	bundlePath, err := bundle.CombinedBundlePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(bundlePath); err != nil {
		return fmt.Errorf("%s not found; run \"cg generate bundles\" first", bundlePath)
	}

	binDir, err := bundle.BinDir()
	if err != nil {
		return err
	}

	proxyURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Port())
	written, err := shim.Generate(shim.Options{
		ProxyURL:   proxyURL,
		BundlePath: bundlePath,
		BinDir:     binDir,
	})
	if err != nil {
		return err
	}

	for _, path := range written {
		fmt.Printf("wrote %s\n", path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	fmt.Printf("\nAdd to your shell profile:\n  export PATH=\"%s/.cg/bin:$PATH\"\n", home)
	return nil
}
