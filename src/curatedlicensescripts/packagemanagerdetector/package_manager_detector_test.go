package packagemanagerdetector_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/curatedlicensescripts/packagemanagerdetector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageManagerDetector_Detect(t *testing.T) {
	tests := []struct {
		name             string
		files            []string
		expectedManagers []string
	}{
		{
			name:             "No package manager files",
			files:            []string{},
			expectedManagers: []string{},
		},
		{
			name:             "NPM package manager file",
			files:            []string{"package.json"},
			expectedManagers: []string{"npm"},
		},
		{
			name:             "Go package manager file",
			files:            []string{"go.mod"},
			expectedManagers: []string{"go modules"},
		},
		{
			name:             "Multiple package manager files",
			files:            []string{"package.json", "go.mod", "requirements.txt"},
			expectedManagers: []string{"npm", "go modules", "pip"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the specified files in a temporary directory
			dir := t.TempDir()
			for _, file := range tt.files {
				filePath := filepath.Join(dir, file)
				err := os.WriteFile(filePath, []byte{}, 0644)
				require.NoError(t, err)
			}

			sut := packagemanagerdetector.New(dir)

			// Act
			detectedManagers, err := sut.FindLikelyPackageManagers()
			require.NoError(t, err)
			assert.ElementsMatch(t, tt.expectedManagers, detectedManagers)
		})
	}
}
