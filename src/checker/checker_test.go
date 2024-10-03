package checker

import (
	"testing"

	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLicenseAllowed(t *testing.T) {
	t.Run("only known licenses", func(t *testing.T) {
		allowedLicenses := []string{"MIT", "Apache-2.0"}
		disallowedLicenses := []string{"GPL-3.0"}

		tests := map[string]bool{
			"MIT":     true,
			"GPL-3.0": false,

			"MIT && Apache-2.0": true,
			"MIT || GPL-3.0":    true,
		}

		lc := NewFromLists(allowedLicenses, disallowedLicenses)
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
		lc := NewFromLists(allowedLicenses, disallowedLicenses)

		var errUnknownLicense *UnknownLicenseError
		_, err := lc.IsLicenseAllowed("unknown")
		assert.ErrorAs(t, err, &errUnknownLicense)
	})
}

func TestUpdate(t *testing.T) {
	license := "MIT"
	lc := NewFromMap(
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

func TestValidateCurrentLicenses(t *testing.T) {
	allowedLicenses := []string{"MIT", "Apache-2.0"}
	disallowedLicenses := []string{"GPL-3.0"}

	lc := NewFromLists(allowedLicenses, disallowedLicenses)

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

func TestNewFromFile(t *testing.T) {
	t.Run("valid content", func(t *testing.T) {
		content := `MIT,true
GPL-3.0,false
`
		licensesFile := helpers_test.CreateTempFileWithContents(t, content)

		lc, err := NewFromFile(licensesFile)
		require.NoError(t, err)

		expected := map[string]bool{
			"MIT":     true,
			"GPL-3.0": false,
		}
		assert.Equal(t, expected, lc.context)
	})

	t.Run("invalid content", func(t *testing.T) {
		content := `MIT,true
GPL-3.0,notabool`
		licensesFile := helpers_test.CreateTempFileWithContents(t, content)

		_, err := NewFromFile(licensesFile)
		assert.Error(t, err)
	})
}

func TestWrite(t *testing.T) {
	licenseDecisions := map[string]bool{
		"MIT":        true,
		"Apache-2.0": true,
		"GPL-3.0":    false,
	}
	lc := NewFromMap(licenseDecisions)

	file := helpers_test.CreateTempFile(t, "licenses.yaml").Name()
	err := lc.Write(file)
	require.NoError(t, err)

	helpers_test.AssertYamlFileExists(t, file, licenseDecisions)
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
