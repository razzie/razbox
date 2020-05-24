package razbox

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/razzie/razbox/internal"
)

const thumbnailRetryAfter = time.Hour

// Thumbnail ...
type Thumbnail struct {
	Data   []byte
	MIME   string
	Bounds ThumbnailBounds
}

// ThumbnailBounds ...
type ThumbnailBounds struct {
	Width  int
	Height int
}

func newThumbnail(thumb *internal.Thumbnail) *Thumbnail {
	return &Thumbnail{
		Data: thumb.Data,
		MIME: thumb.MIME,
		Bounds: ThumbnailBounds{
			Width:  thumb.Bounds.Dx(),
			Height: thumb.Bounds.Dy(),
		},
	}
}

// GetFileThumbnail ...
func (api API) GetFileThumbnail(token *AccessToken, filePath string) (*Thumbnail, error) {
	filePath = internal.RemoveTrailingSlash(filePath)
	dir := path.Dir(filePath)
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: dir}
	}

	basename := filepath.Base(filePath)
	file, err := folder.GetFile(basename)
	if err != nil {
		return nil, &ErrNotFound{}
	}

	if !internal.IsThumbnailSupported(file.MIME) {
		return nil, fmt.Errorf("unsupported format: %s", file.MIME)
	}

	thumb := file.Thumbnail
	if thumb == nil || (len(thumb.Data) == 0 && thumb.Timestamp.Add(thumbnailRetryAfter).Before(time.Now())) {
		data, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer data.Close()

		thumb, err = internal.GetThumbnail(data, file.MIME)
		defer file.Save()
		if err != nil {
			file.Thumbnail = &internal.Thumbnail{Timestamp: time.Now()}
			return nil, err
		}

		file.Thumbnail = thumb
	}

	return newThumbnail(thumb), nil
}
