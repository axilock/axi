package main

import (
	"fmt"

	"github.com/axilock/axi/installer"
	"github.com/axilock/axi/internal/auth"
	"github.com/axilock/axi/internal/config"
	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/filesio"
	"github.com/axilock/axi/internal/git"
	"github.com/axilock/axi/internal/utils"
	pb "github.com/axilock/axilock-protos/client"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Kong bindings
type InstallCmd struct {
	ApiKey           string `optional:"" help:"API key to connect to axi backend (optional)"`
	SkipDependencies bool   `help:"Skip dependencies" default:"false"`
}

type ReInstallCmd struct {
	SkipDependencies bool `help:"Skip dependencies" default:"false"`
}

type UninstallCmd struct{}

func (c *InstallCmd) Run(cfg *config.Config, logger logr.Logger) error {
	conn, err := cfg.FreshGRPCConn()
	if err != nil {
		return &ErrInstallationFailed{error: err}
	}

	afs := filesio.AxiFS{Home: cfg.Home()}
	if err := afs.Create(); err != nil {
		return &ErrInstallationFailed{error: err}
	}

	errDeps := make(chan error, 1)
	if !c.SkipDependencies {
		fmt.Println("Downloading dependencies: trufflehog")
		go func() {
			if err := installer.InstallTrufflehog(cfg.Home()); err != nil {
				logger.Error(err, "Install dependencies failed")
				errDeps <- &ErrInstallationFailed{error: err}
			}
			errDeps <- nil
		}()
	}

	if !cfg.Offline {
		if c.ApiKey == "" {
			key, err := auth.Login(conn, cfg.BackendUrl)
			if err != nil {
				return &ErrInstallationFailed{error: err}
			}
			c.ApiKey = key
		}

		reqMetadata := utils.GetSystemMetadataJson(cfg.Version)
		client := pb.NewMetadataServiceClient(conn)
		ctx, cancel := context.GRPCContext()
		ctx = context.WithAuth(ctx, c.ApiKey)
		defer cancel()
		_, err = client.InitMetadata(ctx, &pb.InstallerInitRequest{
			Status:   pb.InstallerInitRequest_STATE_INIT,
			Metadata: reqMetadata,
		})
		if err != nil {
			if grpcErr, ok := status.FromError(err); ok && grpcErr.Code() == codes.Unauthenticated {
				logger.Error(err, "Unauthenticated: invalid api key")
				fmt.Println("Unauthenticated")
			} else {
				logger.Error(err, "Could not sync server for installation")
			}
			return &ErrInstallationFailed{error: err}
		}
	}

	if err := installer.Install(cfg.Home(), c.ApiKey); err != nil {
		logger.Error(err, "Install failed")
		return &ErrInstallationFailed{error: err}
	}

	if !cfg.Offline {
		client := pb.NewMetadataServiceClient(conn)
		ctx, cancel := context.GRPCContext()
		ctx = context.WithAuth(ctx, c.ApiKey)
		defer cancel()
		_, err = client.InitMetadata(ctx, &pb.InstallerInitRequest{
			Status:   pb.InstallerInitRequest_STATE_DONE,
			Metadata: "{}",
		})
		if err != nil {
			logger.Info("Could not send installation done metadata")
		}
	}

	if err = <-errDeps; err != nil {
		return err
	}
	return nil
}

func (c *ReInstallCmd) Run(cli *CLI, cfg *config.Config, logger logr.Logger, conn *grpc.ClientConn) error {
	afs := filesio.AxiFS{Home: cfg.Home()}
	key, err := afs.APIKey()
	if err != nil {
		logger.Error(err, "Could not get api key. Please perform a fresh installation")
	}

	logger.Info("Using api key", "key", key)
	installCmd := InstallCmd{ApiKey: key, SkipDependencies: c.SkipDependencies}
	return installCmd.Run(cfg, logger)
}

func (r *UninstallCmd) Run(cfg *config.Config, conn *grpc.ClientConn) error {
	// TODO: Send logs to backend about uninstall
	if err := git.UnsetGlobalCoreHooksPath(); err != nil {
		return err
	}

	afs := filesio.AxiFS{Home: cfg.Home()}
	if err := afs.Delete(); err != nil {
		return err
	}
	fmt.Println("Uninstall successfull")
	return nil
}

type ErrInstallationFailed struct {
	error
}

func (e *ErrInstallationFailed) Error() string {
	return "Installation failed: " + e.error.Error()
}
