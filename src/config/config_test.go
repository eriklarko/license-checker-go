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
		content := `licenses-file: "test_licenses.yaml"
licenses-script: "test_script.sh"`
		configFile := helpers_test.CreateTempFileWithContents(t, content)

		config, err := LoadConfig(configFile)
		require.NoError(t, err)

		assert.Equal(t, "test_licenses.yaml", config.LicensesFile)
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
		LicensesFile:   "test_licenses.yaml",
		LicensesScript: "test_script.sh",

		Path: configFile,
	}

	err := config.Write()
	require.NoError(t, err)

	// Verify file content
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)

	assert.Contains(t, string(content), "licenses-file: test_licenses.yaml\n")
	assert.Contains(t, string(content), "licenses-script: test_script.sh\n")

	// verify permissions
	fileInfo, err := os.Stat(configFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644).String(), fileInfo.Mode().String())
}

func TestApplyDefaults(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		config := &Config{}
		config.applyDefaults()

		assert.Equal(t, ".license-checker", config.CacheDir)
		assert.Equal(t, ".license-checker/print-current-licenses.sh", config.LicensesScript)
		assert.Equal(t, ".license-checker/licenses.yaml", config.LicensesFile)
	})

	t.Run("does not override existing values", func(t *testing.T) {
		config := &Config{
			CacheDir:       ".cache",
			LicensesFile:   "test_licenses.yaml",
			LicensesScript: "test_script.sh",
		}
		config.applyDefaults()

		assert.Equal(t, ".cache", config.CacheDir)
		assert.Equal(t, "test_licenses.yaml", config.LicensesFile)
		assert.Equal(t, "test_script.sh", config.LicensesScript)
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &Config{
			LicensesFile:   "licenses.yaml",
			LicensesScript: "script.sh",
			CacheDir:       ".cache",
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing LicensesFile", func(t *testing.T) {
		config := &Config{
			CacheDir:       ".cache",
			LicensesFile:   "",
			LicensesScript: "script.sh",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "licenses-file")
	})

	t.Run("missing LicensesScript", func(t *testing.T) {
		config := &Config{
			CacheDir:       ".cache",
			LicensesFile:   "licenses.yaml",
			LicensesScript: "",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "licenses-script")
	})

	t.Run("missing CacheDir", func(t *testing.T) {
		config := &Config{
			CacheDir:       "",
			LicensesFile:   "licenses.yaml",
			LicensesScript: "script.sh",
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cache-dir")
	})
}
