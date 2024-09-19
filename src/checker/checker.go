package checker

import (
	"fmt"

	"github.com/eriklarko/license-checker/src/boolexpr"
)

type LicenseChecker struct {
	context map[string]bool
}

func NewLicenseChecker(allowedLicenses, disallowedLicenses []string) *LicenseChecker {
	return &LicenseChecker{
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

	solution, err := node.Solve(lc.context)
	if err != nil {
		return solution, fmt.Errorf("failed to solve license '%s': %w", license, err)
	}

	return solution, nil
}
