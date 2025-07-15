package hooks

import (
	"fmt"
)

type ErrTextMismatch struct {
	File     string
	LineNo   int
	Expected string
	Found    string
	Hint     string
}

func (e *ErrTextMismatch) Error() string {
	return fmt.Sprintf("Text mismatch in file: %s, expected: %s, found: %s, on line: %d, hint:%s",
		e.File, e.Expected, e.Found, e.LineNo, e.Hint)
}

type ErrUnsupportedConfiguration struct {
	Reason   string
	Current  string
	Expected []string
}

func (e *ErrUnsupportedConfiguration) Error() string {
	return fmt.Sprintf("Unsupported installation. Contact axi. Currently set to %s, expected: %s. %s", e.Current, e.Expected, e.Reason)
}

type ErrUnsupportedHook struct {
	Name string
}

func (e *ErrUnsupportedHook) Error() string {
	return "Unsupported hook: " + e.Name
}

type ErrCorruptedHook struct {
	Name       string
	Path       string
	MatchError error
}

func (e *ErrCorruptedHook) Error() string {
	return "Error: " + e.Name + " hook cannot be used. The file is corrupted or non-updatable. \n" +
		"Please delete " + e.Path + " to resolve. \n" +
		"If you need the currently installed hook, move it to " + e.Name + ".user, \n" +
		"eg: mv " + e.Path + " " + e.Path + ".user"
}
