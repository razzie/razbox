package main

import (
	"net/http"
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

func staticPageHandler(r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := path.Clean(r.URL.Path[8:]) // skip /static/
	if razbox.IsFolder(uri) {
		return razlink.ErrorView(r, "Forbidden", http.StatusForbidden)
	}
	return func(w http.ResponseWriter) {
		http.ServeFile(w, r, path.Join("web/static", uri))
	}
}

// GetStaticPage returns a razlink.Page that handles static assets
func GetStaticPage() *razlink.Page {
	return &razlink.Page{
		Path:    "/static/",
		Handler: staticPageHandler,
	}
}
