package page

import (
	"net/http"
	"os"
	"path"

	"github.com/razzie/razlink"
)

func staticPageHandler(r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := path.Clean(r.URL.Path[8:]) // skip /static/
	if fi, _ := os.Stat(path.Join("web/static", uri)); fi != nil && fi.IsDir() {
		return razlink.ErrorView(r, "Forbidden", http.StatusForbidden)
	}
	return func(w http.ResponseWriter) {
		http.ServeFile(w, r, path.Join("web/static", uri))
	}
}

// Static returns a razlink.Page that handles static assets
func Static() *razlink.Page {
	return &razlink.Page{
		Path:    "/static/",
		Handler: staticPageHandler,
	}
}
