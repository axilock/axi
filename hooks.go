package main

import (
	"fmt"
	"os"

	"github.com/axilock/axi/hooks"
	"github.com/axilock/axi/internal/config"
	"github.com/axilock/axi/internal/context"
	"github.com/axilock/axi/internal/git"
	"github.com/axilock/axi/scanner"
	pb "github.com/axilock/axilock-protos/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Kong bindings
type HookCmd struct {
	Name string   `arg:"" help:"Name of hook invoked" enum:"pre-push"`
	Args []string `arg:"" help:"Arguments to this hook"`
}

func (h *HookCmd) Run(
	cli *CLI,
	cfg *config.Config,
	conn *grpc.ClientConn,
	ret *int,
) error {
	logger := context.Background().Logger()

	switch h.Name {
	case "pre-push":
		if len(h.Args) != 2 {
			return fmt.Errorf("pre-push hook requires 2 arguments")
		}
		hook := hooks.NewPrePushHook(
			cfg.Home(),
			scanner.NewTrufflehog(cfg.TrufflehogPath()),
		)
		repo := h.Args[1]
		out, err := hook.Run(h.Args[0], repo)
		if err != nil {
			return err
		}

		if len(out.Commits) > 0 {
			if !cfg.Offline {
				err := sendCommitData(conn, repo, out.Commits)
				if err != nil {
					logger.Error(err, err.Error())
				}
			}
		} else {
			logger.V(1).Info("No commits to send")
		}

		if len(out.Secrets) > 0 {
			*ret = 1
			if !cfg.Offline {
				err := sendSecretAlerts(conn, repo, out.Secrets)
				if err != nil {
					logger.Error(err, err.Error())
				}
			}
			fmt.Fprint(os.Stderr, out.Message())
		} else {
			logger.V(1).Info("No secret alerts to send")
		}

		return nil
	}
	return &hooks.ErrUnsupportedHook{Name: string(h.Name)}
}

func sendSecretAlerts(conn *grpc.ClientConn, repo string, secrets []scanner.Secret) error {
	logger := context.Background().Logger()

	client := pb.NewAlertServiceClient(conn)
	errc := make(chan error, 1)

	go func() {
		for _, secret := range secrets {
			request := pb.SecretAlertRequest{
				FileName:   secret.File,
				Repo:       repo,
				LineNumber: int64(secret.Line),
				CommitId:   secret.Commit.ID,
				SecretType: secret.Type,
				// FIXME: >_<
				IsVerified: false,
				Fragment:   "",
			}
			ctx, cancel := context.GRPCContext()
			defer cancel()
			if _, err := client.SecretAlert(ctx, &request); err != nil {
				errc <- err
				return
			}
		}
		errc <- nil
	}()

	logger.Info(fmt.Sprintf("Alert sent for %d secrets", len(secrets)))
	return <-errc
}

func sendCommitData(conn *grpc.ClientConn, repo string, commits []git.Commit) error {
	logger := context.Background().Logger()

	client := pb.NewCommitDataServiceClient(conn)

	var commitspb []*pb.SendCommitDataRequest_CommitObjects
	for _, commit := range commits {
		commitspb = append(commitspb, &pb.SendCommitDataRequest_CommitObjects{
			CommitId:     commit.ID,
			CommitAuthor: commit.Author,
			CommitTime:   timestamppb.New(commit.Time),
		})
	}

	logger.Info(fmt.Sprintf("Synced %d commits", len(commits)))
	logger.V(1).Info(fmt.Sprintf("Commits: %v", commitspb))

	request := pb.SendCommitDataRequest{
		Commits:  commitspb,
		RepoUrl:  repo,
		PushTime: timestamppb.Now(),
	}

	ctx, cancel := context.GRPCContext()
	defer cancel()

	_, err := client.SendCommitData(ctx, &request)

	st, ok := status.FromError(err)
	if ok {
		switch st.Code() {
		case codes.AlreadyExists:
			return nil
		}
	}
	return err
}
