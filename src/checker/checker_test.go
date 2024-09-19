package checker_test

import (
	"testing"

	"github.com/eriklarko/license-checker/src/checker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseChecker_IsLicenseAllowed(t *testing.T) {
	allowedLicenses := []string{"MIT", "Apache-2.0"}
	disallowedLicenses := []string{"GPL-3.0"}

	tests := map[string]bool{
		"MIT":     true,
		"GPL-3.0": false,

		"Unknown": false,

		"MIT && Apache-2.0": true,
		"MIT || GPL-3.0":    true,

		"!GPL-3.0": true,
	}

	lc := checker.NewLicenseChecker(allowedLicenses, disallowedLicenses)
	for name, expected := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := lc.IsLicenseAllowed(name)
			require.NoError(t, err)

			assert.Equal(t, expected, result)
		})
	}
}
