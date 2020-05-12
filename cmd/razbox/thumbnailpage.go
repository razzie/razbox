package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

func handleThumbnailPage(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[7:] // skip /thumb/
	filename = razbox.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)

	var folder *razbox.Folder
	var err error
	folderCached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		folderCached = false
		folder, err = razbox.GetFolder(dir)
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

	if !strings.HasPrefix(file.MIME, "image/") {
		return razlink.ErrorView(r, "Not an image", http.StatusInternalServerError)
	}

	var thumb *razbox.Thumbnail
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
		thumb, err = razbox.GetThumbnail(reader)
		if err != nil {
			log.Println(filename, "error:", err.Error())
			return razlink.ErrorView(r, "Could not create thumbnail", http.StatusInternalServerError)
		}
	}

	if db != nil && !thumbCached {
		defer db.CacheThumbnail(filename, thumb)
	}

	return func(w http.ResponseWriter) {
		thumb.ServeHTTP(w, r)
	}
}

// GetThumbnailPage returns a razlink.Page that handles image file thumbnails
func GetThumbnailPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path: "/thumb/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return handleThumbnailPage(db, r, view)
		},
	}
}
