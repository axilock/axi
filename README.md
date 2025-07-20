# Axi - Git Pre Push Secret Prevention

[![release](https://github.com/axilock/axi/actions/workflows/release.yaml/badge.svg)](https://github.com/axilock/axi/actions/workflows/release.yaml) [![beta](https://github.com/axilock/axi/actions/workflows/beta.yaml/badge.svg)](https://github.com/axilock/axi/actions/workflows/beta.yaml)

Axi is a powerful Git hooks management tool that can:
1. Scan commits for potential secrets using [TruffleHog](https://github.com/trufflesecurity/trufflehog)
2. Block pushes when secrets are detected
3. Report findings to a centralized backend for metrics, monitoring and coverage
4. Integrate with Sentry for reliable error tracking
5. Supports YAML-based configuration for easy customization

## Requirements

- Go 1.24 or higher
- Git
- [TruffleHog](https://github.com/trufflesecurity/trufflehog) (for secret scanning) : pulled during install

## Quickstart

```bash
curl -sL https://get.axilock.ai | sh
```

## Building from Source

1. Clone the repository:
```bash
git clone https://github.com/axilock/axi.git
cd axi
```

2. Edit ``config.mk`` for default options (some of these can be overridden by ``config.yaml``)

3. Build and Install axi
```bash
make
bin/axi install
```

Debug builds can be created by setting the DEBUG flag:
```bash
make dev DEBUG=true
```

Read the [documentation](https://docs.axilock.ai/secret-prevention/self-host/cli/) for more options.

## Usage

Axi automatically integrates with your Git workflow once installed. It primarily operates through the pre-push hook to scan commits for secrets before they are pushed to remote repositories.

### Configuration

Configuration can be specified in `~/.axi/config.yaml` or `~/.axi/config.yml`:

Default configuration:
```yaml
verbose: false                                   # Enable verbose output
autoupdate: on                                   # Enable auto-updates. Use on, off or notify
sentry: true                                     # Enable error tracking
debug: false                                     # Enable debug logging, disable autoupdate and Sentry
environment: release                             # Environment name: dev or release, depending on ``make dev`` or ``make``
grpc_server_name: grpc.axilock.ai                # Insights backend grpc server name (not url)
grpc_port: 443                                   # Insights backend grpc server port
grpc_tls: true                                   # Are you using tls at backend grpc ?
sentry_dsn: https://<key>@sentry.io/<project_id>
sentry_log_levels_to_capture:                    # Only these log levels will be captured at sentry
- error
- fatal
verbose: false                                   # Enable debug logging
frontend_url: https://app.axilock.ai/            # Insights frontend http/s url
offline: false                                   # Run completey offline, send no metrics whatsoever
```


## License

Copyright (c) Axilock. All rights reserved.  
SPDX-License-Identifier: Apache-2.0

Trufflehog is downloaded at runtime and is licesed under [AGPL 3.0](https://github.com/trufflesecurity/trufflehog?tab=AGPL-3.0-1-ov-file)
