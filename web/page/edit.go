package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type editPageView struct {
	Error      string   `json:"error,omitempty"`
	Folder     string   `json:"folder,omitempty"`
	Filename   string   `json:"filename,omitempty"`
	Tags       string   `json:"tags,omitempty"`
	Public     bool     `json:"public,omitempty"`
	Redirect   string   `json:"redirect,omitempty"`
	Thumb      bool     `json:"thumb,omitempty"`
	Subfolders []string `json:"subfolders,omitempty"`
}

func editPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	token := api.AccessTokenFromRequest(r)
	entry, _, err := api.GetFolderEntries(token, filename)
	if err != nil {
		return HandleError(r, err)
	}

	if !entry[0].EditMode {
		return pr.RedirectView(
			fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()),
			beepboop.WithErrorMessage("Write access required", http.StatusUnauthorized))
	}

	pr.Title = "Edit " + filename
	subfolders, _ := api.GetSubfolders(token, dir)
	v := &editPageView{
		Folder:     dir,
		Filename:   entry[0].Name,
		Tags:       strings.Join(entry[0].Tags, " "),
		Public:     entry[0].Public,
		Redirect:   redirect,
		Thumb:      entry[0].HasThumbnail,
		Subfolders: subfolders,
	}

	if r.Method == "POST" {
		r.ParseForm()

		o := &razbox.EditFileOptions{
			Folder:           dir,
			OriginalFilename: entry[0].Name,
			NewFilename:      r.FormValue("filename"),
			Tags:             strings.Fields(r.FormValue("tags")),
			Public:           r.FormValue("public") == "public",
			MoveTo:           r.FormValue("move"),
		}
		err := api.EditFile(token, o)
		if err != nil {
			v.Error = err.Error()
			return pr.Respond(v, beepboop.WithError(err, http.StatusInternalServerError))
		}

		return pr.RedirectView(redirect)
	}

	return pr.Respond(v)
}

// Edit returns a beepboop.Page that handles edits
func Edit(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/edit/",
		ContentTemplate: GetContentTemplate("edit"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return editPageHandler(api, pr)
		},
	}
}
