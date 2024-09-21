package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteLicenseMapToCSV(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "/test_licenses.csv")
	defer os.Remove(filePath)

	config := NewConfig(filePath)
	licenseMap := map[string]bool{
		"MIT":     true,
		"GPL-3.0": false,
	}

	err := config.WriteLicenseMapToCSV(licenseMap)
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(filePath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "MIT,true\n")
	assert.Contains(t, string(content), "GPL-3.0,false\n")
}

func TestReadLicenseMapFromCSV(t *testing.T) {
	content := `MIT,true
GPL-3.0,false
`
	filePath := createTempFile(t, content)
	defer os.Remove(filePath)

	config := NewConfig(filePath)
	licenseMap, err := config.ReadLicenseMapFromCSV()
	require.NoError(t, err)

	expected := map[string]bool{
		"MIT":     true,
		"GPL-3.0": false,
	}
	assert.Equal(t, expected, licenseMap)
}

func TestReadLicenseMapFromCSVInvalidContent(t *testing.T) {
	content := `MIT,true
GPL-3.0,notabool
`
	filePath := createTempFile(t, content)
	defer os.Remove(filePath)

	config := NewConfig(filePath)
	_, err := config.ReadLicenseMapFromCSV()
	assert.Error(t, err)
}

func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "test_licenses_*.csv")
	require.NoError(t, err)

	_, err = tmpFile.Write([]byte(content))
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}
