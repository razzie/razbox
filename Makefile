build:
	go build -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) ./cmd/razbox

.PHONY: build