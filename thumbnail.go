package razbox

import (
	"bytes"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"strconv"

	"github.com/nfnt/resize"
	"golang.org/x/image/webp"
)

func init() {
	image.RegisterFormat("jpeg", "jpeg", jpeg.Decode, jpeg.DecodeConfig)
	image.RegisterFormat("gif", "gif", gif.Decode, gif.DecodeConfig)
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	image.RegisterFormat("webp", "webp", webp.Decode, webp.DecodeConfig)
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

// GetThumbnail returns a thumbnail image in bytes from an io.Reader
func GetThumbnail(img io.Reader) (*Thumbnail, error) {
	src, _, err := image.Decode(img)
	if err != nil {
		return nil, err
	}

	dst := resize.Thumbnail(250, 500, src, resize.NearestNeighbor)

	var result bytes.Buffer
	err = jpeg.Encode(&result, dst, &jpeg.Options{Quality: 90})
	if err != nil {
		return nil, err
	}

	return &Thumbnail{
		Data:   result.Bytes(),
		MIME:   "image/jpeg",
		Bounds: dst.Bounds(),
	}, nil
}
