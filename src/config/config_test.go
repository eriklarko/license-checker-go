package config

import (
	"os"
	"testing"

	helpers_test "github.com/eriklarko/license-checker/src/helpers"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// write test for LoadConfig
func TestLoadConfig(t *testing.T) {

	t.Run("valid, existing config", func(t *testing.T) {
		content := `licenses-file: "test_licenses.csv"
licenses-script: "test_script.sh"`
		configFile := helpers_test.CreateTempFileWithContents(t, content)

		config, err := LoadConfig(configFile)
		require.NoError(t, err)

		assert.Equal(t, "test_licenses.csv", config.LicensesFile)
		assert.Equal(t, "test_script.sh", config.LicensesScript)
	})

	t.Run("invalid, existing config", func(t *testing.T) {
		content := `foo` // no keys
		configFile := helpers_test.CreateTempFileWithContents(t, content)

		_, err := LoadConfig(configFile)
		assert.False(t, os.IsNotExist(err))
		assert.Error(t, err)
	})

	t.Run("non-existing config", func(t *testing.T) {
		_, err := LoadConfig("non-existing.yaml")
		assert.True(t, os.IsNotExist(err))
	})
}

func TestWriteConfig(t *testing.T) {
	configFile := helpers_test.CreateTempFile(t, "test_config.yaml").Name()

	config := &Config{
		LicensesFile:   "test_licenses.csv",
		LicensesScript: "test_script.sh",

		Path: configFile,
	}

	err := config.Write()
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "licenses-file: test_licenses.csv\n")
	assert.Contains(t, string(content), "licenses-script: test_script.sh\n")

	// verify permissions
	fileInfo, err := os.Stat(configFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), fileInfo.Mode())
}

func TestWriteLicenseMap(t *testing.T) {
	licensesFile := helpers_test.CreateTempFile(t, "test_licenses.csv").Name()

	config := &Config{LicensesFile: licensesFile}
	licenseMap := map[string]bool{
		"MIT":     true,
		"GPL-3.0": false,
	}

	err := config.WriteLicenseMap(licenseMap)
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(licensesFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "MIT,true\n")
	assert.Contains(t, string(content), "GPL-3.0,false\n")
}

func TestReadLicenseMap(t *testing.T) {

	t.Run("valid content", func(t *testing.T) {
		content := `MIT,true
GPL-3.0,false
`
		licensesFile := helpers_test.CreateTempFileWithContents(t, content)

		config := &Config{LicensesFile: licensesFile}
		licenseMap, err := config.ReadLicenseMap()
		require.NoError(t, err)

		expected := map[string]bool{
			"MIT":     true,
			"GPL-3.0": false,
		}
		assert.Equal(t, expected, licenseMap)
	})

	t.Run("invalid content", func(t *testing.T) {
		content := `MIT,true
GPL-3.0,notabool`
		licensesFile := helpers_test.CreateTempFileWithContents(t, content)

		config := &Config{LicensesFile: licensesFile}
		_, err := config.ReadLicenseMap()
		assert.Error(t, err)
	})
}
