package fetcher

import (
	"errors"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/axilock/axi/internal/context"
	pb "github.com/axilock/axilock-protos/client"
	"google.golang.org/grpc"
)

type GRPCFetcher struct {
	Version     string
	Interval    time.Duration
	Updated     bool
	Conn        *grpc.ClientConn
	Environment string
}

func (s *GRPCFetcher) Init() error {
	if s.Version == "" {
		return errors.New("version must be specified")
	}

	if s.Interval == 0 {
		s.Interval = 60 * time.Second
	}

	return nil
}

func (s *GRPCFetcher) Fetch() (io.Reader, error) {
	var logger = context.Background().Logger().WithName("AutoUpdate")

	if s.Updated {
		select {} // only one update per invocation
	}

	time.Sleep(s.Interval)

	client := pb.NewHealthServiceClient(s.Conn)
	request := pb.ClientUpdateRequest{ClientVer: s.Version}

	switch runtime.GOOS {
	case "darwin":
		request.Os = pb.OS_OS_DARWIN
	case "linux":
		request.Os = pb.OS_OS_LINUX
	case "windows":
		request.Os = pb.OS_OS_WINDOWS
	default:
		request.Os = pb.OS_OS_UNSPECIFIED
	}

	switch runtime.GOARCH {
	case "amd64":
		request.Arch = pb.ARCH_ARCH_AMD64
	case "arm64":
		request.Arch = pb.ARCH_ARCH_ARM64
	default:
		request.Arch = pb.ARCH_ARCH_UNSPECIFIED
	}

	switch s.Environment {
	case "dev":
		request.Environment = pb.Env_ENV_DEV
	case "release":
		request.Environment = pb.Env_ENV_RELEASE
	default:
		request.Environment = pb.Env_ENV_UNSPECIFIED
	}

	ctx, cancel := context.WithGrpcTimeout(context.Background())
	defer cancel()
	response, err := client.ClientUpdateRpc(ctx, &request)
	if err != nil {
		return nil, err
	}

	if response.ToUpdate {
		logger.Info("Downloading update from " + response.GetClientUpdatePath())
		resp, err := http.Get(response.GetClientUpdatePath())
		if err != nil {
			return nil, err
		}
		// do not defer resp.Close, it is handled by
		// https://github.com/jpillora/overseer/blob/master/proc_master.go#L219

		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to download update")
		}

		logger.Info("Update downloaded, size: " + strconv.FormatInt(resp.ContentLength, 10) + ". Refreshing binary")
		s.Updated = true
		return resp.Body, nil
	} else {
		logger.Info("Already on latest version")
	}

	s.Updated = true // at most one successful update check per cli invocation
	logger.Info("Update checked once. Will not check again")
	return nil, nil
}
