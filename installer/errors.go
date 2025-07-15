package installer

import "fmt"

type ErrUnsupportedConfiguration struct {
	Reason   string
	Current  string
	Expected []string
}

func (e *ErrUnsupportedConfiguration) Error() string {
	return fmt.Sprintf("Unsupported installation. Contact axi. Currently set to %s, expected: %s. %s", e.Current, e.Expected, e.Reason)
}
