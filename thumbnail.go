package razbox

import (
	"fmt"
	"path"
	"path/filepath"
	"time"

	"github.com/razzie/razbox/internal"
)

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
func (api API) GetFileThumbnail(token *AccessToken, filePath string, maxWidth uint) (*Thumbnail, error) {
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
	if thumb == nil || (len(thumb.Data) == 0 && thumb.Timestamp.Add(api.ThumbnailRetryAfter).Before(time.Now())) {
		thumb, err = internal.GetThumbnail(path.Join(api.root, file.RelPath+".bin"), file.MIME, maxWidth)
		defer file.Save()
		if err != nil {
			file.Thumbnail = &internal.Thumbnail{Timestamp: time.Now()}
			return nil, err
		}

		file.Thumbnail = thumb
	}

	return newThumbnail(thumb), nil
}
