# Axi - Git Hooks Security Manager

Axi is a powerful Git hooks management tool that helps secure your development workflow by automatically scanning for secrets and sensitive information in your commits. It integrates seamlessly with your Git workflow and provides real-time security checks before code is pushed to remote repositories.

## Features

- 🔒 **Automated Secret Detection**: Integrates with TruffleHog to scan commits for potential secrets and sensitive information
- 🔄 **Auto-Updates**: Built-in auto-update mechanism to keep the tool current
- 🪝 **Git Hooks Management**: Manages Git hooks with focus on pre-push security checks
- 📊 **Remote Monitoring**: GRPC-based backend communication for centralized monitoring
- 🎯 **Error Tracking**: Built-in Sentry integration for reliable error tracking
- ⚙️ **Flexible Configuration**: YAML-based configuration for easy customization

## Requirements

- Go 1.x or higher
- Git
- TruffleHog (for secret scanning)

## Installation

1. Obtain an API key from your Axi backend administrator / Or login to https://app.axilock.ai/ and get api token from `local-storage`
2. Download AXI Binary from url: https://s3.ap-south-1.amazonaws.com/sekrit-releases/dev/v0.0.9-3-gee375b0/darwin/amd64/axi
2. Install Axi using the following command:
```bash
./axi install --api-key YOUR_API_KEY
```

To reinstall or update your installation:
```bash
./axi reinstall
```

## Building from Source

1. Clone the repository:
```bash
git clone https://github.com/axilock/axi.git
cd axi
```

2. Update submodules:
```bash
make protos
```

3. Build the project:

For development:
```bash
make dev
```

For release:
```bash
make release
```

Debug builds can be created by setting the DEBUG flag:
```bash
make dev DEBUG=true
```

## Usage

Axi automatically integrates with your Git workflow once installed. It primarily operates through the pre-push hook to scan commits for secrets before they are pushed to remote repositories.

### Pre-push Hook

The pre-push hook automatically:
1. Scans new commits for potential secrets
2. Reports findings to the configured backend
3. Blocks pushes if secrets are detected

### Configuration

Configuration can be specified in `~/.axi/config.yaml` or `~/.axi/config.yml`:

```yaml
verbose: true    # Enable verbose output
autoupdate: true # Enable auto-updates
sentry: true     # Enable error tracking
```

## Error Codes

- `1`: Secrets found in commits
- `3`: Configuration or setup errors

## Development

### Building and Installing

For development workflow:
```bash
make buildAndInstall
```

### Distribution

Create development distribution:
```bash
make dist-dev
```

Create release distribution:
```bash
make dist-release
```

List available distributions:
```bash
make dist-list
```

## Security

Axi helps prevent accidental commit of secrets by:
- Scanning commits using TruffleHog
- Blocking pushes when secrets are detected
- Reporting findings to a centralized backend for monitoring

## License

[Add your license information here]
