package checker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordAllowed(t *testing.T) {
	report := &Report{}
	report.RecordAllowed("MIT", "github.com/example/repo")

	assert.Equal(
		t,
		map[string][]string{
			"MIT": {
				"github.com/example/repo",
			},
		},
		report.Allowed,
	)
}

func TestRecordDisallowed(t *testing.T) {
	report := &Report{}
	report.RecordDisallowed("MIT", "github.com/example/repo")

	assert.Equal(
		t,
		map[string][]string{
			"MIT": {
				"github.com/example/repo",
			},
		},
		report.Disallowed,
	)
}

func TestRecordUnknownLicense(t *testing.T) {
	report := &Report{}
	report.RecordUnknownLicense("MIT", "github.com/example/repo")

	assert.Equal(
		t,
		map[string][]string{
			"MIT": {
				"github.com/example/repo",
			},
		},
		report.Unknown,
	)
}

func TestRecordDecision(t *testing.T) {
	report := &Report{}
	report.RecordDecision("MIT", "github.com/example/repo1", true)
	report.RecordDecision("GPL", "github.com/example/repo2", false)

	assert.Equal(
		t,
		map[string][]string{
			"MIT": {
				"github.com/example/repo1",
			},
		},
		report.Allowed,
	)
	assert.Equal(
		t,
		map[string][]string{
			"GPL": {
				"github.com/example/repo2",
			},
		},
		report.Disallowed,
	)
}
