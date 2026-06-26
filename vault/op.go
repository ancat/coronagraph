package vault

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

func ReadOP(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", errors.New("1Password reference is empty")
	}

	out, err := exec.Command("op", "read", "-n", ref).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("op read failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}

		return "", fmt.Errorf("op read failed: %w", err)
	}

	return string(out), nil
}
