package config

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// required values
	LicensesScript string `yaml:"licenses-script"`
	LicensesFile   string `yaml:"licenses-file"`
	CacheDir       string `yaml:"cache-dir"`

	// optional values
	CuratedListsSource  string `yaml:"curated-list-source"`
	SelectedCuratedList string `yaml:"selected-curated-list,omitempty"`

	CuratedScriptsSource string `yaml:"curated-scripts-source,omitempty"`

	// the file this config was read from
	Path string `yaml:"-"` // not serialized
}

func DefaultConfig() *Config {
	conf := &Config{}
	conf.applyDefaults()
	return conf
}

func LoadConfig(path string) (*Config, error) {
	path = tryMakeAbsolute(path)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("config file %s is not valid yaml: %s", path, err)
	}

	(&config).applyDefaults()

	config.Path = tryMakeAbsolute(path)

	return &config, nil
}

// TODO: test
func (c *Config) applyDefaults() {
	if c.CacheDir == "" {
		c.CacheDir = ".license-checker"
	}
	if c.LicensesScript == "" {
		c.LicensesScript = filepath.Join(c.CacheDir, "print-current-licenses.sh")
	}
	if c.LicensesFile == "" {
		c.LicensesFile = filepath.Join(c.CacheDir, "licenses.csv")
	}

	if c.CuratedListsSource == "" {
		c.CuratedListsSource = "https://raw.githubusercontent.com/eriklarko/license-checker-go/refs/heads/main/lists/list-metadata.yaml"
	}

	if c.Path == "" {
		c.Path = ".license-checker.yaml"
	}
}

// tryMakeAbsolute converts a relative file path to an absolute file path.
// If the conversion fails, it returns the original relative path.
func tryMakeAbsolute(relativePath string) string {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return relativePath
	}

	return absPath
}

// Validate checks if the config is valid, i.e. all required fields are set.
// Returns nil if the config is valid, otherwise an error message.
func (c *Config) Validate() error {
	var errs []string

	if c.CacheDir == "" {
		errs = append(errs, "cache-dir cannot be empty")
	}
	if c.LicensesScript == "" {
		errs = append(errs, "licenses-script cannot be empty")
	}
	if c.LicensesFile == "" {
		errs = append(errs, "licenses-file cannot be empty")
	}

	if len(errs) > 0 {
		return fmt.Errorf("validation errors: %s", errs)
	}

	return nil
}

// Write writes the config to a file.
func (c *Config) Write() error {
	// Open file for writing, creating it if it doesn't exist. Using 0644 which
	// grants the owner read and write access, while the group members and other
	// system users only have read acces
	file, err := os.OpenFile(c.Path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening/creating file %s: %w", c.Path, err)
	}
	defer file.Close()

	// Make sure the file is only writeable by the owner if the config file
	// already existed before this write
	os.Chmod(c.Path, 0644)

	enc := yaml.NewEncoder(file)
	err = enc.Encode(c)
	if err != nil {
		return fmt.Errorf("failed encoding config as yaml: %w", err)
	}

	return nil
}

// WriteLicenseMap writes a map from license to a boolean indicating whether it
// is allowed or not to a specified file.
func (c *Config) WriteLicenseMap(licenseMap map[string]bool) error {
	path := tryMakeAbsolute(c.LicensesFile)
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
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

// ReadLicenseMap reads a map from license to a boolean indicating whether it is
// allowed or not from a specified file.
func (c *Config) ReadLicenseMap() (map[string]bool, error) {
	path := tryMakeAbsolute(c.LicensesFile)
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read records from file %s: %w", path, err)
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

// TODO: test
func (c *Config) PersistCuratedListChoice(listName string) error {
	c.SelectedCuratedList = listName
	return c.Write()
}

// String returns the YAML representation of the Config struct.
func (c *Config) String() string {
	// TODO: converting to yaml is such a bad idea
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("failed to marshal config: %v", err)
	}
	return string(data)
}
