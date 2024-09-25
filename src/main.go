package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/eriklarko/license-checker/src/checker"
	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/environment"
	"github.com/eriklarko/license-checker/src/tui"
)

const defaultConfigFilePath = ".license-checker.yaml"

var configFilePath = flag.String("config-file", defaultConfigFilePath, "Path to the config file")
var licensesScript = flag.String("licenses-script", "", "Path to the script for getting current licenses")
var licensesFile = flag.String("licenses-file", "", "Path to the file containing approved and disapproved licenses")
var interactive = flag.Bool("interactive", false, "Force the script into interactive mode")

func main() {
	// Parse command line flags
	flag.Parse()

	// Set interactive mode if the flag is provided
	if *interactive {
		environment.ForceSetIsInteractive(*interactive)
	}

	// Load the config
	config, err := setUpConfig()
	if err != nil {
		panic(err)
	}

	licenseChecker, err := setUpLicenseChecker(config)
	if err != nil {
		panic(err)
	}

	/////////////////////////////////////////////////
	//////// SET UP IS DONE, time to do work ////////
	licenses, err := getCurrentLicenses(config.LicensesScript)
	if err != nil {
		panic(err)
	}

	report, err := licenseChecker.ValidateCurrentLicenses(licenses)
	if err != nil {
		panic(err)
	}
	if len(report.Disallowed) > 0 {
		slog.Error("Some licenses are not allowed", "licenses", report.Disallowed)
		os.Exit(1)
	}
	/////////////////////////////////////////////////

}

func setUpConfig() (*config.Config, error) {
	config, err := config.LoadConfig(*configFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	overwriteConfigWithCmdLineFlags(config)

	err = config.Validate()
	if err != nil {
		return nil, fmt.Errorf("config is invalid: %w", err)
	}

	return config, nil
}

func overwriteConfigWithCmdLineFlags(config *config.Config) {
	if config.LicensesScript == "" {
		config.LicensesScript = *licensesScript
	}
	if config.LicensesFile == "" {
		config.LicensesFile = *licensesFile
	}
}

func setUpLicenseChecker(config *config.Config) (*checker.LicenseChecker, error) {
	fmt.Printf("Reading existing decisions from %s\n", config.LicensesFile)
	licenseMap, err := config.ReadLicenseMap()
	if err != nil {
		return nil, fmt.Errorf("failed to read license map: %w", err)
	}

	tui := tui.New()
	return checker.NewFromMap(licenseMap, func(license, dependency string) bool {
		return handleUnknownLicense(dependency, license, tui)
	}), nil
}

// TODO: Move and test
func getCurrentLicenses(script string) (map[string]string, error) {
	cmd := exec.Command(script)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout for script %s: %w", script, err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command %s: %w", script, err)
	}

	licenses := make(map[string]string)
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()

		parts := strings.Split(line, ",")
		licenses[parts[0]] = parts[1]
	}

	err = cmd.Wait()
	if err != nil {
		return nil, fmt.Errorf("failed to wait for command %s to finish: %w", script, err)
	}

	return licenses, nil
}

// TODO: Test
func handleUnknownLicense(
	license, dependency string,
	tui *tui.TUI,
) bool {
	if environment.IsInteractive() {
		return tui.AskForever(
			"Unknown license %s from %s detected. Should it be allowed? [y/N]: ",
			license,
			dependency,
		)
	} else {
		// TODO: verify hint
		slog.Warn("Unknown license detected. To decide if the license is allowed or not, please run this tool again interactively.",
			"license", license,
			"hint", "For example, run `./license-checker .` from the project root.",
		)
		return false
	}

}
