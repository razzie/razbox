package main

import (
	"net/http"
	"path"

	"github.com/razzie/razlink"
)

func staticPageHandler(w http.ResponseWriter, r *http.Request) {
	uri := path.Clean(r.URL.Path[8:]) // skip /static/
	http.ServeFile(w, r, path.Join("web/static", uri))
}

// GetStaticPage returns a razlink.Page that handles static assets
func GetStaticPage() *razlink.Page {
	return &razlink.Page{
		Path: "/static/",
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return func(w http.ResponseWriter) {
				staticPageHandler(w, r)
			}
		},
	}
}
