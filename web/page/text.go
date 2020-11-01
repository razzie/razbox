package page

import (
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type textPageView struct {
	Filename string `json:"filename,omitempty"`
	Folder   string `json:"folder,omitempty"`
	Text     string `json:"text,omitempty"`
}

func textPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	token := beepboop.NewAccessTokenFromRequest(pr)
	file, err := api.OpenFile(token, filename)
	if err != nil {
		return HandleError(r, err)
	}
	defer file.Close()

	if !strings.HasPrefix(file.MimeType(), "text/") {
		return pr.RedirectView("/x/" + filename)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return pr.ErrorView("Could not read file", http.StatusInternalServerError)
	}

	pr.Title = filename
	v := &textPageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
		Text:     string(data),
	}
	return pr.Respond(v)
}

// Text returns a beepboop.Page that visualizes text files
func Text(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/text/",
		ContentTemplate: GetContentTemplate("text"),
		Stylesheets: []string{
			"/static/highlight.tomorrow.min.css",
		},
		Scripts: []string{
			"/static/highlight.min.js",
			"/static/highlightjs-line-numbers.min.js",
		},
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return textPageHandler(api, pr)
		},
	}
}
