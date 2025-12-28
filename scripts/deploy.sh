#!/bin/bash

# Configuration
BINARY_NAME="backup-service"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/backup-service"

# If run with 'remote' argument, skip building and moving binary (the Makefile handles that)
if [ "$1" == "remote" ]; then
    echo "Running remote configuration setup..."
else
    echo "Building $BINARY_NAME..."
    go build -buildvcs=false -o $BINARY_NAME ./cmd/backup-service

    echo "Installing $BINARY_NAME to $INSTALL_DIR..."
    sudo mv $BINARY_NAME $INSTALL_DIR/
fi

echo "Setting up config directory $CONFIG_DIR..."
sudo mkdir -p $CONFIG_DIR
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    sudo cp configs/config.yaml.template $CONFIG_DIR/config.yaml
    echo "Default config created at $CONFIG_DIR/config.yaml. PLEASE EDIT IT."
fi

# We can also sync the template if it's newer
if [ -f "configs/config.yaml.template" ]; then
    sudo cp configs/config.yaml.template $CONFIG_DIR/config.yaml.template
fi

echo "Deployment complete!"
echo ""
echo "To schedule daily backups, add the following to your crontab (sudo crontab -e):"
cat configs/crontab.example || cat $CONFIG_DIR/crontab.example
