package e2e_test

import (
	"bytes"
	"encoding/csv"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	licensechecker "github.com/eriklarko/license-checker/src/checker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseChecker(t *testing.T) {
	// Capture log output
	var logOutput bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&logOutput, nil)))

	// Set up allowed and disallowed licenses
	allowedLicenses := []string{"MIT", "Apache-2.0"}
	disallowedLicenses := []string{"GPL-3.0", "BSD-3-Clause"}

	// Create a new LicenseChecker
	checker := licensechecker.NewFromLists(allowedLicenses, disallowedLicenses)

	// Read the go-licenses_license.csv file
	licenses, err := ReadLicenseFile("go-licenses_licenses.csv")
	if err != nil {
		t.Fatalf("Failed to read license file: %v", err)
	}

	report, err := checker.ValidateCurrentLicenses(licenses)
	require.NoError(t, err)

	expectedAllowed := map[string][]string{
		"Apache-2.0": {
			"github.com/golang/glog",
			"github.com/google/go-licenses",
			"go.opencensus.io",
			"github.com/google/licenseclassifier/stringclassifier",
			"gopkg.in/src-d/go-billy.v4",
			"gopkg.in/src-d/go-git.v4",
			"github.com/golang/groupcache/lru",
			"github.com/spf13/cobra",
			"github.com/google/licenseclassifier",
			"github.com/xanzy/ssh-agent",
		},
		"MIT": {
			"github.com/mitchellh/go-homedir",
			"github.com/sergi/go-diff/diffmatchpatch",
			"github.com/jbenet/go-context/io",
			"github.com/otiai10/copy",
			"github.com/kevinburke/ssh_config",
		},
	}
	expectedDisallowed := map[string][]string{
		"BSD-2-Clause": {
			"github.com/emirpasic/gods",
			"gopkg.in/warnings.v0",
		},
		"BSD-3-Clause": {
			"golang.org/x/crypto",
			"golang.org/x/tools",
			"golang.org/x/xerrors",
			"github.com/spf13/pflag",
			"github.com/src-d/gcfg",
			"golang.org/x/mod/semver",
			"golang.org/x/net",
			"github.com/google/go-licenses/internal/third_party/pkgsite",
			"golang.org/x/sys",
		},
	}
	assertMapsEqual(t, expectedAllowed, report.Allowed)
	assertMapsEqual(t, expectedDisallowed, report.Disallowed)
}

func assertMapsEqual(t *testing.T, expected, actual map[string][]string) {
	t.Helper()

	t.Logf("expected: %v", expected)
	t.Logf("actual: %v", actual)

	assert.Equal(t, len(expected), len(actual), "maps have different lengths")

	for key, expectedValues := range expected {
		actualValues, ok := actual[key]
		assert.True(t, ok, "key %q not found in actual map", key)
		assert.ElementsMatch(t, expectedValues, actualValues)
	}
}

func ReadLicenseFile(filename string) (map[string]string, error) {
	// Convert the filename into an absolute path
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	// Open the file
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Create a map to store the licenses
	licenses := make(map[string]string)

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read the CSV records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Iterate over the records
	for _, record := range records {
		// Extract the dependency name and license from each record
		dependency := record[0]
		license := record[2]

		// Add the dependency and license to the map
		licenses[dependency] = license
	}

	return licenses, nil
}
