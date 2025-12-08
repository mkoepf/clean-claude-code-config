package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ConfirmResult represents the result of a confirmation prompt.
type ConfirmResult int

const (
	ConfirmYes ConfirmResult = iota
	ConfirmNo
	ConfirmError
)

// Confirmer handles user confirmation prompts.
type Confirmer struct {
	In  io.Reader
	Out io.Writer
}

// Confirm prompts the user for confirmation and returns the result.
// Default is No (pressing Enter without input returns ConfirmNo).
// Only "y" or "yes" (case-insensitive) returns ConfirmYes.
func (c *Confirmer) Confirm(prompt string) ConfirmResult {
	fmt.Fprint(c.Out, prompt)

	reader := bufio.NewReader(c.In)
	input, err := reader.ReadString('\n')
	if err != nil {
		return ConfirmNo
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "y" || input == "yes" {
		return ConfirmYes
	}

	return ConfirmNo
}

// ConfirmChanges displays a preview and prompts for confirmation.
// If autoYes is true, it displays the preview but skips the prompt.
func ConfirmChanges(preview *Preview, in io.Reader, out io.Writer, autoYes bool) (bool, error) {
	if err := preview.Display(out); err != nil {
		return false, err
	}

	if autoYes {
		return true, nil
	}

	confirmer := &Confirmer{In: in, Out: out}
	result := confirmer.Confirm("\nProceed? [y/N]: ")

	if result != ConfirmYes {
		fmt.Fprintln(out, "Aborted. No changes made.")
		return false, nil
	}

	return true, nil
}
