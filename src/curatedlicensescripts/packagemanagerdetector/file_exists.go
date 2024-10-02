package packagemanagerdetector

import (
	"fmt"
	"os"
)

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("failed to check if file '%s' exists: %w", path, err)
	}

	return true, nil
}
