package filesio

import (
	"io"
	"os"
	"path/filepath"

	"github.com/axilock/axi/internal/context"
)

func CopyBinary(srcPath, destPath string) (err error) {
	var logger = context.Background().Logger()

	if err != nil {
		logger.Error(err, "Error getting executable path")
		return
	}

	absExecPath, err := filepath.Abs(srcPath)
	if err != nil {
		logger.Error(err, "Error getting absolute path of executable")
		return
	}

	sourceFile, err := os.Open(absExecPath)
	if err != nil {
		logger.Error(err, "Error opening source file")
		return
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		logger.Error(err, "Error creating destination file")
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		logger.Error(err, "Error copying file")
		return
	}

	sourceInfo, err := os.Stat(absExecPath)
	if err != nil {
		logger.Error(err, "Error getting source file info")
		return
	}

	err = os.Chmod(destPath, sourceInfo.Mode())
	if err != nil {
		logger.Error(err, "Error setting permissions on copied file")
		return
	}

	return nil
}

func WriteAPIKey(location, apiKey string) error {
	return os.WriteFile(location, []byte(apiKey), 0644)
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func DirExists(filename string) bool {
	var logger = context.Background().Logger()
	info, err := os.Stat(filename)
	if err != nil {
		// FIXME: handle this better
		logger.Error(err, "Fatal error checking if directory exists")
		return false
	}
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

func WriteExecutableFileWithContent(filename, content string) error {
	return atomicWrite(filename, content, 0755)
}

func SameFile(f1_name, f2_name string) bool {
	f1, err := os.Stat(f1_name)
	if err != nil {
		return false
	}

	f2, err := os.Stat(f2_name)
	if err != nil {
		return false
	}

	return os.SameFile(f1, f2)
}

func atomicWrite(filename, data string, mode os.FileMode) error {
	dir := filepath.Dir(filename)

	tmpFile, err := os.CreateTemp(dir, "tmp-axi-hook-*")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.WriteString(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return err
	}
	tmpFile.Close()

	if err := os.Chmod(tmpName, mode); err != nil {
		return err
	}
	return os.Rename(tmpName, filename)
}
