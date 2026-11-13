package main

import "strings"

// style is a look that a terminal gives to text, as the parameters of an SGR sequence.
//
// https://en.wikipedia.org/wiki/ANSI_escape_code#SGR
type style string

const (
	bold  style = "1"
	faint style = "2"
	red   style = "31"
	green style = "32"
	// cyan is the accent colour.
	cyan style = "36"
)

// pen styles text for a terminal; the zero pen leaves text plain.
type pen bool

func (p pen) paint(look style, text string) string {
	if !p {
		return text
	}
	return "\x1b[" + string(look) + "m" + text + "\x1b[m"
}

// profile gives the accent to the role in a profile's name.
func (p pen) profile(name string) string {
	role := strings.LastIndex(name, "/") + 1
	return name[:role] + p.paint(cyan, name[role:])
}

// bar draws the share of a total that is done.
func (p pen) bar(done, total int) string {
	const width = 24
	filled := width * done / total
	return p.paint(cyan, strings.Repeat("━", filled)) + p.paint(faint, strings.Repeat("─", width-filled))
}

// hyperlink is underlined text that a terminal opens as a link on a click.
//
// https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
func hyperlink(text, url string) string {
	return "\x1b]8;;" + url + "\x1b\\\x1b[4m" + text + "\x1b[24m\x1b]8;;\x1b\\"
}
