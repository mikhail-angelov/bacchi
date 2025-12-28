# Go Backup Service 🚀

A robust, production-ready backup service written in Go. It performs smart full/incremental backups of specified folders, encrypts them, and uploads them to S3 (or S3-compatible storage) with an automated retention policy.

## Features

- **Smart Full/Incremental Strategy**: Automatically identifies if a full backup is needed (e.g., at the start of each month) or creates an incremental slice using GNU `tar` snapshots.
- **Automatic Chained Restore**: The tool automatically reconstructs the full state by identifying and applying the base full backup and all subsequent incremental slices in order.
- **S3 Integration**: Works with AWS S3, MinIO, and other S3-compatible providers.
- **Rotation Policy**: Keeps 10 daily and 1 monthly backup automatically.
- **GPG Encryption**: 🔐 Symmetric encryption with a passphrase for secure storage.
- **Telegram Notifications**: 🤖 Get status alerts directly in your Telegram chat (success/failure details).
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

## Makefile Commands

The project includes a robust `Makefile` for both local development and remote server management.

### Remote Server Management
Run these on your local machine to control the remote server (requires `SSH_USER` and `SSH_HOST` in `.env` or `Makefile`):

- `make deploy`: Build specifically for Linux (amd64) and push binary + configs to the server.
- `make backup`: Trigger a remote backup (auto full/incremental).
- `make backup-full`: Force a remote full backup.
- `make list`: List all backups currently in S3.
- `make restore`: Restore the **latest** state for all backup sets on the server.
- `make restore TAG=path/to/backup`: Restore a specific backup chain on the server.
- `make logs`: Stream remote application logs.

### Local Utilities
- `make restore-local`: Downloads and reconstructs the latest state from S3 into a local `./tmp` folder for verification.
- `make lint`: Run `golangci-lint` with production-ready settings.
- `make test`: Run all Go unit tests.

## Usage

### Local Development

```bash
# Build the binary
go build -o backup-service ./cmd/backup-service

# Run a backup manually
./backup-service backup --config=config.yaml

# Restore a backup (this will automatically find the full backup chain)
./backup-service restore "server1/web-app_20251228.inc.tar.gz.gpg" ./target-dir
```

## License

MIT
