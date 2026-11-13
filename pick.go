package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// fzfPath is the fzf that draws the picker.
var fzfPath = "fzf"

var errNoChoice = errors.New("no profile chosen")

// choose returns the line the user picks; the picker lists each distinct line as it arrives.
//
// https://junegunn.github.io/fzf/reference/
func choose(ctx context.Context, lines io.Reader) (string, error) {
	// The reverse layout puts the prompt at the top and the best match right under it.
	fzf := exec.CommandContext(ctx, fzfPath, "--layout=reverse")
	list, err := fzf.StdinPipe()
	if err != nil {
		return "", err
	}
	go func() {
		defer list.Close()
		listed := map[string]bool{}
		for scanner := bufio.NewScanner(lines); scanner.Scan(); {
			if line := scanner.Text(); !listed[line] {
				listed[line] = true
				fmt.Fprintln(list, line)
			}
		}
	}()
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
