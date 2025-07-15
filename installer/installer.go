package installer

import (
	"os"
	"path/filepath"

	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/filesio"
	"github.com/axilock/axi/internal/git"
)

var hooks []string = []string{
	"applypatch-msg",
	"commit-msg",
	"fsmonitor-watchman",
	"post-applypatch",
	"post-checkout",
	"post-commit",
	"post-merge",
	"post-receive",
	"post-rewrite",
	"post-update",
	"pre-applypatch",
	"pre-auto-gc",
	"pre-commit",
	"pre-merge-commit",
	"pre-push",
	"pre-rebase",
	"pre-receive",
	"prepare-commit-msg",
	"push-to-checkout",
	"reference-transaction",
	"sendemail-validate",
	"update",
}

func Install(home, apiKey string) error {
	var logger = context.Background().Logger()

	//TODO: Coverage install started
	afs := filesio.AxiFS{Home: home}

	currentName, err := os.Executable()
	if err != nil {
		logger.Error(err, "Couldn't find current executable's name.")
		return err
	}

	targetBinaryName := afs.BinaryPath()

	if !filesio.SameFile(currentName, targetBinaryName) { // Reinstall
		err = filesio.CopyBinary(currentName, targetBinaryName)
		if err != nil {
			logger.Error(err, "Error saving axi binary")
			return err
		}
	}

	err = afs.WriteAPIKey(apiKey)
	if err != nil {
		logger.Error(err, "Error writing API key")
		return err
	}

	// Currently all hooks are symlinks to axi binary itself
	for _, hook := range hooks {
		if err := installHook(hook, afs.HooksDir(), afs.BinaryPath()); err != nil {
			logger.Error(err, "Error writing hook:"+hook)
			return err
		}
		logger.Info("Hook installed: " + hook)
	}

	// FIXME: Should we error here or continue with a warning ?
	// Errors in unattended installations are very bad
	if err := assertEmptyOrAxiGlobalCoreHooksDir(afs.HooksDir()); err != nil {
		return err
	}

	logger.Info("Setting git config --global core.hooksPath to: " + afs.HooksDir())
	if err = git.SetGlobalCoreHooksPath(afs.HooksDir()); err != nil {
		logger.Error(err, "Error setting global git hooks path")
		return err
	}

	logger.Info("Success. Send coverage data.")

	//TODO: Coverage install completed
	return err
}

func assertEmptyOrAxiGlobalCoreHooksDir(expectedPath string) error {
	path, err := git.GetGlobalCoreHooksPath()
	if err != nil {
		return err
	}
	if path != "" && path != expectedPath {
		return &ErrUnsupportedConfiguration{Current: path,
			Expected: []string{"<empty>", expectedPath},
			Reason: "Cannot use global core hooks path. " +
				"Please run git config --global --unset core.hooksPath and restart installation"}
	}
	return nil
}

func installHook(name, dir, hookBinary string) error {
	return os.Symlink(hookBinary, filepath.Join(dir, name))
}
