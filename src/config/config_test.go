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
	assert.Equal(t, os.FileMode(0644).String(), fileInfo.Mode().String())
}
