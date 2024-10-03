package tui

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/samber/lo"
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

func (t *TUI) Println(a ...any) {
	_, err := fmt.Fprintln(t.output, a...)
	if err != nil {
		t.onPrintError(err)
	}
}

func (t *TUI) Printf(format string, a ...any) {
	_, err := fmt.Fprintf(t.output, format, a...)
	if err != nil {
		t.onPrintError(err)
	}
}

func (t *TUI) onPrintError(err error) {
	slog.Error("failed to write output", "error", err)
	panic("failed to write output: " + err.Error())
}

func (t *TUI) scanln(a ...any) {
	_, err := fmt.Fscanln(t.input, a...)

	// TODO: if unexpected newline
	if err != nil {
		slog.Error("failed to read user input", "error", err)
		panic("failed to read user input: " + err.Error())
	}
}

func (t *TUI) PrintList(header string, listItems []any, indicator string) {
	t.Println(header)
	for i, item := range listItems {
		indicatorString := indicator
		if indicator == "#" {
			currentNumber := i + 1
			currentNumberString := fmt.Sprintf("%d", currentNumber)
			// use numbers for the indicator
			highestNumber := len(listItems)
			widestNumber := len(fmt.Sprintf("%d", highestNumber))

			currentNumberWidth := len(currentNumberString)
			widthDifference := widestNumber - currentNumberWidth

			indicatorString = strings.Repeat(" ", widthDifference) + currentNumberString + ":"
		}
		t.Printf(" %s %s\n", indicatorString, item)
	}
}

func (t *TUI) Ask(question string, possibleAnswers ...string) string {
	if len(possibleAnswers) > 5 {
		index := t.AskMultipleChoice(question, possibleAnswers...)
		return possibleAnswers[index]
	}

	for {
		// few enough answers, print them on one line
		questionWithAnswers := fmt.Sprintf("%s [%s]: ", question, strings.Join(possibleAnswers, "/"))
		t.Printf(questionWithAnswers)

		var response string
		t.scanln(&response)
		if lo.Contains(possibleAnswers, response) {
			return response
		}
	}
}

func (t *TUI) AskForeverWithPreamble(preamble, repeatingQuestion string, a ...any) bool {
	t.Println(preamble)
	return t.AskYesNo(repeatingQuestion, a...)
}

func (t *TUI) AskYesNo(question string, a ...any) bool {
	for {
		t.Printf(question+" [y/n]", a...)

		var response string
		t.scanln(&response)

		switch strings.ToLower(response) {
		case "y", "yes":
			return true
		case "n", "no", "":
			return false
		}
	}
}

func (t *TUI) AskMultipleChoice(question string, answers ...string) int {
	t.Println(question)
	for i, answer := range answers {
		t.Printf("%d) %s\n", i+1, answer)
	}

	for {
		t.Printf("Please enter a number between 1 and %d: ", len(answers))

		var response int
		t.scanln(&response)

		if response > 0 && response <= len(answers) {
			t.Println() // print an empty line to separate the question from whatever comes next
			return response - 1
		}
	}
}

func (t *TUI) FilePicker(msg string) string {
	if strings.HasSuffix(msg, ":") {
		msg += " "
	} else if !strings.HasSuffix(msg, ": ") {
		msg += ": "
	}

	var response string
	for {
		t.Println(msg)
		t.scanln(&response)

		if response == "" {
			continue
		}

		if _, err := os.Stat(response); os.IsNotExist(err) {
			t.Println("File does not exist, please try again.")
			continue
		}

		break
	}

	return response
}

func (t *TUI) NewProgressIndicator() *ProgressIndicator {
	return newProgressIndicator(t)
}

// ProgressIndicator is a simple progress bar that can be used to show progress
// of a long-running operation.
type ProgressIndicator struct {
	tui             *TUI
	lastPrintedLine string
}

func newProgressIndicator(tui *TUI) *ProgressIndicator {
	return &ProgressIndicator{tui: tui}
}

func (p *ProgressIndicator) SetProgress(current, total int) {
	percentage := float64(current) / float64(total) * 100

	// clear last line
	p.tui.Printf("\r%s", strings.Repeat(" ", len(p.lastPrintedLine)))

	p.lastPrintedLine = fmt.Sprintf("Progress: %.2f%%", percentage) // a bit of a lie, but it's fine :)
	p.tui.Printf("\r%s", p.lastPrintedLine)

	if current == total {
		p.tui.Println()
	}
}
