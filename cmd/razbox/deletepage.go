package main

import (
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

func deletePageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[8:] // skip /delete/
	filename = razbox.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)

	var folder *razbox.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		cached = false
		folder, err = razbox.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.Path))
	}

	basename := filepath.Base(filename)
	file, err := folder.GetFile(basename)
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "File not found", http.StatusNotFound)
	}

	err = file.Delete()
	if err != nil {
		return razlink.ErrorView(r, "Cannot delete file", http.StatusInternalServerError)
	}

	folder.CachedFiles = nil
	return razlink.RedirectView(r, "/x/"+dir)
}

// GetDeletePage returns a razlink.Page that handles deletes
func GetDeletePage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path: "/delete/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return deletePageHandler(db, r, view)
		},
	}
}
