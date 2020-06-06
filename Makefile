.DEFAULT_GOAL := razbox
BUILDFLAGS := -mod=vendor -ldflags="-s -w" -gcflags=-trimpath=$(CURDIR)

all: razbox mkfolder mkfile

razbox:
	go build $(BUILDFLAGS) ./cmd/razbox

mkfolder:
	go build $(BUILDFLAGS) ./tools/mkfolder

mkfile:
	go build $(BUILDFLAGS) ./tools/mkfile

video:
	gource --file-filter vendor/ -a 1 -s 3 -c 2 -r 25 -o - | \
	ffmpeg -y -r 25 -f image2pipe -vcodec ppm -i - -vcodec libx264 -preset ultrafast -pix_fmt yuv420p -crf 1 -threads 0 -bf 0 gource.mp4

.PHONY: all razbox mkfolder mkfile video