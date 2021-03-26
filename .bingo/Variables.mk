# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.4.0. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for cue variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(CUE)
#	@echo "Running cue"
#	@$(CUE) <flags/args..>
#
CUE := $(GOBIN)/cue-v0.2.2
$(CUE): $(BINGO_DIR)/cue.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/cue-v0.2.2"
	@cd $(BINGO_DIR) && $(GO) build -mod=mod -modfile=cue.mod -o=$(GOBIN)/cue-v0.2.2 "cuelang.org/go/cmd/cue"

