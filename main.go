package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/axilock/axi/hooks"
	"github.com/axilock/axi/installer"
	"github.com/axilock/axi/internal/auth"
	"github.com/axilock/axi/internal/config"
	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/fetcher"
	"github.com/axilock/axi/internal/filesio"
	"github.com/axilock/axi/internal/log"
	"github.com/axilock/axi/scanner"
	"github.com/fatih/color"
	"github.com/jpillora/overseer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CLI struct {
	Auth      AuthCmd      `cmd:"" help:"Authentication"`
	Version   VersionCmd   `cmd:"" help:"Show version"`
	Install   InstallCmd   `cmd:"" help:"Install axi"`
	Uninstall UninstallCmd `cmd:"" help:"Uninstall axi"`
	Reinstall ReInstallCmd `cmd:"" help:"Reinstall axi"`

	Hook HookCmd `cmd:"" help:"Trigger axi-built hook"`

	Sleep        SleepCmd       `cmd:"" help:"Sleep"`
	CheckUpdates UpdateCheckCmd `cmd:"" help:"Check for updates"`
}

type (
	IsAuth bool
)

func main() {
	prog := func() int { return 0 }

	logger, sync := log.New("axi")
	defer sync()

	defer func() {
		if r := recover(); r != nil {
			code := NewRetCode(NewRuntimePanic(r))
			os.Exit(int(code))
		}
	}()

	cfg := config.NewConfig().WithRuntimeFlags()

	modeMessage := func(w io.Writer, a ...any) {} // noop

	if cfg.Environment == config.Dev {
		modeMessage = color.New(color.FgBlue).FprintlnFunc()
		modeMessage(os.Stderr, "[+] Developer build. Version: "+cfg.Version)
	}

	if cfg.Debug {
		modeMessage = color.New(color.FgBlue).FprintlnFunc()
		modeMessage(os.Stderr, "[!] Debug mode. Some flags will be overridden")
		cfg = cfg.WithDebugFlags()
	}

	if cfg.Offline {
		modeMessage = color.New(color.FgBlue).FprintlnFunc()
		modeMessage(os.Stderr, "[!] Offline mode. Some flags will be overridden")
		cfg = cfg.WithOfflineFlags()
	}

	if cfg.Verbose {
		modeMessage(os.Stderr, "[+] Verbose mode")
		log.SetLevel(10)
		logger.AddConsoleSink(os.Stderr)
	}

	if cfg.Autoupdate != "on" {
		modeMessage(os.Stderr, "[-] Auto update disabled")
	}

	logger.V(1).Info("All flags set")

	if cfg.SentryDsn != "" {
		if err := logger.AddSentrySink(log.SentryConfig{
			SentryDsn:                cfg.SentryDsn,
			Debug:                    cfg.Debug,
			Version:                  cfg.Version,
			Environment:              string(cfg.Environment),
			SentryLogLevelsToCapture: cfg.SentryLogLevelsToCapture,
		}); err != nil {
			// TODO: Couldn't initialize sentry. Send stats to backend
		}
	} else {
		modeMessage(os.Stderr, "[-] Sentry disabled")
	}

	context.SetDefaultLogger(logger.Logger)

	logger.V(1).Info("Config loaded: \n" + cfg.AsYaml())

	afs := filesio.AxiFS{Home: cfg.Home()}
	key, err := afs.APIKey()
	if err != nil {
		logger.Info("Could not get api key (unauth? install?)")
	}

	grpcConn, err := cfg.FreshGRPCConn(
		grpc.WithPerRPCCredentials(
			&auth.APIKeyCredentials{APIKey: key},
		),
	)
	if err != nil {
		logger.Error(err, "Could not establish connection to axi backend")
	}
	defer grpcConn.Close()

	isAuth := IsAuth(key != "") // FIXME: unused

	executable := filepath.Base(os.Args[0])
	logger.Info("Called as " + executable)
	logger.Info(" Args: " + strings.Join(os.Args, " "))

	switch {
	case strings.HasPrefix(executable, filesio.AxiBinaryName): // CLI Invocation
		var cli CLI
		ret := 0
		ctx := kong.Parse(&cli, kong.Vars{
			"version": cfg.Version,
		},
			kong.Bind(grpcConn, logger.AsLogr(), &cfg, isAuth),
		)
		prog = func() int {
			if err := ctx.Run(&ret); err != nil {
				ret = NewRetCode(err) // overrides ret from kong bind
			}
			return ret
		}
	case executable == "reference-transaction": // too noisy
		cfg.Autoupdate = "off"

	default: // Catchall invocation
		prog = func() int {
			if err := hooks.Catchall(grpcConn, &cfg, cfg.Home(), executable, cfg.Version, os.Args[1:]...); err != nil {
				if h, ok := err.(*hooks.HookError); ok {
					logger.Error(h.CausedBy, "Hook failed")
					return NewRetCode(err)
				}
				logger.Error(err, "Hook failed")
				return NewRetCode(err)
			}
			return 0
		}
	}

	switch cfg.Autoupdate {
	case "on":
		updateCfg := overseer.Config{
			Program: func(state overseer.State) {
				os.Exit(prog())
			},
			NoRestart: true,
			Debug:     cfg.Verbose,
			Fetcher: &fetcher.GRPCFetcher{
				Version:     cfg.Version,
				Interval:    1 * time.Second,
				Environment: string(cfg.Environment),
				Conn:        grpcConn,
			},
		}
		overseer.Run(updateCfg)
	case "off":
		os.Exit(prog())
	case "notify":
		wait := make(chan int, 1)
		ret := make(chan int, 1)
		go func() {
			logger.V(1).Info("Checking for updates")
			response, _ := fetcher.UpdateRequest(grpcConn, cfg.Version, string(cfg.Environment))
			if response.ToUpdate {
				fmt.Fprintf(os.Stderr,
					"Axilock update available. %s => %s\n",
					cfg.Version, response.LatestClientver)
				fmt.Fprintf(os.Stderr,
					"Tip: you can enable autoupdate by setting "+
						"``autoupdate: true`` in ~/.axi/config.yaml\n")
			}
			wait <- 1
		}()
		go func() {
			ret <- prog()
		}()
		<-wait
		os.Exit(<-ret)
	}
}

