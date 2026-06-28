package bundle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	CACertFilename     = "ca.pem"
	CombinedBundleName = "bundle.pem"
)

// DefaultDir returns ~/.cg.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cg"), nil
}

// Generate writes ca.pem (coronagraph CA only) and bundle.pem (system CAs + coronagraph CA)
// into outputDir.
func Generate(outputDir, coronagraphCACertPath string) (caPath, combinedPath string, err error) {
	coronagraphCA, err := os.ReadFile(coronagraphCACertPath)
	if err != nil {
		return "", "", fmt.Errorf("read coronagraph CA: %w", err)
	}

	systemBundlePath, err := SystemCABundlePath()
	if err != nil {
		return "", "", err
	}

	systemCA, err := os.ReadFile(systemBundlePath)
	if err != nil {
		return "", "", fmt.Errorf("read system CA bundle %q: %w", systemBundlePath, err)
	}

	if err := os.MkdirAll(outputDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create output dir: %w", err)
	}

	caPath = filepath.Join(outputDir, CACertFilename)
	if err := os.WriteFile(caPath, ensureTrailingNewline(coronagraphCA), 0o644); err != nil {
		return "", "", fmt.Errorf("write %s: %w", caPath, err)
	}

	combined := append(ensureTrailingNewline(systemCA), ensureTrailingNewline(coronagraphCA)...)
	combinedPath = filepath.Join(outputDir, CombinedBundleName)
	if err := os.WriteFile(combinedPath, combined, 0o644); err != nil {
		return "", "", fmt.Errorf("write %s: %w", combinedPath, err)
	}

	return caPath, combinedPath, nil
}

// SystemCABundlePath returns the path to the platform OpenSSL-style CA bundle PEM file.
func SystemCABundlePath() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SSL_CERT_FILE")); path != "" {
		if err := validateBundleFile(path); err != nil {
			return "", fmt.Errorf("SSL_CERT_FILE: %w", err)
		}
		return path, nil
	}

	if path, err := rubyOpenSSLDefaultCertFile(); err == nil {
		if err := validateBundleFile(path); err == nil {
			return path, nil
		}
	}

	for _, path := range defaultCABundleCandidates() {
		if err := validateBundleFile(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("could not locate system CA bundle (set SSL_CERT_FILE or install CA certificates)")
}

func rubyOpenSSLDefaultCertFile() (string, error) {
	out, err := exec.Command("ruby", "-ropenssl", "-e", "puts OpenSSL::X509::DEFAULT_CERT_FILE").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func defaultCABundleCandidates() []string {
	return []string{
		"/etc/ssl/cert.pem",                     // macOS system export
		"/etc/ssl/certs/ca-certificates.crt",    // Debian/Ubuntu
		"/etc/pki/tls/certs/ca-bundle.crt",      // RHEL/Fedora
		"/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
		"/opt/homebrew/etc/openssl@3/cert.pem",   // Homebrew OpenSSL 3 (macOS)
		"/opt/homebrew/etc/openssl@1.1/cert.pem", // Homebrew OpenSSL 1.1 (macOS)
		"/usr/local/etc/openssl@3/cert.pem",
		"/usr/local/etc/openssl@1.1/cert.pem",
	}
}

func validateBundleFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", path)
	}
	return nil
}

func ensureTrailingNewline(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	if data[len(data)-1] == '\n' {
		return data
	}
	out := make([]byte, len(data)+1)
	copy(out, data)
	out[len(data)] = '\n'
	return out
}
