package page

import (
	"net/http"
	"os"
	"path"

	"github.com/razzie/beepboop"
)

func staticPageHandler(pr *beepboop.PageRequest) *beepboop.View {
	uri := path.Clean(pr.RelPath)
	if fi, _ := os.Stat(path.Join("web/static", uri)); fi != nil && fi.IsDir() {
		return pr.ErrorView("Forbidden", http.StatusForbidden)
	}
	return pr.HandlerView(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("web/static", uri))
	})
}

// Static returns a beepboop.Page that handles static assets
func Static() *beepboop.Page {
	return &beepboop.Page{
		Path:    "/static/",
		Handler: staticPageHandler,
	}
}
