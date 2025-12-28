BINARY_NAME=backup-service
SSH_USER=root
SSH_HOST=your-server-ip
INSTALL_DIR=/usr/local/bin
CONFIG_DIR=/etc/backup-service

build:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME) ./cmd/backup-service

clean:
	rm -f $(BINARY_NAME)

test:
	go test -v ./...

lint:
	@echo "Linting code..."
	@$(shell go env GOPATH)/bin/golangci-lint run

deploy: build
	ssh $(SSH_USER)@$(SSH_HOST) "mkdir -p $(CONFIG_DIR)"
	scp $(BINARY_NAME) $(SSH_USER)@$(SSH_HOST):$(INSTALL_DIR)/
	scp configs/config.yaml.template $(SSH_USER)@$(SSH_HOST):$(CONFIG_DIR)/config.yaml.template
	scp configs/crontab.example $(SSH_USER)@$(SSH_HOST):$(CONFIG_DIR)/
	scp scripts/deploy.sh $(SSH_USER)@$(SSH_HOST):/tmp/deploy.sh
	ssh $(SSH_USER)@$(SSH_HOST) "chmod +x /tmp/deploy.sh && /tmp/deploy.sh remote"

# Remote commands
backup:
	ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) backup --config=$(CONFIG_DIR)/config.yaml"

list:
	ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) list --config=$(CONFIG_DIR)/config.yaml"

logs:
	ssh $(SSH_USER)@$(SSH_HOST) "tail -f /var/log/backup-service.log"
