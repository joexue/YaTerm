package pty

import (
	"errors"
	"os/exec"
	"strings"
)

var errNotStarted = errors.New("pty: shell not started yet")

// DefaultProgram is the program a terminal runs when none is set: PowerShell
// on Windows, bash elsewhere
func DefaultProgram() string {
	return defaultProgram
}

// SplitCommand splits a command line into the program and its arguments, at
// spaces. Quotes keep spaces in a word, e.g. a path like
// "C:\Program Files\PowerShell\7\pwsh.exe". Backslashes are no escapes, so
// Windows paths work as written.
func SplitCommand(cmd string) []string {
	var (
		words   []string
		word    strings.Builder
		inQuote rune
		inWord  bool
	)
	for _, r := range cmd {
		switch {
		case inQuote != 0:
			if r == inQuote {
				inQuote = 0
			} else {
				word.WriteRune(r)
			}
		case r == '"' || r == '\'':
			inQuote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(r)
			inWord = true
		}
	}
	if inWord {
		words = append(words, word.String())
	}
	return words
}

// FindProgram returns the program of a command line with its full path, found
// in PATH when it has none, and the arguments. ok is false when there is no
// such program.
func FindProgram(cmd string) (path string, args []string, ok bool) {
	words := SplitCommand(cmd)
	if len(words) == 0 {
		return "", nil, false
	}
	path, err := exec.LookPath(words[0])
	if err != nil {
		return "", nil, false
	}
	return path, words[1:], true
}
