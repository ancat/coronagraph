//go:build !darwin

package sudo

// Authenticate is not supported on this platform.
func Authenticate(reason string) bool {
	return false
}
