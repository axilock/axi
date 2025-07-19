package config

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsentry/sentry-go"
	"github.com/goccy/go-yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// set via go build -X github.com/axilock/axi/internal/config.FLAG=value
var Version string
var debug string
var env string
var autoupdate string
var grpcServerName string
var grpcPort string
var sentryDsn string
var grpcTls string
var backendUrl string
var verbose string
var offline string

type Env string

const (
	Dev Env = "dev"
	Rel Env = "release"
)

type Config struct {
	AxiHomeDirName           string
	Debug                    bool
	Autoupdate               string
	Version                  string
	Environment              Env
	GRPCServerName           string
	GRPCPort                 string
	GRPCTLs                  bool
	SentryDsn                string
	SentryLogLevelsToCapture []sentry.Level
	Verbose                  bool
	BackendUrl               string
	Offline                  bool
	home                     string
}

type ConfigP struct { // for yaml unmarshalling
	AxiHomeDirName           *string         `yaml:"-"`
	Debug                    *bool           `yaml:"debug"`
	Autoupdate               *string         `yaml:"autoupdate"`
	Version                  *string         `yaml:"version"`
	Environment              *Env            `yaml:"environment"`
	GRPCServerName           *string         `yaml:"grpc_server_name"`
	GRPCPort                 *string         `yaml:"grpc_port"`
	GRPCTLs                  *bool           `yaml:"grpc_tls"`
	SentryDsn                *string         `yaml:"sentry_dsn"`
	SentryLogLevelsToCapture *[]sentry.Level `yaml:"sentry_log_levels_to_capture"`
	Verbose                  *bool           `yaml:"verbose"`
	BackendUrl               *string         `yaml:"backend_url"`
	Offline                  *bool           `yaml:"offline"`
}

func NewConfig() Config {
	return Config{
		AxiHomeDirName:           ".axi",
		Debug:                    debug == "true",
		Autoupdate:               autoupdate,
		Version:                  Version,
		Environment:              Env(env),
		GRPCServerName:           grpcServerName,
		GRPCPort:                 grpcPort,
		GRPCTLs:                  grpcTls == "true",
		SentryDsn:                sentryDsn,
		BackendUrl:               backendUrl,
		SentryLogLevelsToCapture: []sentry.Level{"error", "fatal"},
		Verbose:                  verbose == "true",
		Offline:                  offline == "true",
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

		configYaml := ConfigP{}
		err = yaml.Unmarshal(data, &configYaml)
		if err != nil {
			continue
		}

		if configYaml.Debug != nil {
			c.Debug = *configYaml.Debug
		}
		if configYaml.Autoupdate != nil {
			c.Autoupdate = *configYaml.Autoupdate
		}
		if configYaml.Environment != nil {
			c.Environment = *configYaml.Environment
		}
		if configYaml.GRPCServerName != nil {
			c.GRPCServerName = *configYaml.GRPCServerName
		}
		if configYaml.GRPCPort != nil {
			c.GRPCPort = *configYaml.GRPCPort
		}
		if configYaml.GRPCTLs != nil {
			c.GRPCTLs = *configYaml.GRPCTLs
		}
		if configYaml.SentryDsn != nil {
			c.SentryDsn = *configYaml.SentryDsn
		}
		if configYaml.SentryLogLevelsToCapture != nil {
			c.SentryLogLevelsToCapture = *configYaml.SentryLogLevelsToCapture
		}
		if configYaml.Verbose != nil {
			c.Verbose = *configYaml.Verbose
		}
		if configYaml.BackendUrl != nil {
			c.BackendUrl = *configYaml.BackendUrl
		}
		if configYaml.Offline != nil {
			c.Offline = *configYaml.Offline
		}

		break
	}

	return c
}

func (c Config) WithDebugFlags() Config {
	c.Autoupdate = "off"
	c.SentryDsn = ""
	c.Verbose = true
	return c
}

func (c Config) WithOfflineFlags() Config {
	c.Autoupdate = "off"
	c.SentryDsn = ""
	return c
}

func (c *Config) AsYaml() string {
	yaml, _ := yaml.Marshal(c)
	return string(yaml)
}

func (c *Config) Home() string {
	if c.home != "" {
		return c.home
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Unable to get user's home dir. Falling back to tmp directory.")
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
