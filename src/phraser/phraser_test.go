package phraser_test

import (
	"testing"

	"github.com/eriklarko/license-checker/src/phraser"
	"github.com/montanaflynn/stats"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPhraser_Get(t *testing.T) {
	phrases := []string{"1", "2", "3", "4", "5", "6"}

	t.Run("first call returns first phrases", func(t *testing.T) {
		phraser := phraser.New(phrases)

		result := phraser.Get()
		assert.Equal(t, "1", result)
	})

	t.Run("all phrases are seen", func(t *testing.T) {
		phraser := phraser.New(phrases)

		seen := make(map[string]bool)
		for i := 0; i < len(phrases); i++ {
			result := phraser.Get()
			seen[result] = true
		}

		assert.ElementsMatch(t, phrases, lo.Keys(seen))
	})

	t.Run("random distribution", func(t *testing.T) {
		phraser := phraser.New(phrases)

		seen := make(map[string]int)
		for i := 0; i < len(phrases)*100; i++ {
			result := phraser.Get()
			seen[result]++
		}
		t.Logf("Times seen each phrase: %v", seen)

		// expect first phrase to be seen once
		numTimesSeenFirstPhrase := seen[phrases[0]]
		assert.Equal(t, 1, numTimesSeenFirstPhrase, "First phrase should be seen once")

		// expect the other phrases to be seen about the same number of times
		// calculate variance for the phrases
		delete(seen, phrases[0]) // remove the first phrase
		varianceInput := lo.Map(
			lo.Values(seen),
			func(timesSeen int, i int) float64 {
				return float64(timesSeen)
			},
		)
		variance, err := stats.Variance(varianceInput)
		require.NoError(t, err)
		// check that variance is low enough
		assert.InDelta(t, 1, variance, 0.9)
	})

	t.Run("handles empty phrases", func(t *testing.T) {
		phraser := phraser.New([]string{})

		result := phraser.Get()
		assert.Equal(t, "", result)
	})

	t.Run("handles single phrase", func(t *testing.T) {
		phraser := phraser.New([]string{"Only phrase"})

		seen := make(map[string]int)
		for i := 0; i < 100; i++ {
			result := phraser.Get()
			seen[result]++
		}

		assert.Equal(t, map[string]int{"Only phrase": 100}, seen)
	})
}

func TestPhraser_Get_FormatArgs(t *testing.T) {

	t.Run("only one phrase", func(t *testing.T) {
		phraser := phraser.New([]string{"foo %s"})

		result := phraser.Get("bar")
		assert.Equal(t, "foo bar", result)
	})

	t.Run("multiple phrases", func(t *testing.T) {
		phrases := []string{"foo %s", "bar %s"}
		phraser := phraser.New(phrases)

		seen := make(map[string]bool)
		for range phrases {
			result := phraser.Get("baz")
			seen[result] = true
		}
		assert.Equal(t, []string{"foo baz", "bar baz"}, lo.Keys(seen))
	})
}
