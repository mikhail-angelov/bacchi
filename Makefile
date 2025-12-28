include .env

BINARY_NAME=backup-service
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
	scp configs/config.yaml $(SSH_USER)@$(SSH_HOST):$(CONFIG_DIR)/config.yaml
	scp configs/crontab.example $(SSH_USER)@$(SSH_HOST):$(CONFIG_DIR)/
	ssh $(SSH_USER)@$(SSH_HOST) "echo 'Deployment complete!'"
	ssh $(SSH_USER)@$(SSH_HOST) "echo 'To schedule daily backups, add the following to your crontab (sudo crontab -e):'"
	ssh $(SSH_USER)@$(SSH_HOST) "cat $(CONFIG_DIR)/crontab.example"

# Remote commands
backup:
	ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) backup --config=$(CONFIG_DIR)/config.yaml"

backup-full:
	ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) backup --full --config=$(CONFIG_DIR)/config.yaml"

list:
	ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) list --config=$(CONFIG_DIR)/config.yaml"

restore:
	@if [ -n "$(TAG)" ]; then \
		ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) restore $(TAG) / --config=$(CONFIG_DIR)/config.yaml"; \
	else \
		names=$$(grep "name:" configs/config.yaml | sed 's/.*name: //;s/"//g;s/ //g'); \
		for name in $$names; do \
			echo "Finding latest backup for $$name..."; \
			latest=$$(ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) list --config=$(CONFIG_DIR)/config.yaml | grep "/$$name" | sort | tail -n 1"); \
			if [ -z "$$latest" ]; then \
				echo "No backups found for $$name"; \
			else \
				echo "Restoring latest backup for $$name: $$latest"; \
				ssh $(SSH_USER)@$(SSH_HOST) "$(INSTALL_DIR)/$(BINARY_NAME) restore \"$$latest\" / --config=$(CONFIG_DIR)/config.yaml"; \
			fi; \
		done; \
	fi

logs:
	ssh $(SSH_USER)@$(SSH_HOST) "tail -f /var/log/backup-service.log"

restore-local:
	-rm -rf ./tmp
	mkdir -p ./tmp
	@names=$$(grep "name:" configs/config.yaml | sed 's/.*name: //;s/"//g;s/ //g'); \
	for name in $$names; do \
		echo "Finding latest backup for $$name..."; \
		latest=$$(go run ./cmd/backup-service/main.go list --config=configs/config.yaml | grep "/$$name" | sort | tail -n 1); \
		if [ -z "$$latest" ]; then \
			echo "No backups found for $$name"; \
		else \
			echo "Restoring latest backup for $$name: $$latest"; \
			go run ./cmd/backup-service/main.go restore "$$latest" ./tmp --config=configs/config.yaml; \
		fi; \
	done
