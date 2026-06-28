package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"coronagraph/bundle"
	"coronagraph/config"
)

var (
	generateConfigPath string
)

var generateBundlesCmd = &cobra.Command{
	Use:   "bundles",
	Short: "Write CA certificate bundles to ~/.cg",
	Long: `Write two PEM files under ~/.cg:

  ca.pem     coronagraph CA certificate only (from config)
  bundle.pem system CA bundle plus coronagraph CA

Use bundle.pem with SSL_CERT_FILE, BUNDLE_SSL_CA_CERT, NODE_EXTRA_CA_CERTS, etc.`,
	RunE: runGenerateBundles,
}

func runGenerateBundles(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(generateConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	outputDir, err := bundle.DefaultDir()
	if err != nil {
		return err
	}

	systemPath, err := bundle.SystemCABundlePath()
	if err != nil {
		return err
	}

	caPath, combinedPath, err := bundle.Generate(outputDir, cfg.TLSCertificatePath())
	if err != nil {
		return err
	}

	fmt.Printf("system CA bundle: %s\n", systemPath)
	fmt.Printf("wrote %s\n", caPath)
	fmt.Printf("wrote %s\n", combinedPath)
	return nil
}
