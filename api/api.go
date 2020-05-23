package api

import (
	"path/filepath"

	"github.com/razzie/razbox/lib"
)

// API ...
type API struct {
	root string
	db   *lib.DB
}

// New ...
func New(root string) (*API, error) {
	if !filepath.IsAbs(root) {
		var err error
		root, err = filepath.Abs(root)
		if err != nil {
			return nil, err
		}
	}

	return &API{
		root: root,
	}, nil
}

// ConnectDB ...
func (api *API) ConnectDB(redisAddr, redisPw string, redisDb int) error {
	db, err := lib.NewDB(redisAddr, redisPw, redisDb)
	if err != nil {
		return err
	}

	api.db = db
	return nil
}

func (api API) getFolder(folderName string) (folder *lib.Folder, cached bool, err error) {
	cached = true
	if api.db != nil {
		folder, _ = api.db.GetCachedFolder(folderName)
	}
	if folder == nil {
		cached = false
		folder, err = lib.GetFolder(api.root, folderName)
	}
	return
}

func (api API) goCacheFolder(folder *lib.Folder) {
	if api.db != nil {
		go api.db.CacheFolder(folder)
	}
}

func (api API) getThumbnail(filename string, file *lib.File) (*Thumbnail, error) {
	var thumb *lib.Thumbnail
	thumbCached := true

	if api.db != nil {
		thumb, _ = api.db.GetCachedThumbnail(filename)
	}
	if thumb == nil {
		thumbCached = false
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		thumb, err = lib.GetThumbnail(reader, file.MIME)
		if err != nil {
			return nil, err
		}
	}

	if api.db != nil && !thumbCached {
		defer api.db.CacheThumbnail(filename, thumb)
	}

	return &Thumbnail{
		Data:   thumb.Data,
		MIME:   thumb.MIME,
		Bounds: thumb.Bounds,
	}, nil
}