// NewRetCode gets return code according to error type.
// now i think of it, it will always be 0,
// since all non secret response codes should
// allow git to proceed
func NewRetCode(err error) int {
	logger := context.Background().Logger()

	var unsupportedConfiguration *hooks.ErrUnsupportedConfiguration
	var unsupportedInstallationConfiguration *installer.ErrUnsupportedConfiguration
	var unsupportedHook *hooks.ErrUnsupportedHook
	var trufflehogError *scanner.ErrTrufflehogNotInstalled
	var corruptedHook *hooks.ErrCorruptedHook
	var hookError *hooks.HookError

	// TODO: In case of non-secret errors, suggest running doctor
	if errors.As(err, &corruptedHook) ||
		errors.As(err, &unsupportedConfiguration) ||
		errors.As(err, &trufflehogError) ||
		errors.As(err, &unsupportedInstallationConfiguration) ||
		errors.As(err, &unsupportedHook) {
		logger.Error(err, "Irrecoverable error.")
		fmt.Println(err.Error())
		return 0
	}

	// hook errors are caused by axi pre-push run after install or some user hook
	if errors.As(err, &hookError) {
		return err.(*hooks.HookError).ExitCode
	}

	log := true // some errors don't need to be logged (at sentry or cli) like unauth

	// grpc errors
	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.Unauthenticated:
			fmt.Println("Unauthenticated. Please run `~/.axi/bin/axi auth` to authenticate.")
			log = false
		}
	}

	if !errors.Is(err, nil) && log {
		// ensures an error log printed and sent to sentry
		logger.Error(err, err.Error())
	}

	// installation errors are fatal
	var installationFailed *ErrInstallationFailed
	if errors.As(err, &installationFailed) {
		fmt.Println("Installation failed. Please contact support@axilock.ai for help.")
		return 1
	}

	return 0
}
