package internal

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/nfnt/resize"
	"golang.org/x/image/webp"
)

// MaxThumbnailWidth ...
const MaxThumbnailWidth = 250

var ffmpegOK bool

func init() {
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("webp", "webp", webp.Decode, webp.DecodeConfig)

	cmd := exec.Command("ffmpeg", "-version")
	ffmpegOK = cmd.Run() == nil
}

// Thumbnail contains a thumbnail image in bytes + the MIME type and bounds
type Thumbnail struct {
	Data      []byte          `json:"data"`
	MIME      string          `json:"mime"`
	Bounds    image.Rectangle `json:"bounds"`
	Timestamp time.Time       `json:"timestamp"`
}

// IsThumbnailSupported returns whether thumbnails can be created for the specified mime type
func IsThumbnailSupported(mime string) bool {
	if strings.HasPrefix(mime, "image/") {
		return true
	}

	if ffmpegOK && strings.HasPrefix(mime, "video/") {
		return true
	}

	return false
}

// GetThumbnail returns the thumbnail of a media file
func GetThumbnail(filename string, mime string) (*Thumbnail, error) {
	if strings.HasPrefix(mime, "image/") {
		f, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			return nil, err
		}
		return getThumbnailImage(img, MaxThumbnailWidth)
	}

	if ffmpegOK && strings.HasPrefix(mime, "video/") {
		return getThumbnailFFMPEG(filename, MaxThumbnailWidth)
	}

	return nil, &ErrUnsupportedFileFormat{MIME: mime}
}

func getThumbnailImage(img image.Image, maxWidth uint) (*Thumbnail, error) {
	thumb := resize.Thumbnail(maxWidth, maxWidth*2, img, resize.NearestNeighbor)

	var result bytes.Buffer
	err := jpeg.Encode(&result, thumb, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return &Thumbnail{
		Data:      result.Bytes(),
		MIME:      "image/jpeg",
		Bounds:    thumb.Bounds(),
		Timestamp: time.Now(),
	}, nil
}

func getThumbnailFFMPEG(filename string, maxWidth uint) (*Thumbnail, error) {
	cmd := exec.Command("ffmpeg",
		"-hide_banner",
		"-loglevel", "error",
		"-i", filename,
		"-ss", "00:00:01.000",
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale=%d:-2", maxWidth),
		"-f", "singlejpeg", "-")

	var output, stderr bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("[%s] %s", err.Error(), string(stderr.Bytes()))
	}

	cfg, err := jpeg.DecodeConfig(bytes.NewBuffer(output.Bytes()))

	return &Thumbnail{
		Data:      output.Bytes(),
		MIME:      "image/jpeg",
		Bounds:    image.Rect(0, 0, cfg.Width, cfg.Height),
		Timestamp: time.Now(),
	}, err
}
