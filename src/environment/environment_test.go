package environment

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForceSetIsInteractive(t *testing.T) {
	ForceSetIsInteractive(true)
	assert.True(t, IsInteractive(), "Expected IsInteractive to return true when overridden with true")

	ForceSetIsInteractive(false)
	assert.False(t, IsInteractive(), "Expected IsInteractive to return false when overridden with false")
}
