package main

import (
	"bufio"
	"cmp"
	"fmt"
	"io"
	"strings"
)

// prompter puts questions to the user.
type prompter struct {
	in  *bufio.Reader
	out io.Writer
	pen pen
}

// ask returns the user's answer to a question; an empty answer accepts the suggestion.
func (p prompter) ask(question, suggestion string) (string, error) {
	if suggestion != "" {
		question += p.pen.paint(faint, " ["+suggestion+"]")
	}
	for {
		fmt.Fprint(p.out, p.pen.paint(cyan, "▸"), " ", question, ": ")
		line, err := p.in.ReadString('\n')
		if answer := cmp.Or(strings.TrimSpace(line), suggestion); answer != "" {
			return answer, nil
		}
		if err != nil {
			return "", fmt.Errorf("reading the answer: %w", err)
		}
	}
}

// confirm reports whether the user answers yes.
func (p prompter) confirm(question string) bool {
	fmt.Fprint(p.out, p.pen.paint(cyan, "▸"), " ", question, p.pen.paint(faint, " [y/N] "))
	line, _ := p.in.ReadString('\n')
	return strings.EqualFold(strings.TrimSpace(line), "y")
}
