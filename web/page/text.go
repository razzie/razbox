package page

import (
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razbox/internal"
	"github.com/razzie/razlink"
)

type textPageView struct {
	Filename string
	Folder   string
	Text     string
}

func textPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[6:] // skip /text/
	filename = internal.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)

	token := api.AccessTokenFromCookies(r.Cookies())
	file, err := api.OpenFile(token, filename)
	if err != nil {
		return HandleError(r, err)
	}
	_, download := r.URL.Query()["download"]
	if download {
		return ServeFileAttachment(r, file)
	}
	defer file.Close()

	if !strings.HasPrefix(file.MIME, "text/") {
		return razlink.ErrorView(r, "Not a text file", http.StatusInternalServerError)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return razlink.ErrorView(r, "Could not read file", http.StatusInternalServerError)
	}

	v := &textPageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
		Text:     string(data),
	}
	return view(v, &filename)
}

// Text returns a razlink.Page that visualizes text files
func Text(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/text/",
		ContentTemplate: GetContentTemplate("text"),
		Stylesheets: []string{
			"/static/highlight.tomorrow.min.css",
			"/static/highlightjs-line-numbers.css",
		},
		Scripts: []string{
			"/static/highlight.min.js",
			"/static/highlightjs-line-numbers.min.js",
		},
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return textPageHandler(api, r, view)
		},
	}
}
