package bundle

import "path/filepath"

// BinDir returns ~/.cg/bin for generated command shims.
func BinDir() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bin"), nil
}

// CombinedBundlePath returns ~/.cg/bundle.pem.
func CombinedBundlePath() (string, error) {
	dir, err := DefaultDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, CombinedBundleName), nil
}
