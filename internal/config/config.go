package config

import (
	"crypto/tls"
	"crypto/x509"
	"os"
	"path/filepath"

	"github.com/axilock/axi/internal/context"
	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// set via go build -X internal/config.FLAG
var version string
var debug string
var env string // dev or release
var autoupdate string
var grpcServerName string
var grpcPort string
var sentryDsn string
var grpcTls string
var backendUrl string

type Env string

const (
	Dev Env = "dev"
	Rel Env = "release"
)

type Config struct {
	AxiHomeDirName           string         `yaml:"axi_dir_name"`
	Debug                    bool           `yaml:"debug"`
	Autoupdate               bool           `yaml:"autoupdate"`
	Version                  string         `yaml:"version"`
	Environment              Env            `yaml:"environment"`
	GRPCServerName           string         `yaml:"grpc_server_name"`
	GRPCPort                 string         `yaml:"grpc_port"`
	GRPCTLs                  bool           `yaml:"grpc_tls"`
	SentryDsn                string         `yaml:"sentry_dsn"`
	AxiBinary                string         `yaml:"axi_binary"`
	SentryLogLevelsToCapture []sentry.Level `yaml:"sentry_log_levels_to_capture"`
	Verbose                  bool           `yaml:"verbose"`
	BackendUrl               string         `yaml:"backend_url"`
	home                     string         `yaml:"-"` // internal
}

func NewConfig() *Config {
	return &Config{
		AxiHomeDirName:           ".axi",
		Debug:                    debug == "true",
		Autoupdate:               autoupdate == "true",
		Version:                  version,
		Environment:              Env(env),
		GRPCServerName:           grpcServerName,
		GRPCPort:                 grpcPort,
		GRPCTLs:                  grpcTls == "true",
		SentryDsn:                sentryDsn,
		BackendUrl:               backendUrl,
		SentryLogLevelsToCapture: []sentry.Level{"error", "fatal"},
	}
}

func (c Config) WithRuntimeFlags() Config {
	return c.WithRuntimeYAML()
}

func (c Config) WithRuntimeYAML() Config {
	configFiles := []string{
		filepath.Join(c.Home(), "config.yaml"),
		filepath.Join(c.Home(), "config.yml"),
	}

	for _, configFile := range configFiles {
		data, err := os.ReadFile(configFile)
		if err != nil {
			continue
		}

		err = yaml.Unmarshal(data, c)
		if err != nil {
			continue
		}

		break
	}

	return c
}

func (c Config) WithDebugFlags() Config {
	c.Autoupdate = false
	c.SentryDsn = ""
	c.Verbose = true
	return c
}

func (c *Config) Home() string {
	var logger = context.Background().Logger()

	if c.home != "" {
		return c.home
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		logger.Error(err, "Unable to get user's home dir. Falling back to tmp directory.")
		homeDir = os.TempDir()
	}

	c.home = filepath.Join(homeDir, c.AxiHomeDirName)
	return c.home
}

func (c *Config) TrufflehogPath() string {
	return filepath.Join(c.Home(), "bin", "trufflehog")
}

func (c *Config) GRPCEndpoint() string {
	return c.GRPCServerName + ":" + c.GRPCPort
}

// Create a new grpc connection to backend, unauthenticated
func (c *Config) FreshGRPCConn(grpcOpts ...grpc.DialOption) (*grpc.ClientConn, error) {
	creds := insecure.NewCredentials()
	if c.GRPCTLs {
		systemRoots, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		tlsConfig := &tls.Config{
			ServerName: c.GRPCServerName,
			RootCAs:    systemRoots,
			MinVersion: tls.VersionTLS12,
		}
		creds = credentials.NewTLS(tlsConfig)
	}

	grpcOpts = append(grpcOpts, grpc.WithTransportCredentials(creds))
	return grpc.NewClient(c.GRPCEndpoint(), grpcOpts...)
}
