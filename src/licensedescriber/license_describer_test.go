package licensedescriber

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	helpers_test "github.com/eriklarko/license-checker-go/src/helpers"
	"github.com/gocolly/colly"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLicenseSummary(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// Create a mock server responding with a copy of a real tldrlegal.com page
	httpMock := helpers_test.NewMockServer()
	defer httpMock.Close()

	err := httpMock.AddFileResponse(
		fmt.Sprintf("%s/license/apache-license-2-0-apache-2-0", httpMock.URL()),
		"https___www.tldrlegal.com_license_apache-license-2-0-apache-2-0.html",
	)
	require.NoError(t, err)

	sut := &TLDRLegalLicenseDescriber{
		urlPattern: fmt.Sprintf("%s/license/%%s", httpMock.URL()),
		collyObj: colly.NewCollector(
			colly.AllowedDomains(strings.TrimPrefix(httpMock.URL(), "http://")),
		),
	}

	summary, err := sut.GetLicenseSummary("APACHE-2.0")
	require.NoError(t, err)

	// Assert that the summary is not empty and contains expected content
	assert.Contains(t, summary.Summary, "You can do what you like with the software")
	assert.NotContains(t, strings.ToLower(summary.Summary), "disclaimer")

	assert.ElementsMatch(t, []string{
		"Commercial Use",
		"Modify",
		"Distribute",
		"Sublicense",
		"Place Warranty",
		"Private Use",
		"Use Patent Claims",
	}, summary.Can)
	assert.ElementsMatch(t, []string{
		"Hold Liable",
		"Use Trademark",
	}, summary.Cannot)
	assert.ElementsMatch(t, []string{
		"Include Copyright",
		"Include License",
		"State Changes",
		"Include Notice",
	}, summary.Must)
}

func TestGetLicenseSummary_MissingSummary(t *testing.T) {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))

	// Create a mock server responding with HTML without the expected content
	httpMock := helpers_test.NewMockServer()
	defer httpMock.Close()

	httpMock.AddStringResponse(
		fmt.Sprintf("%s/license/apache-license-2-0-apache-2-0", httpMock.URL()),
		"<html><body>All your base</body></html>",
	)

	sut := &TLDRLegalLicenseDescriber{
		urlPattern: fmt.Sprintf("%s/license/%%s", httpMock.URL()),
		collyObj: colly.NewCollector(
			colly.AllowedDomains(strings.TrimPrefix(httpMock.URL(), "http://")),
		),
	}

	summary, err := sut.GetLicenseSummary("apache-2.0")
	require.NoError(t, err) // it's not an error if the summary is not found
	assert.Empty(t, summary)
}
