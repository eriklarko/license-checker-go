package config

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	filePath string
}

func NewConfig(filePath string) *Config {
	return &Config{
		filePath: filePath,
	}
}

// WriteLicenseMapToCSV writes a map from license to a boolean indicating whether it is allowed or not to a CSV file.
func (c *Config) WriteLicenseMapToCSV(licenseMap map[string]bool) error {
	absPath, err := filepath.Abs(c.filePath)
	if err != nil {
		// this isn't an error enough to stop execution. It's just to make it
		// easier for the user to find the file. Best effort.
		absPath = c.filePath
	}

	file, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", absPath, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for license, allowed := range licenseMap {
		record := []string{license, strconv.FormatBool(allowed)}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record %v: %w", record, err)
		}
	}

	return nil
}

// ReadLicenseMapFromCSV reads a map from license to a boolean indicating whether it is allowed or not from a CSV file.
func (c *Config) ReadLicenseMapFromCSV() (map[string]bool, error) {
	absPath, err := filepath.Abs(c.filePath)
	if err != nil {
		absPath = c.filePath
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", absPath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read records from file %s: %w", absPath, err)
	}

	licenseMap := make(map[string]bool)
	for _, record := range records {
		if len(record) != 2 {
			return nil, fmt.Errorf("invalid record %v: expected 2 fields, got %d", record, len(record))
		}

		license := record[0]
		allowed, err := strconv.ParseBool(record[1])
		if err != nil {
			return nil, fmt.Errorf("failed to parse boolean %s: %w", record[1], err)
		}

		licenseMap[license] = allowed
	}

	return licenseMap, nil
}
