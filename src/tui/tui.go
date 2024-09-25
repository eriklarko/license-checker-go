package tui

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
)

type TUI struct {
	input  io.Reader
	output io.Writer
}

func New() *TUI {
	return &TUI{
		input:  os.Stdin,
		output: os.Stdout,
	}
}

func (t *TUI) SetInput(input *os.File) {
	t.input = input
}

func (t *TUI) SetOutput(output *os.File) {
	t.output = output
}

func (t *TUI) AskForever(question string, a ...any) bool {
	var response string
	for {
		fmt.Fprintf(t.output, question, a...)
		_, err := fmt.Fscanln(t.input, &response)

		if err != nil {
			slog.Error("failed to read user input", "error", err)
			panic("failed to read user input: " + err.Error())
		}

		switch strings.ToLower(response) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		}
	}
}
