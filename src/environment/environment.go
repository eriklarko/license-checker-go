package environment

import (
	"os"

	"github.com/mattn/go-isatty"
)

var interactiveOverride *bool

// ForceSetIsInteractive allows overriding the interactive check, TODO: remember to use this from the CLI
// Sith++
func ForceSetIsInteractive(value bool) {
	interactiveOverride = &value
}

// IsInteractive returns true if the code is run by a user with an interactive shell, false otherwise
func IsInteractive() bool {
	if interactiveOverride != nil {
		return *interactiveOverride
	}
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}
