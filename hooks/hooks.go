package hooks

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/filesio"
	"github.com/axilock/axi/internal/git"
)

const (
	DefaultLocalHooksDir = ".git/hooks"
)

type Hook struct {
	Name string
}

type HookError struct {
	ExitCode int
	CausedBy error
	Hook     *Hook
}

func (e *HookError) Error() string {
	return fmt.Sprintf("Error while running hook %s. Exit code: %d. Reason: %s", e.Hook.Name, e.ExitCode, e.CausedBy.Error())
}

func HooksDir() (string, error) {
	/*
		// check local config
		dir, err := git.GetLocalCoreHooksPath()
		if err == nil && dir != "" {
			return dir, nil
		}

		// check global config
		dir, err = git.GetGlobalCoreHooksPath()
		if err == nil {
			return dir, nil
		}
	*/
	dir, err := git.GetCoreHooksPath()
	if err != nil {
		return "", err
	}
	if dir != "" {
		return dir, nil
	}

	// local & global configs don't exist, fallback to local hooks dir
	return getLocalHooksDir()
}

func (h *Hook) Path() (string, error) {
	dir, err := HooksDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(dir, h.Name), nil
}

func (h *Hook) Exists() bool {
	path, err := h.Path()
	if err != nil {
		return false
	}
	return filesio.FileExists(path)
}

func (h *Hook) Error(exitCode int, causedBy error) *HookError {
	return &HookError{ExitCode: exitCode, CausedBy: causedBy, Hook: h}
}

func (h *Hook) Run(args ...string) *HookError {
	path, err := h.Path()
	if err != nil {
		return h.Error(1, err)
	}

	cmd := exec.Command(path, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err, ok := err.(*exec.ExitError); ok {
		return h.Error(err.ExitCode(), err)
	}
	if err != nil {
		return h.Error(-1, err)
	}
	return nil
}

func (h *Hook) RunIfExists(args ...string) *HookError {
	var logger = context.Background().Logger()

	if h.Exists() {
		if err := h.Run(args...); err != nil {
			logger.Error(err, "failed to run hook: "+h.Name)
			return err
		}
	}
	return nil

}

func MatchFile(filename, script string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	headerLines := strings.Split(script, "\n")
	i := 0
	for ; i < len(headerLines); i++ {
		if !scanner.Scan() {
			if headerLines[i] == "" && i == len(headerLines)-1 { // Trailing newline
				return nil
			}
			return &ErrTextMismatch{
				File:     filename,
				LineNo:   i + 1,
				Expected: headerLines[i],
				Found:    "",
				Hint:     "Target file had fewer lines (" + strconv.Itoa(i) + ") than headerLines (" + strconv.Itoa(len(headerLines)) + ")",
			}
		}
		if line := scanner.Text(); line != headerLines[i] {
			return &ErrTextMismatch{
				File:   filename,
				LineNo: i + 1, Expected: headerLines[i],
				Found: truncate(line, 100),
				Hint:  "Line mismatch",
			}
		}
	}

	if scanner.Scan() { //
		text := scanner.Text()
		if text == "" {
			text = "\\n"
		}
		return &ErrTextMismatch{
			File:     filename,
			LineNo:   i + 1,
			Expected: "",
			Found:    truncate(text, 100),
			Hint:     "target file had more lines than headerLines",
		}
	}
	return nil
}

func UpdateHook(filename, header string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	newContent := header + "\n" + string(content)
	return os.WriteFile(filename, []byte(newContent), 0755)
}

// Get local hooks dir: $GIT_DIR/hooks (absolute path)
func getLocalHooksDir() (string, error) {
	git_dir, err := git.GitDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(git_dir, "hooks")

	absHooksDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	realAbsHooksDir, err := filepath.EvalSymlinks(absHooksDir)
	if err != nil {
		return "", err
	}

	return realAbsHooksDir, nil
}

// Get hooks dir relative to git config as current hooks dir might be set to
// $SEKRIT_HOME/hook by global hooks or be empty
func LocalHooksDirRelToGitTopLevel() (string, error) {
	absHooksDir, err := getLocalHooksDir()
	if err != nil {
		return "", err
	}

	return git.DirRelToGitTopLevel(absHooksDir)

}

func truncate(str string, length int) string {
	if len(str) > length {
		str = str[:length]
	}
	return str
}
