.PHONY: wire build test frontend up down deploy rollback tags remote-status help

# Local development
wire:
	cd backend && GOTOOLCHAIN=go1.26.0 go run github.com/google/wire/cmd/wire@latest ./cmd/server

build:
	cd backend && GOTOOLCHAIN=go1.26.0 go build -o bin/server ./cmd/server

test:
	cd backend && GOTOOLCHAIN=go1.26.0 go test ./...

frontend:
	cd frontend && npm install && npm run build

up:
	docker compose up --build -d

down:
	docker compose down

# Production (OCI) — override: make deploy OCI_HOST=1.2.3.4
OCI_HOST ?= 144.24.34.65
OCI_USER ?= ubuntu
OCI_SSH_KEY ?= $(HOME)/.ssh/oracle-cloud
OCI_APP_DIR ?= family_tree
REMOTE := OCI_HOST=$(OCI_HOST) OCI_USER=$(OCI_USER) OCI_SSH_KEY=$(OCI_SSH_KEY) OCI_APP_DIR=$(OCI_APP_DIR) ./scripts/remote.sh

deploy:
	$(REMOTE) 'set -euo pipefail; cd ~/$(OCI_APP_DIR); git fetch origin main; git reset --hard origin/main; ./scripts/deploy.sh'

rollback:
	@test -n "$(TAG)" || (echo "Usage: make rollback TAG=<git-sha>  (e.g. make rollback TAG=a11bd98)" && exit 1)
	$(REMOTE) 'set -euo pipefail; cd ~/$(OCI_APP_DIR); git fetch origin main; git checkout $(TAG); IMAGE_TAG=$(TAG) ./scripts/rollback.sh $(TAG)'

tags:
	@echo "Recent commits (use short SHA with: make rollback TAG=<sha>):"
	@git log --oneline -15

remote-status:
	$(REMOTE) 'cd ~/$(OCI_APP_DIR) && git log -1 --oneline && echo && sudo docker compose ps'

help:
	@echo "Local:"
	@echo "  make up          Start dev stack (build from source)"
	@echo "  make down        Stop dev stack"
	@echo "  make test        Run Go tests"
	@echo ""
	@echo "Production (OCI):"
	@echo "  make deploy      Pull latest main + Docker Hub images + restart"
	@echo "  make rollback TAG=<sha>   Deploy a previous image tag"
	@echo "  make tags        Show recent git SHAs for rollback"
	@echo "  make remote-status        Show server git + container status"
	@echo ""
	@echo "Overrides: OCI_HOST OCI_USER OCI_SSH_KEY OCI_APP_DIR"