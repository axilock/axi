package git

import (
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/axilock/axi/internal/context"
)

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(b []byte) (err error) {
	date, err := time.Parse(`"2006-01-02 15:04:05 -0700"`, string(b))
	if err != nil {
		return err
	}
	t.Time = date
	return
}

type Commit struct {
	ID     string
	Author string
	Time   time.Time
}

// Set local core hooks path
// NOTE: Must use a relative path to git top level
// This ensures local config stays local to the repo
func SetLocalCoreHooksPath(relPath string) error {
	_, err := execGitConfig("--local", "core.hooksPath", relPath)
	return err
}

func SetGlobalCoreHooksPath(absPath string) error {
	_, err := execGitConfig("--global", "core.hooksPath", absPath)
	return err
}

func UnsetLocalCoreHooksPath() error {
	_, err := execGitConfig("--local", "--unset", "core.hooksPath")
	return err
}

func UnsetGlobalCoreHooksPath() error {
	_, err := execGitConfig("--global", "--unset", "core.hooksPath")
	return err
}
func GetLocalCoreHooksPath() (string, error) {
	return execGitConfig("--local", "core.hooksPath")
}

func GetGlobalCoreHooksPath() (string, error) {
	return execGitConfig("--global", "core.hooksPath")
}

func GetCoreHooksPath() (string, error) {
	return execGitConfig("core.hooksPath")
}

func GetRemoteUrl(name string) (string, error) {
	return execGitConfig("--get", "remote."+name+".url")
}

func GetRevList(fromTo ...string) (string, error) {
	if len(fromTo) == 1 {
		return execGit("rev-list", fromTo[0])
	}
	return execGit("rev-list", fromTo[0], fromTo[1])
}

// Return commit list with author
// FIXME: since can be empty string, in this case return all commits reachable from
// current branch
func GetCommitsList(since, branch string) []Commit {
	var log string

	if since == "" {
		log, _ = execGit("log", "--pretty=format:%H%x00%ce%x00%ai", "--date=iso-strict", branch)
	} else {
		log, _ = execGit("log", "--pretty=format:%H%x00%ce%x00%ai", "--date=iso-strict", since+".."+branch)
	}

	var commits []Commit
	lines := strings.Split(log, "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\x00")
		if len(parts) != 3 {
			continue
		}

		t, err := time.Parse(`2006-01-02 15:04:05 -0700`, string(parts[2]))
		if err != nil {
			t = time.Time{}
		}
		commits = append(commits, Commit{
			ID:     parts[0],
			Author: parts[1],
			Time:   t,
		})
	}
	return commits

}

// LastPushedCommitReachableByBranch returns the SHA of the latest commit that is both
// in your local HEAD and present on *any* remote in your repo.
// NOTE: Can return empty in case of errors (no branch etc) or root branch
// FIXME: In large repos, this can be slow. We should consider limiting
// rev-list output
func LastPushedCommitReachableByBranch(branch string) (string, error) {
	localCommitsStr, err := execGit("rev-list", "--date-order", branch)
	if err != nil {
		return "", err
	}
	localCommits := strings.Split(localCommitsStr, "\n")

	if len(localCommits) == 0 {
		return "", nil
	}

	notRemoteStr, err := execGit("rev-list", branch, "--not", "--remotes")
	if err != nil {
		return "", err
	}
	notRemoteSet := make(map[string]bool)
	for _, c := range strings.Split(notRemoteStr, "\n") {
		if c != "" {
			notRemoteSet[c] = true
		}
	}

	for _, commit := range localCommits {
		if commit != "" && !notRemoteSet[commit] {
			return commit, nil
		}
	}

	// root commit
	return "", nil
}

func execGitConfig(args ...string) (string, error) {
	gitArgs := append([]string{"config", "--null"}, args...)

	stdout, err := execGit(gitArgs...)
	stdout = strings.TrimRight(stdout, "\000")
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			switch e.ExitCode() {
			case 1: // key is not present in a get operation
				return "", nil
			case 5: // you try to unset an option which does not exist
				return "", nil
			}
		}
		return "", err
	}
	return stdout, nil
}

func execGit(args ...string) (string, error) {
	var logger = context.Background().Logger()

	logger.V(1).Info("Running git " + strings.Join(args, " "))

	var stdout bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = io.Discard
	err := cmd.Run()
	return strings.TrimSpace(stdout.String()), err
}

func IsZeroHash(hash string) bool {
	if hash[0] != '0' { //fail fast
		return false
	}
	n, err := strconv.Atoi(hash)
	if err != nil {
		return false
	}

	if n == 0 {
		return true
	}

	return false
}

func IsInsideGitRepo() bool {
	stdout, _ := execGit("rev-parse", "--is-inside-work-tree")
	return stdout == "true"
}

func GitDir() (string, error) {
	return execGit("rev-parse", "--git-dir")
}

func GitTopLevel() (string, error) {
	return execGit("rev-parse", "--show-toplevel")
}

func DirRelToGitTopLevel(absDir string) (string, error) {
	topLevel, err := GitTopLevel()
	if err != nil {
		return "", err
	}

	relDir, err := filepath.Rel(topLevel, absDir)
	if err != nil {
		return "", err
	}

	return relDir, nil
}
