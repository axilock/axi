package hooks

import (
	"slices"

	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/filesio"
	"github.com/axilock/axi/internal/git"
	"github.com/axilock/axi/internal/utils"
	pb "github.com/axilock/axilock-protos/client"
	"google.golang.org/grpc"
)

type CorruptedPrePushError struct {
	FilePath string
}

func (e CorruptedPrePushError) Error() string {
	return "pre-push file corrupted. Please delete " + e.FilePath
}

func Catchall(conn *grpc.ClientConn, home, name, version string, args ...string) error {
	var logger = context.Background().Logger()

	if !git.IsInsideGitRepo() {
		// possible git init invocation
		return nil
	}

	localHooksDir, err := getLocalHooksDir()
	if err != nil {
		return err
	}

	logger.V(1).Info("Local hooks dir: " + localHooksDir)

	if !filesio.DirExists(localHooksDir) {
		// possible git clone, hooks dir not ready yet
		return nil
	}

	hooksDir, err := git.GetCoreHooksPath()
	if err != nil {
		return err
	}

	localHooksDirRelToGitToplevel, err := git.DirRelToGitTopLevel(localHooksDir)
	if err != nil {
		return err
	}
	if hooksDir == localHooksDirRelToGitToplevel {
		// Multiple runs triggered
		// possibly due to hooks like reference-transaction etc
		// TODO: something...
	} else {

		if err := installAxiHook(home, localHooksDir); err != nil {
			return err
		}

		// FIXME: Highlight this!!
		if err := git.SetLocalCoreHooksPath(localHooksDirRelToGitToplevel); err != nil {
			logger.Error(err, "Could not deregsiter global hooks. User local hooks will not work!")
			return err
		}

		metadata := utils.GetSystemMetadataJson(version)
		client := pb.NewMetadataServiceClient(conn)

		// FIXME: what if remote is not origin ? #7
		repourl, _ := git.GetRemoteUrl("origin")
		_, err := client.RepoMetadata(context.Background(), &pb.MetadataRepoRequest{
			RepoUrl:  repourl,
			Metadata: metadata,
		})
		if err != nil {
			logger.Error(err, "Could not send repo metadata")
		}
	}

	hook := Hook{Name: name}
	if err := hook.RunIfExists(args...); err != nil {
		return err
	}

	// TODO: Store this repo path locally for uninstallation
	return nil
}

func installAxiHook(home, localHooksDir string) error {
	var logger = context.Background().Logger()

	localHooksDirRelToGitTopLevel, err := git.DirRelToGitTopLevel(localHooksDir)
	if err != nil {
		return err
	}

	logger.V(1).Info("Local hooks dir: " + localHooksDir)
	logger.V(1).Info("Local hooks dir relative to git top level: " + localHooksDirRelToGitTopLevel)

	afs := filesio.AxiFS{Home: home}
	if err := assertHooksDirs(localHooksDirRelToGitTopLevel, afs.HooksDir()); err != nil {
		return err
	}

	if err := createOrUpdatePrePushHook(
		home,
		localHooksDir,
	); err != nil {
		logger.Error(err, "Could not validate existing local pre push hook")
		return err
	}

	logger.Info("Axi pre-push hook installed")
	return nil
}

func assertHooksDirs(local, global string) error {
	dir, err := git.GetCoreHooksPath()
	expectedDirs := []string{local, global}
	if err != nil {
		return err
	}
	if slices.Contains(expectedDirs, dir) {
		return nil
	}

	dir, err = git.GetLocalCoreHooksPath()
	if err != nil {
		return err
	}
	if dir != "" {
		return &ErrUnsupportedConfiguration{Current: dir,
			Expected: expectedDirs,
			Reason:   "Cannot use a custom local hooks directory. Please run git config --local --unset core.hooksPath to resolve"}
	}

	return &ErrUnsupportedConfiguration{
		Current:  dir,
		Expected: expectedDirs,
		Reason:   "Cannot use a custom global hooks directory. Please run git config --global core.hooksPath \"" + global + "\""}
}
