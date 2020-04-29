build:
	go build -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR) .

.PHONY: build