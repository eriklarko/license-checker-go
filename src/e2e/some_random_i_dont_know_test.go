package e2e_test

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"

	licensechecker "github.com/eriklarko/license-checker/src/checker"
	"github.com/stretchr/testify/require"
)

func TestLicenseChecker(t *testing.T) {
	// Set up allowed and disallowed licenses
	allowedLicenses := []string{"MIT", "Apache-2.0"}
	disallowedLicenses := []string{"GPL-3.0", "BSD-3-Clause"}

	// Create a new LicenseChecker
	checker := licensechecker.NewLicenseChecker(allowedLicenses, disallowedLicenses)

	// Read the go-licenses_license.csv file
	licenses, err := ReadLicenseFile("go-licenses_licenses.csv")
	if err != nil {
		t.Fatalf("Failed to read license file: %v", err)
	}

	// Check if each license is allowed or not
	for dependency, license := range licenses {
		allowed, err := checker.IsLicenseAllowed(license)
		require.NoError(t, err)

		if allowed {
			t.Logf("License %s is allowed (%s)", license, dependency)
		} else {
			t.Errorf("License %s is not allowed (%s)", license, dependency)
		}
	}

	t.Fail()
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
