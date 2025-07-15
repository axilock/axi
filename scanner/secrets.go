package scanner

import (
	"fmt"
	"os"

	"github.com/axilock/axi/internal/git"
)

type Secret struct {
	Commit   git.Commit
	Value    string
	Redacted string
	File     string
	Line     int
	Type     string
}

func (s *Secret) Print() {
	fmt.Fprintf(os.Stderr, "---- Secret Details --- :\n")
	fmt.Fprint(os.Stderr, s.String())
}

func (s *Secret) StringWithPrefix(prefix string) string {
	return fmt.Sprintf(
		prefix+"Commit ID: %s\n"+
			prefix+"Redacted value: %.10s\n"+
			prefix+"File: %s\n"+
			prefix+"Line: %d\n"+
			prefix+"Type: %s\n",
		s.Commit.ID, s.Value, s.File, s.Line, s.Type)
}

func (s *Secret) String() string {
	return s.StringWithPrefix("")
}

type SecretScanner interface {
	Run(dir, sinceCommit, branch string) ([]Secret, error)
}

type ScanError struct {
	scanner string
	reason  string
}

func (s *ScanError) Error() string {
	return s.scanner + " faced errors. Results might be incomplete. " + s.reason
}
