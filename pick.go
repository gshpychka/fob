package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// fzfPath is the fzf that draws the picker.
var fzfPath = "fzf"

var errNoChoice = errors.New("no profile chosen")

// choose returns the line the user picks out of lines; the output of a refresh command takes their place once it is complete.
//
// https://junegunn.github.io/fzf/reference/
func choose(ctx context.Context, lines []string, refresh string) (string, error) {
	fzf := exec.CommandContext(ctx, fzfPath)
	if refresh != "" {
		// fzf raises load for every list it finishes reading, so the binding unbinds itself.
		fzf.Args = append(fzf.Args, "--bind", "load:unbind(load)+reload-sync("+refresh+")")
	}
	fzf.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	fzf.Stderr = os.Stderr
	choice, err := fzf.Output()
	// fzf exits with 1 when nothing matches the query and with 130 when the user leaves it.
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
		return "", errNoChoice
	}
	if err != nil {
		return "", fmt.Errorf("fzf: %w", err)
	}
	return strings.TrimSpace(string(choice)), nil
}
