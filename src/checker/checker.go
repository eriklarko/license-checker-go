package checker

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/eriklarko/license-checker/src/boolexpr"
	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/environment"
)

type LicenseChecker struct {
	config  *config.Config // st st st stutter
	context map[string]bool
}

func NewLicenseChecker(config *config.Config, allowedLicenses, disallowedLicenses []string) *LicenseChecker {
	return &LicenseChecker{
		config:  config,
		context: buildContext(allowedLicenses, disallowedLicenses),
	}
}

func buildContext(approvedLicenses []string, disallowedLicenses []string) map[string]bool {
	context := make(map[string]bool)

	for _, license := range approvedLicenses {
		context[license] = true
	}

	for _, license := range disallowedLicenses {
		context[license] = false
	}

	return context
}

func (lc *LicenseChecker) IsLicenseAllowed(license string) (bool, error) {
	node, err := boolexpr.New(license)
	if err != nil {
		return false, fmt.Errorf("failed to parse license '%s': %w", license, err)
	}

	var errUnknownVar *boolexpr.UnknownVariableError
	solution, err := node.Solve(lc.context)
	if errors.As(err, &errUnknownVar) {
		// When running interactively this is an error, when running
		// on CI it's not, it's just defaulting to not allowing things it
		// doesn't know about. Good sense.
		if environment.IsInteractive() {
			for {
				if askForever("Unknown license detected. Should it be allowed? [y/N]: ") {
					// user allowed license
					lc.context[license] = true
				} else {
					// user disallowed license
					lc.context[license] = false
				}

				lc.config.WriteLicenseMapToCSV(lc.context)
			}
		} else {
			// TODO: verify hint
			slog.Warn("Unknown license detected. To decide if the license is allowed or not, please run this tool again interactively.",
				"license", license,
				"hint", "For example, run `./license-checker .` from the project root.",
			)
			return false, nil
		}
	} else if err != nil {
		return solution, fmt.Errorf("failed to solve license '%s': %w", license, err)
	}

	return solution, nil
}

// TODO: Move to some cui package
func askForever(question string) bool {
	var response string
	for {
		fmt.Print(question)
		_, err := fmt.Scanln(&response)

		if err != nil {
			slog.Error("failed to read user input", "error", err)
			panic("failed to read user input: " + err.Error())
		}

		switch strings.ToLower(response) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		}
	}
}
