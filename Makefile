.DEFAULT_GOAL := razbox

razbox:
	go build -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./cmd/razbox

mkfolder:
	go build -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./tools/mkfolder

mkfile:
	go build -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./tools/mkfile

.PHONY: razbox mkfolder mkfile