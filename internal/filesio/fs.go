package filesio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axilock/axi/internal/context"
)

const (
	AxiBinaryName = "axi"
)

type AxiFS struct {
	Home   string
	apiKey string
}

// (Re)Create axi file system.
// NOTE: This deletes all existing axi directories and files within them.
// Following files are preserved:
// Files existing directly in axi home
// The axi binary dir (since it is supposed to be copied later)
func (s *AxiFS) Create() error {
	var logger = context.Background().Logger()

	dirs := []string{
		s.BinaryDir(),
		s.HooksDir(),
	}

	for _, dir := range dirs {
		if dir == s.BinaryDir() && FileExists(s.BinaryPath()) {
			continue
		}

		_, err := os.Stat(dir)
		if err == nil { // Dir already exists, remove
			os.RemoveAll(dir)
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Error(err, "Could not create directory for axi binary. Tried creating: "+dir)
			return err
		}

	}

	return nil
}

func (s *AxiFS) Delete() error {
	var logger = context.Background().Logger()

	if !strings.HasSuffix(s.Home, ".axi") {
		fmt.Println("Axi Home dir does not end with .axi. Refusing to delete") // safety
		return nil
	}

	logger.Info("Deleting axi home", "home", s.Home)
	return os.RemoveAll(s.Home)
}

func (s *AxiFS) BinaryPath() string {
	return filepath.Join(s.BinaryDir(), s.BinaryName())
}

func (s *AxiFS) BinaryDir() string {
	return filepath.Join(s.Home, "bin")
}

func (s *AxiFS) BinaryName() string {
	return AxiBinaryName
}

func (s *AxiFS) APIKeyPath() string {
	return filepath.Join(s.Home, "api_key")
}

func (s *AxiFS) HooksDir() string {
	return filepath.Join(s.Home, "hooks")
}

func (s *AxiFS) WriteAPIKey(apiKey string) error {
	return WriteAPIKey(s.APIKeyPath(), apiKey)
}

func (s *AxiFS) APIKey() (string, error) {
	if s.apiKey != "" {
		return s.apiKey, nil
	}

	keyb, err := os.ReadFile(s.APIKeyPath())
	if err != nil {
		return "", err
	}
	s.apiKey = strings.TrimSpace(string(keyb))
	return s.apiKey, nil
}
