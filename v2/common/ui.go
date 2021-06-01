package common

import (
	"time"

	"github.com/pterm/pterm"
)

func NewSpinner() pterm.SpinnerPrinter {
	s := pterm.DefaultSpinner
	s.Sequence = []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}
	s.Delay = 100 * time.Millisecond
	s.Style = pterm.NewStyle(pterm.FgMagenta)
	e := &pterm.Error
	e.ShowLineNumber = false
	s.FailPrinter = e
	return s
}

func DisableSpinnerOutput() {
	pterm.DisableOutput()
}
