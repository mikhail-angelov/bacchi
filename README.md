# Go Backup Service 🚀

A robust, production-ready backup service written in Go. It performs daily incremental backups of specified folders, encrypts them, and uploads them to S3 (or S3-compatible storage) with an automated retention policy.

## Features

- **Incremental Backups**: Uses GNU `tar` snapshots to minimize storage and bandwidth.
- **S3 Integration**: Works with AWS S3, MinIO, and other S3-compatible providers.
- **Rotation Policy**: Keeps 10 daily and 1 monthly backup automatically.
- **GPG Encryption**: 🔐 Symmetric encryption with a passphrase for secure storage.
- **Telegram Notifications**: 🤖 Get status alerts directly in your Telegram chat.
- **Simple Deployment**: No complex daemon; runs via standard system `cron`.
- **Remote Management**: A powerful `Makefile` for one-command deployment and remote control.

## Prerequisites

- **Go**: 1.25 or higher
- **GNU tar**: Required for incremental backup support
- **GPG**: Required for encryption
- **S3 Bucket**: A bucket where backups will be stored

## Configuration

The service uses a YAML configuration file. See the [template](configs/config.yaml.template) for details.

```yaml
s3:
  bucket: "my-backups"
  region: "us-east-1"
  access_key_id: "..."
  secret_access_key: "..."
  prefix: "server1/"

backups:
  - name: "web-app"
    folders: ["/var/www/html"]
    exclude: ["node_modules"]

encryption:
  enabled: true
  passphrase: "your-secret-key"

telegram:
  enabled: true
  bot_token: "..."
  chat_id: "..."
```

## Usage

### Local Development

```bash
# Build the binary
make build

# Run a backup manually
./backup-service backup --config=config.yaml

# List backups in S3
./backup-service list --config=config.yaml

# Restore a backup
./backup-service restore "server1/web-app_20251228.tar.gz.gpg" ./restored-files
```

### Remote Deployment

1. Set your server details in the `Makefile`:
   ```makefile
   SSH_USER=root
   SSH_HOST=your.server.ip
   ```
2. Deploy everything:
   ```bash
   make deploy
   ```
3. Copy the crontab line printed at the end of the deployment to your server's `crontab -e`.

## Development

```bash
# Run tests
make test

# Run linter (requires golangci-lint)
make lint
```

## CI/CD

GitHub Actions are configured for:
- **CI**: Automated testing and linting on every push.
- **Release**: Automated binary builds and GitHub Releases on tag push (e.g., `v1.0.0`).

## License

MIT
