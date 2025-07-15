package scanner

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/git"
)

type Trufflehog struct {
	name string
	path string
}

type TrufflehogResult struct {
	SourceMetadata struct {
		Data struct {
			Git struct {
				Commit     string   `json:"commit"`
				File       string   `json:"file"`
				Email      string   `json:"email"`
				Repository string   `json:"repository"`
				Timestamp  git.Time `json:"timestamp"`
				Line       int      `json:"line"`
			} `json:"Git"`
		} `json:"Data"`
	} `json:"SourceMetadata"`
	SourceID            int    `json:"SourceID"`
	SourceType          int    `json:"SourceType"`
	SourceName          string `json:"SourceName"`
	DetectorType        int    `json:"DetectorType"`
	DetectorName        string `json:"DetectorName"`
	DetectorDescription string `json:"DetectorDescription"`
	DecoderName         string `json:"DecoderName"`
	Verified            bool   `json:"Verified"`
	Raw                 string `json:"Raw"`
	RawV2               string `json:"RawV2"`
	Redacted            string `json:"Redacted"`
	ExtraData           any    `json:"ExtraData"`
	StructuredData      any    `json:"StructuredData"`
}

func NewTrufflehog(absPath string) *Trufflehog {
	return &Trufflehog{
		name: "trufflehog",
		path: absPath,
	}
}

// branch and sinceCommit could be empty strings if not required
func (t *Trufflehog) Run(dir, sinceCommit, branch string) ([]Secret, error) {
	var logger = context.Background().Logger().WithName("trufflehog")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	trufflehog, err := exec.LookPath(t.path)
	if err != nil {
		return nil, NewTrufflehogNotInstalledError()
	}
	logger.V(1).Info("Trufflehog is at " + trufflehog)

	args := []string{
		trufflehog,
		"git",
		"file://.",
		"--fail",
		"--json",
		"--force-skip-binaries",
		"--force-skip-archives",
		"--no-verification",
		"--no-update",
	}

	if sinceCommit != "" {
		args = append(args, "--since-commit", sinceCommit)
	}

	if branch != "" {
		args = append(args, "--branch", branch)
	}

	cmd := exec.Cmd{
		Path:   trufflehog,
		Args:   args,
		Stdout: &stdout,
		Stderr: &stderr,
		Dir:    dir,
	}

	logger.Info("Running " + t.name + " with args " + strings.Join(args, " "))

	err = cmd.Run()
	logger.V(1).Info("command completed")

	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			logger.Info(t.name + " exited with code " + strconv.Itoa(e.ExitCode()))
			switch e.ExitCode() {
			case 0:
				return nil, nil
			case 1:
				return trufflehogResultsToSecrets(&stdout), &ScanError{scanner: t.name, reason: stderr.String()}
			case 183:
				return trufflehogResultsToSecrets(&stdout), nil
			default:
				err := errors.New("Unknown exit code from trufflehog: " + strconv.Itoa(e.ExitCode()))
				return nil, err
			}
		}
	}

	return nil, err
}

func trufflehogResultsToSecrets(raw *bytes.Buffer) []Secret {
	var secrets []Secret

	eofReached := false
	for {
		if eofReached {
			break
		}
		line, err := raw.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				eofReached = true // ensure last line is read
			}
		}
		var result TrufflehogResult
		if err := json.Unmarshal(line, &result); err != nil {
			continue
		}

		if result.Raw == "" {
			continue
		}

		secrets = append(secrets, Secret{
			Commit: git.Commit{
				ID:     result.SourceMetadata.Data.Git.Commit,
				Author: result.SourceMetadata.Data.Git.Email,
				Time:   result.SourceMetadata.Data.Git.Timestamp.Time,
			},
			Value: result.Raw,
			File:  result.SourceMetadata.Data.Git.File,
			Line:  result.SourceMetadata.Data.Git.Line,
			Type:  result.DetectorName,
		})
	}

	return secrets
}
