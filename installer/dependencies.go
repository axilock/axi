package installer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/axilock/axi/internal/filesio"
)

// InstallTrufflehog downloads and installs trufflehog in the axi directory
func InstallTrufflehog(home string) error {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	afs := filesio.AxiFS{Home: home}
	binDir := afs.BinaryDir()

	version := "3.89.1"
	var url string

	switch goos {
	case "linux":
		url = fmt.Sprintf("https://github.com/trufflesecurity/trufflehog/releases/download/v%s/trufflehog_%s_Linux_%s.tar.gz", version, version, goarch)
	case "darwin":
		url = fmt.Sprintf("https://github.com/trufflesecurity/trufflehog/releases/download/v%s/trufflehog_%s_darwin_%s.tar.gz", version, version, goarch)
	case "windows":
		url = fmt.Sprintf("https://github.com/trufflesecurity/trufflehog/releases/download/v%s/trufflehog_%s_Windows_%s.zip", version, version, goarch)
	default:
		return fmt.Errorf("unsupported OS: %s", goos)
	}

	tmpFile := filepath.Join(binDir, "trufflehog_download")
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download trufflehog: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("failed to save trufflehog: %v", err)
	}

	if goos == "linux" || goos == "darwin" {
		cmd := exec.Command("tar", "-xzf", tmpFile, "-C", binDir, "trufflehog", "LICENSE")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to extract tar.gz: %v", err)
		}
		os.Rename(filepath.Join(binDir, "LICENSE"), filepath.Join(binDir, "trufflehog_LICENSE"))
	} else if goos == "windows" {
		//FIXME
		return fmt.Errorf("windows extraction not implemented")
	}

	os.Remove(tmpFile)
	return nil
}
