package hooks

import (
	"bufio"
	"os"
	"strings"

	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/git"
	"github.com/axilock/axi/scanner"
)

type PrePushHookOutput struct {
	Commits []git.Commit
	Secrets []scanner.Secret
	message string
}

func (p *PrePushHookOutput) Message() string {
	msg := `================ AXILOCK PUSH PROTECTION ================
    Commits you tried to push had secrets in them.
    Please remove the secrets and try to push again.

    Following secrets were found:`
	for _, secret := range p.Secrets {
		msg += "\n"
		msg += secret.StringWithPrefix("    ")
		msg += "\n"
	}

	return msg
}

type PrePushHook struct {
	name    string
	home    string
	scanner scanner.SecretScanner
}

func NewPrePushHook(home string, scanner scanner.SecretScanner) PrePushHook {
	hook := PrePushHook{name: "pre-push", home: home, scanner: scanner}
	return hook
}

func (p *PrePushHook) Run(remote, url string) (out PrePushHookOutput, err error) {
	var logger = context.Background().Logger()

	if err := ensureUpdatedPrePushHook(p.home); err != nil {
		logger.Error(err, "Could not validate existing pre push hook")
	}

	logger.Info("Running pre-push hook on " + remote + " " + url)
	bufScanner := bufio.NewScanner(os.Stdin)
	var allCommits []git.Commit // across all branches being pushed
	var allSecrets []scanner.Secret
	for bufScanner.Scan() {
		var since, branch string
		line := bufScanner.Text()
		fields := strings.Fields(line)
		logger.V(1).Info(line)

		if len(fields) < 4 {
			continue
		}

		localRef, localOID, _, remoteOID := fields[0], fields[1], fields[2], fields[3]

		branch = localRef

		if git.IsZeroHash(localOID) {
			// branch delete
			continue
		}

		// Update to existing branch, examine commits after current remote pointer
		if !git.IsZeroHash(remoteOID) {
			// since = remoteOID //FIXME: This could be wrong, we need to find the common ancestor
			// That's why we reuse LastPushedCommitReachableByBranch
			since, _ = git.LastPushedCommitReachableByBranch(branch)
			logger.V(1).Info("Last pushed commit reachable by branch is: " + since)
		} else {
			// new branch
			// noFIXME: Possibly taking first commit on branch should do as other
			// branches will be pushed seperately
			// Not really, it is possible only one branch is being pushed and other parent branches are not.
			// in this case, the current branch will have all commits of local parent branches as well
			// and needs to be scanned
			since, _ = git.LastPushedCommitReachableByBranch(branch)
			logger.V(1).Info("Last pushed commit reachable by branch is: " + since)
		}

		commits := git.GetCommitsList(since, branch)

		if err := bufScanner.Err(); err != nil {
			return PrePushHookOutput{Commits: commits}, err
		}

		dir := os.Getenv("GIT_DIR")
		secrets, err := p.scanner.Run(dir, since, branch)
		if err != nil {
			logger.Error(err, "Error running scanner")
		}

		allCommits = append(allCommits, commits...)
		allSecrets = append(allSecrets, secrets...)
	}

	return PrePushHookOutput{Commits: allCommits, Secrets: allSecrets}, nil
}

func ensureUpdatedPrePushHook(home string) error {
	return ensureUpdatedAxiHook(home, "pre-push")
}

func createOrUpdatePrePushHook(home, localHooksDir string) error {
	return createOrUpdateAxiHook(home, "pre-push", localHooksDir)
}
