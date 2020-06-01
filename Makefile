.DEFAULT_GOAL := razbox
BUILDFLAGS := -mod=vendor -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR)

all: razbox mkfolder mkfile

razbox:
	go build $(BUILDFLAGS) ./cmd/razbox

mkfolder:
	go build $(BUILDFLAGS) ./tools/mkfolder

mkfile:
	go build $(BUILDFLAGS) ./tools/mkfile

.PHONY: all razbox mkfolder mkfile