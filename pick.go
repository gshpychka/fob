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

var errNoChoice = errors.New("the picker closed before a choice was made")

// choose returns the profile name the user picks; the picker lists each distinct name as it arrives.
//
// https://junegunn.github.io/fzf/reference/
func choose(ctx context.Context, lines io.Reader) (string, error) {
	// fzf renders the colours in its text where the terminal shows colour, and prints the choice without them.
	ink := pen(true)
	fzf := exec.CommandContext(ctx, fzfPath, "--ansi",
		// The picker opens under the cursor, with the prompt at its top and the best match right under it.
		"--height=50%", "--layout=reverse",
		"--border=rounded", "--border-label="+ink.paint(cyan, " "+program+" "), "--border-label-pos=3",
		"--prompt="+ink.paint(cyan, "▸")+" ", "--ghost=account or role",
		"--info=inline-right", "--no-separator", "--highlight-line",
	)
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
				fmt.Fprintln(list, ink.profile(line))
			}
		}
	}()
	fzf.Stderr = os.Stderr
	choice, err := fzf.Output()
	// fzf exits with 1 when the query filters every line out and with 130 when the user leaves it.
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
		return "", errNoChoice
	}
	if err != nil {
		return "", fmt.Errorf("fzf: %w", err)
	}
	return strings.TrimSpace(string(choice)), nil
}
