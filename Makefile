.DEFAULT_GOAL := razbox
VERSION := `git describe --tags`
BUILDFLAGS := -mod=vendor -ldflags="-s -w -X main.version=$(VERSION)" -gcflags=-trimpath=$(CURDIR)

all: razbox mkfolder mkfile

razbox:
	go build $(BUILDFLAGS) ./cmd/razbox

mkfolder:
	go build $(BUILDFLAGS) ./tools/mkfolder

mkfile:
	go build $(BUILDFLAGS) ./tools/mkfile

video:
	gource --file-filter vendor/ -a 1 -s 3 -c 2 -r 25 -1280x720 --multi-sampling -o - | \
	ffmpeg -y -r 25 -f image2pipe -vcodec ppm -i - -vcodec libx264 -crf 20 -pix_fmt yuv420p -threads 0 -bf 0 razbox.mp4

.PHONY: all razbox mkfolder mkfile video
