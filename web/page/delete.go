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

func deletePageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[8:] // skip /delete/
	filename = lib.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	var folder *lib.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		cached = false
		folder, err = lib.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()))
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
	return razlink.RedirectView(r, redirect)
}

// Delete returns a razlink.Page that handles deletes
func Delete(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path: "/delete/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return deletePageHandler(db, r, view)
		},
	}
}
