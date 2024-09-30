package e2e_test

import (
	"encoding/csv"
	"os"
	"path/filepath"
)

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
