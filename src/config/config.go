package config

import (
	"fmt"
	"os"
	"path/filepath"

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

	CuratedScriptsSource  string `yaml:"curated-scripts-source,omitempty"`
	SelectedCuratedScript string `yaml:"selected-curated-script,omitempty"`

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

// TODO: Test
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

// String returns the YAML representation of the Config struct.
func (c *Config) String() string {
	// TODO: converting to yaml is such a bad idea
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("failed to marshal config: %v", err)
	}
	return string(data)
}
