//go:build !darwin

package cmd

func authenticate(reason string) bool {
	return false
}
