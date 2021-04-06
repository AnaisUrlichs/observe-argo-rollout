include .bingo/Variables.mk
FILES_TO_FMT      ?= $(shell find . -path ./vendor -prune -o -name '*.go' -print)

define require_clean_work_tree
	@git update-index -q --ignore-submodules --refresh

    @if ! git diff-files --quiet --ignore-submodules --; then \
        echo >&2 "cannot $1: you have unstaged changes."; \
        git diff-files --name-status -r --ignore-submodules -- >&2; \
        echo >&2 "Please commit or stash them."; \
        exit 1; \
    fi

    @if ! git diff-index --cached --quiet HEAD --ignore-submodules --; then \
        echo >&2 "cannot $1: your index contains uncommitted changes."; \
        git diff-index --cached --name-status -r --ignore-submodules HEAD -- >&2; \
        echo >&2 "Please commit or stash them."; \
        exit 1; \
    fi

endef

help: ## Displays help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-z0-9A-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: gen

.PHONY: build
build:
	@cd app && go build -o $(GOBIN) github.com/AnaisUrlichs/observe-argo-rollout/app

.PHONY: docker
docker:
	@cd app && docker build .

.PHONY: format
format: ## Formats Go code including imports and cleans up white noise.
format: $(GOIMPORTS)
	@echo ">> formatting code"
	@$(GOIMPORTS) -w $(FILES_TO_FMT)

.PHONY: gen
gen:
	@cd source && go run . generate -o ../manifests/generated

.PHONY: lint
lint: ## Runs various static analysis against our code.
lint: $(REVIVE) format
	@echo ">> examining all of the Go files"
	@cd app && go vet -stdmethods=false ./...
	@echo ">> linting all of the Go files GOGC=${GOGC}"
	@cd app && $(REVIVE) -config .revive.toml -formatter stylish ./...
	$(call require_clean_work_tree,"detected white noise or/and files without copyright; run 'make lint' file and commit changes.")
