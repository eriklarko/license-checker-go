package helpers_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// CreateTempFile creates a temporary file in the test's temporary directory,
// and automatically removes it when the test is done.
func CreateTempFile(t *testing.T, fileName string) *os.File {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), fileName)
	require.NoError(t, err)

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile
}

// CreateTempFileWithContents creates a temporary file in the test's temporary
// directory, writes the given content to it, and automatically removes it when
// the test is done.
func CreateTempFileWithContents(t *testing.T, content string) string {
	t.Helper()

	tmpFile := createFileWithContents(t, content)

	err := tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func createFileWithContents(t *testing.T, content string) *os.File {
	t.Helper()

	tmpFile := CreateTempFile(t, "license-checker-test-*")

	_, err := tmpFile.Write([]byte(content))
	require.NoError(t, err)

	return tmpFile
}

func CreateTempScript(t *testing.T, content string) string {
	t.Helper()

	tmpFile := CreateTempFile(t, "license-checker-test-*.sh")

	_, err := tmpFile.Write([]byte(content))
	require.NoError(t, err)
	// make script executable
	tmpFile.Chmod(0711)

	err = tmpFile.Close()
	require.NoError(t, err)

	return tmpFile.Name()
}

func AssertYamlFileExists(t *testing.T, path string, content map[string]any) {
	t.Helper()

	assert.FileExists(t, path)

	// check that the file contains the expected content
	fileContent, err := os.ReadFile(path)
	require.NoError(t, err)

	contentBytes, err := yaml.Marshal(content)
	require.NoError(t, err)

	assert.YAMLEq(t, string(contentBytes), string(fileContent))
}
