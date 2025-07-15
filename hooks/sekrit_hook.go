package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/filesio"
)

type AxiShellScript struct {
	Header string
	Body   string
	Footer string
}

const (
	axiHeader = "#!/bin/sh"
	axiFooter = "# AXILOCK WARNING MESSAGE" +
		"\n# DO NOT EDIT" +
		"\n# IF YOU NEED PRE-PUSH HOOK, WRITE IT IN A NEW pre-push.user FILE" +
		"\n"
)

func NewAxiShellScript(body string) *AxiShellScript {
	return &AxiShellScript{
		Header: axiHeader,
		Body:   body,
		Footer: axiFooter,
	}
}

func (as *AxiShellScript) String() string {
	return as.Header + "\n" + as.Body + "\n" + as.Footer
}

// string match
func (as *AxiShellScript) Match(str string) bool {
	return as.String() == str
}

// Match only header and footer
func (as *AxiShellScript) RoughMatch(script string) bool {
	return strings.HasPrefix(script, as.Header) && strings.HasSuffix(script, as.Footer)
}

func IsAnyAxiScript(script string) bool {
	dummy := NewAxiShellScript("")
	if dummy.RoughMatch(script) {
		return true
	}

	return IsOldAxiScript(script)
}

func IsOldAxiScript(script string) bool {
	oldScripts := []AxiShellScript{
		// add newer scripts' header/footer here
		{
			Header: "#!/bin/sh",
			Footer: "# SEKRIT WARNING MESSAGE" +
				"\n# DO NOT EDIT" +
				"\n# IF YOU NEED PRE-PUSH HOOK, WRITE IT IN A NEW pre-push.user FILE" +
				"\n",
		},
	}

	for _, as := range oldScripts {
		if as.RoughMatch(script) {
			return true
		}
	}
	return false
}

func ensureUpdatedAxiHook(home, name string) error {
	localHooksDir, err := getLocalHooksDir()
	if err != nil {
		return err
	}
	return createOrUpdateAxiHook(home, name, localHooksDir)
}

// createOrUpdateAxiHook creates or updates the hook name given.
// To update the hook, the hook must be owned by axi,
// this is done by checking if headers and footers match.
// If they match, then hook is updated (if required)
// as per 'script' parameter.
// In order to update header/footer itself, add old ones in
// IsOldAxiScript and modify NewAxiShellScript struct
func createOrUpdateAxiHook(home, name, hooksDir string) error {
	var logger = context.Background().Logger()

	afs := filesio.AxiFS{Home: home}
	//FIXME: .git/hooks/pre-push.user will not work for submodules
	script := NewAxiShellScript(fmt.Sprintf(`
AXI="%s"
HOOK_NAME=$(basename "$0")
HOOK="$0"
USER_HOOK=".git/hooks/$HOOK_NAME.user"
INPUT=$(cat)
if [ -f "$AXI" ]; then
    printf "%%s" "$INPUT" | "$AXI" hook "$HOOK_NAME" "$@"
    RET=$?
    if [ $RET -ne 0 ]; then
        exit $RET
    fi
elif [ -f "$USER_HOOK" ]; then
	git config --local --unset core.hooksPath
	mv "$USER_HOOK" "$HOOK"
else
	git config --local --unset core.hooksPath
	rm "$HOOK"
fi

# user defined hook
if [ -f "$USER_HOOK" ]; then
	printf "%%s" "$INPUT" | "$USER_HOOK" "$@"
    exit $?
fi`, afs.BinaryPath()))

	path := filepath.Join(hooksDir, name)
	// Happy flow: no user defined hook exists
	if exists := filesio.FileExists(path); !exists {
		if err := filesio.WriteExecutableFileWithContent(path, script.String()); err != nil {
			return err
		}
		return nil
	}
	// Midly unhappy flow: user had prior hook set
	existingb, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	existing := string(existingb)

	if script.Match(existing) {
		return nil
	}

	if IsAnyAxiScript(existing) {
		logger.Info("Old axi script found. Replacing...")
		if err := filesio.WriteExecutableFileWithContent(path, script.String()); err != nil {
			return err
		}
		return nil
	}

	// Unhappy flow: user's hook is not axi owned
	if err := MatchFile(path, script.String()); err != nil {
		err = &ErrCorruptedHook{Name: name, Path: path, MatchError: err}
		return err
	}

	return nil
}

/*
type AxiHook struct {
	Name string
	Home string
}

type AvailableAxiHook string

const (
	PrePush AvailableAxiHook = "pre-push"
)

func (s *AxiHook) preHook() error {
	//TODO: healthcheck for install/uninstall
	return nil
}

func (s *AxiHook) Run(args ...string) error {
	return nil
}

func (s *AxiHook) postHook() error {
	return nil
}
*/
