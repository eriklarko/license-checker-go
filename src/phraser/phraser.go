package phraser

import (
	"fmt"
	"math/rand"
)

type Phraser struct {
	phrases           []string
	lastPhraseIdx     int
	hasShuffledBefore bool
}

// New creates a new Phraser with the given phrases. It will always start with
// the first phrase, and then choose randomly between the other phrases for
// subsequent calls to Get.
//
// Usage:
//   phraser := phraser.New([]string{
//     "To start let's look at license %s",
//     "Next up license %s",
//     "Let's look at %s",
//   })
//
//   for someCondition {
//      license := ...
//      fmt.Println(phraser.Get(license)) // prints one of the phrases, always starting with the first one
//   }

func New(phrases []string) *Phraser {
	return &Phraser{
		phrases:           phrases,
		lastPhraseIdx:     -1,
		hasShuffledBefore: false,
	}
}

func (p *Phraser) shuffle() {
	if len(p.phrases) < 2 {
		// no point shuffling fewer than 2 phrases
		return
	}

	// copy the phrases so we don't shuffle the original slice
	shuffledPhrases := make([]string, len(p.phrases))
	copy(shuffledPhrases, p.phrases)

	// and shuffle the copy
	rand.Shuffle(len(shuffledPhrases), func(i, j int) {
		shuffledPhrases[i], shuffledPhrases[j] = shuffledPhrases[j], shuffledPhrases[i]
	})

	p.phrases = shuffledPhrases
}

func (p *Phraser) Get(formatArgs ...any) string {
	if len(p.phrases) == 0 {
		return ""
	}
	if len(p.phrases) == 1 {
		return fmt.Sprintf(p.phrases[0], formatArgs...)
	}

	p.lastPhraseIdx++
	if p.lastPhraseIdx >= len(p.phrases) {
		// we've used all phrases, shuffle and start again
		if !p.hasShuffledBefore {
			// this is the first time we're shuffling the phrases, the first
			// phrase is still in `p.phrases` so we need to remove that so we
			// don't ever return it again
			p.phrases = p.phrases[1:]
			p.hasShuffledBefore = true
		}
		p.shuffle()
		p.lastPhraseIdx = 0
	}

	return fmt.Sprintf(p.phrases[p.lastPhraseIdx], formatArgs...)
}
