package checker_test

import (
	"testing"

	"github.com/eriklarko/license-checker/src/checker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseChecker_IsLicenseAllowed(t *testing.T) {
	t.Run("only known licenses", func(t *testing.T) {
		allowedLicenses := []string{"MIT", "Apache-2.0"}
		disallowedLicenses := []string{"GPL-3.0"}

		tests := map[string]bool{
			"MIT":     true,
			"GPL-3.0": false,

			"MIT && Apache-2.0": true,
			"MIT || GPL-3.0":    true,
		}

		lc := checker.NewFromLists(allowedLicenses, disallowedLicenses)
		for name, expected := range tests {
			t.Run(name, func(t *testing.T) {
				result, err := lc.IsLicenseAllowed(name)
				require.NoError(t, err)

				assert.Equal(t, expected, result)
			})
		}
	})

	t.Run("only unknown licenses", func(t *testing.T) {
		allowedLicenses := []string{}
		disallowedLicenses := []string{}
		lc := checker.NewFromLists(allowedLicenses, disallowedLicenses)

		var errUnknownLicense *checker.UnknownLicenseError
		_, err := lc.IsLicenseAllowed("unknown")
		assert.ErrorAs(t, err, &errUnknownLicense)
	})
}

func TestLicenseChecker_Update(t *testing.T) {
	license := "MIT"
	lc := checker.NewFromMap(
		map[string]bool{
			// disallowed to start with
			license: false,
		},
	)

	isAllowed, err := lc.IsLicenseAllowed(license)
	require.NoError(t, err)
	assert.False(t, isAllowed)

	lc.Update(license, true)

	isAllowed, err = lc.IsLicenseAllowed(license)
	require.NoError(t, err)
	assert.True(t, isAllowed)

}

func TestLicenseChecker_ValidateCurrentLicenses(t *testing.T) {
	allowedLicenses := []string{"MIT", "Apache-2.0"}
	disallowedLicenses := []string{"GPL-3.0"}

	lc := checker.NewFromLists(allowedLicenses, disallowedLicenses)

	currentLicenses := map[string]string{
		"some-dependency-1": "MIT",
		"some-dependency-2": "MIT",
		"some-dependency-3": "Apache-2.0",
		"some-dependency-4": "GPL-2.0",
		"some-dependency-5": "GPL-3.0",
	}
	report, err := lc.ValidateCurrentLicenses(currentLicenses)
	require.NoError(t, err)

	expectedAllowed := map[string][]string{
		"MIT":        {"some-dependency-1", "some-dependency-2"},
		"Apache-2.0": {"some-dependency-3"},
	}
	expectedDisallowed := map[string][]string{
		"GPL-3.0": {"some-dependency-5"},
	}
	expectedUnknown := map[string][]string{
		"GPL-2.0": {"some-dependency-4"},
	}
	assertMapsEqual(t, expectedAllowed, report.Allowed)
	assertMapsEqual(t, expectedDisallowed, report.Disallowed)
	assertMapsEqual(t, expectedUnknown, report.Unknown)
}

func assertMapsEqual(t *testing.T, expected, actual map[string][]string) {
	t.Helper()

	t.Logf("expected: %v", expected)
	t.Logf("actual: %v", actual)

	assert.Equal(t, len(expected), len(actual), "maps have different lengths")

	for key, expectedValues := range expected {
		actualValues, ok := actual[key]
		assert.True(t, ok, "key %q not found in actual map", key)
		assert.ElementsMatch(t, expectedValues, actualValues)
	}
}
