.DEFAULT_GOAL := razbox
VERSION := `git describe --tags`
BUILDFLAGS := -mod=vendor -ldflags="-s -w -X main.version=$(VERSION)" -gcflags=-trimpath=$(CURDIR)
IMAGE_NAME := razbox
IMAGE_REGISTRY ?= ghcr.io/razzie
FULL_IMAGE_NAME := $(IMAGE_REGISTRY)/$(IMAGE_NAME):$(VERSION)

.PHONY: all
all: razbox mkfolder mkfile

.PHONY: razbox
razbox:
	go build $(BUILDFLAGS) ./cmd/razbox

.PHONY: mkfolder
mkfolder:
	go build $(BUILDFLAGS) ./tools/mkfolder

.PHONY: mkfile
mkfile:
	go build $(BUILDFLAGS) ./tools/mkfile

.PHONY: video
video:
	gource --file-filter vendor/ -a 1 -s 3 -c 2 -r 25 -1280x720 --multi-sampling -o - | \
	ffmpeg -y -r 25 -f image2pipe -vcodec ppm -i - -vcodec libx264 -crf 20 -pix_fmt yuv420p -threads 0 -bf 0 razbox.mp4

.PHONY: docker-build
docker-build:
	docker build . -t $(FULL_IMAGE_NAME)

.PHONY: docker-push
docker-push: docker-build
	docker push $(FULL_IMAGE_NAME)
