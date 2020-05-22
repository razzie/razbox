package page

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type textPageView struct {
	Filename string
	Folder   string
	Text     string
}

func textPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[6:] // skip /text/
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

	hasViewAccess := folder.EnsureReadAccess(r) == nil
	basename := filepath.Base(filename)
	file, err := folder.GetFile(basename)
	if err != nil {
		if !hasViewAccess { // fake legacy behavior
			return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
		}
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "File not found", http.StatusNotFound)
	}

	if !file.Public && !hasViewAccess {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	if !strings.HasPrefix(file.MIME, "text/") {
		return razlink.ErrorView(r, "Not a text file", http.StatusInternalServerError)
	}

	_, download := r.URL.Query()["download"]
	if download {
		return func(w http.ResponseWriter) {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", basename))
			file.ServeHTTP(w, r)
		}
	}

	reader, err := file.Open()
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Could not open file", http.StatusInternalServerError)
	}
	defer reader.Close()

	data, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(filename, "error:", err.Error())
		return razlink.ErrorView(r, "Could not read file", http.StatusInternalServerError)
	}

	v := &textPageView{
		Filename: basename,
		Folder:   dir,
		Text:     string(data),
	}
	return view(v, &filename)
}

// Text returns a razlink.Page that visualizes text files
func Text(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/text/",
		ContentTemplate: internal.GetContentTemplate("text"),
		Stylesheets: []string{
			"/static/highlight.tomorrow.min.css",
			"/static/highlightjs-line-numbers.css",
		},
		Scripts: []string{
			"/static/highlight.min.js",
			"/static/highlightjs-line-numbers.min.js",
		},
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return textPageHandler(db, r, view)
		},
	}
}
