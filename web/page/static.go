package page

import (
	"net/http"
	"os"
	"path"

	"github.com/razzie/razlink"
)

func staticPageHandler(pr *razlink.PageRequest) *razlink.View {
	uri := path.Clean(pr.RelPath)
	if fi, _ := os.Stat(path.Join("web/static", uri)); fi != nil && fi.IsDir() {
		return pr.ErrorView("Forbidden", http.StatusForbidden)
	}
	return pr.HandlerView(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("web/static", uri))
	})
}

// Static returns a razlink.Page that handles static assets
func Static() *razlink.Page {
	return &razlink.Page{
		Path:    "/static/",
		Handler: staticPageHandler,
	}
}
