package page

import (
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type textPageView struct {
	Filename string
	Folder   string
	Text     string
}

func textPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	token := api.AccessTokenFromRequest(r)
	file, err := api.OpenFile(token, filename)
	if err != nil {
		return HandleError(r, err)
	}
	_, download := r.URL.Query()["download"]
	if download {
		return ServeFileAttachmentAsync(r, file)
	}
	defer file.Close()

	if !strings.HasPrefix(file.MIME, "text/") {
		return razlink.RedirectView(r, "/x/"+filename)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return razlink.ErrorView(r, "Could not read file", http.StatusInternalServerError)
	}

	pr.Title = filename
	v := &textPageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
		Text:     string(data),
	}
	return pr.Respond(v)
}

// Text returns a razlink.Page that visualizes text files
func Text(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/text/",
		ContentTemplate: GetContentTemplate("text"),
		Stylesheets: []string{
			"/static/highlight.tomorrow.min.css",
		},
		Scripts: []string{
			"/static/highlight.min.js",
			"/static/highlightjs-line-numbers.min.js",
		},
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return textPageHandler(api, pr)
		},
	}
}
