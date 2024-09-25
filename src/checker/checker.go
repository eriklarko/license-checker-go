package checker

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/eriklarko/license-checker/src/boolexpr"
)

type UnknownLicenseError struct {
	License string
}

func (ule *UnknownLicenseError) Error() string {
	return fmt.Sprintf("unknown license '%s'", ule.License)
}

type UnknownLicenseHandler func(license, dependency string) bool

type LicenseChecker struct {
	context               map[string]bool
	unknownLicenseHandler UnknownLicenseHandler
}

func NewFromMap(context map[string]bool, onUnknownLicense UnknownLicenseHandler) *LicenseChecker {
	return &LicenseChecker{
		context:               context,
		unknownLicenseHandler: onUnknownLicense,
	}
}

func NewFromLists(allowedLicenses, disallowedLicenses []string, onUnknownLicense UnknownLicenseHandler) *LicenseChecker {
	return NewFromMap(
		buildContext(allowedLicenses, disallowedLicenses),
		onUnknownLicense,
	)
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

// Update updates the license decision for a dependency
func (lc *LicenseChecker) Update(license string, isAllowed bool) {
	lc.context[license] = isAllowed
}

func (lc *LicenseChecker) IsLicenseAllowed(license string) (bool, error) {
	node, err := boolexpr.New(license)
	if err != nil {
		return false, fmt.Errorf("failed to parse license '%s': %w", license, err)
	}

	var errUnknownVar *boolexpr.UnknownVariableError
	solution, err := node.Solve(lc.context)
	if errors.As(err, &errUnknownVar) {
		return false, &UnknownLicenseError{License: license}
	} else if err != nil {
		return solution, fmt.Errorf("failed to solve license '%s': %w", license, err)
	}

	return solution, nil
}

// TODO: test tesst test
func (lc *LicenseChecker) ValidateCurrentLicenses(currentLicenses map[string]string) (*Report, error) {
	report := &Report{}
	for dependency, license := range currentLicenses {
		slog.Debug("Checking license", "license", license, "dependency", dependency)

		var errUnknownLicense *UnknownLicenseError
		allowed, err := lc.IsLicenseAllowed(license)

		if errors.As(err, &errUnknownLicense) {
			unknownIsAllowed := lc.unknownLicenseHandler(
				license,
				dependency,
			)

			// store the decision so we don't have to ask again
			lc.Update(license, unknownIsAllowed)

			allowed = unknownIsAllowed
		} else if err != nil {
			return nil, fmt.Errorf("failed to check if license is allowed or not: %w", err)
		}

		report.RecordDecision(license, dependency, allowed)
	}

	return report, nil
}
