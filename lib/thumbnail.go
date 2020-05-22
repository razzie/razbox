package lib

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nfnt/resize"
	"golang.org/x/image/webp"
)

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
	Data   []byte          `json:"data"`
	MIME   string          `json:"mime"`
	Bounds image.Rectangle `json:"bounds"`
}

func (t Thumbnail) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", t.MIME)
	w.Header().Set("Content-Length", strconv.Itoa(len(t.Data)))
	w.Write(t.Data)
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

// GetThumbnail returns a thumbnail image in bytes from an io.Reader
func GetThumbnail(r io.ReadSeeker, mime string) (*Thumbnail, error) {
	if strings.HasPrefix(mime, "image/") {
		img, _, err := image.Decode(r)
		if err != nil {
			return nil, err
		}
		return getThumbnailImage(img)
	}

	if ffmpegOK && strings.HasPrefix(mime, "video/") {
		return getThumbnailFFMPEG(r)
	}

	return nil, fmt.Errorf("unsupported format: %s", mime)
}

func getThumbnailImage(img image.Image) (*Thumbnail, error) {
	thumb := resize.Thumbnail(250, 500, img, resize.NearestNeighbor)

	var result bytes.Buffer
	err := jpeg.Encode(&result, thumb, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return &Thumbnail{
		Data:   result.Bytes(),
		MIME:   "image/jpeg",
		Bounds: thumb.Bounds(),
	}, nil
}

func getThumbnailFFMPEG(r io.ReadSeeker) (*Thumbnail, error) {
	var buffer bytes.Buffer
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-ss", "00:00:01.000", "-vframes", "1", "-f", "singlejpeg", "-")
	cmd.Stdin = r
	cmd.Stdout = &buffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(&buffer)
	if err != nil {
		return nil, err
	}

	return getThumbnailImage(img)
}
