.DEFAULT_GOAL := razbox

razbox:
	go build -mod=vendor -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./cmd/razbox

mkfolder:
	go build -mod=vendor -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./tools/mkfolder

mkfile:
	go build -mod=vendor -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./tools/mkfile

.PHONY: razbox mkfolder mkfile