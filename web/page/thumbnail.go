package page

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/razzie/razbox/lib"
	"github.com/razzie/razlink"
)

func thumbnailPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[7:] // skip /thumb/
	filename = lib.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)

	var folder *lib.Folder
	var err error
	folderCached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		folderCached = false
		folder, err = lib.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !folderCached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	basename := filepath.Base(filename)
	file, err := folder.GetFile(basename)
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "File not found", http.StatusNotFound)
	}

	if !lib.IsThumbnailSupported(file.MIME) {
		return razlink.ErrorView(r, "Unsupported format: "+file.MIME, http.StatusInternalServerError)
	}

	var thumb *lib.Thumbnail
	thumbCached := true

	if db != nil {
		thumb, _ = db.GetCachedThumbnail(filename)
	}
	if thumb == nil {
		thumbCached = false
		reader, err := file.Open()
		if err != nil {
			log.Println(filename, "error:", err.Error())
			return razlink.ErrorView(r, "Could not open file", http.StatusInternalServerError)
		}
		defer reader.Close()
		thumb, err = lib.GetThumbnail(reader, file.MIME)
		if err != nil {
			log.Println(filename, "error:", err.Error())
			return razlink.RedirectView(r, "/x/"+filename)
		}
	}

	if db != nil && !thumbCached {
		defer db.CacheThumbnail(filename, thumb)
	}

	return func(w http.ResponseWriter) {
		thumb.ServeHTTP(w, r)
	}
}

// Thumbnail returns a razlink.Page that handles image file thumbnails
func Thumbnail(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path: "/thumb/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return thumbnailPageHandler(db, r, view)
		},
	}
}
