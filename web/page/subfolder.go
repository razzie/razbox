package page

import (
	"fmt"
	"net/http"
	"path"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type createSubfolderPageView struct {
	Error    string `json:"error,omitempty"`
	Folder   string `json:"folder,omitempty"`
	Redirect string `json:"redirect,omitempty"`
}

func createSubfolderPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	dir := path.Clean(pr.RelPath)
	pr.Title = "Create subfolder in " + dir
	v := &createSubfolderPageView{
		Folder:   dir,
		Redirect: "/x/" + dir,
	}

	if r.Method == "POST" {
		r.ParseForm()
		subfolderName := r.FormValue("subfolder")

		token := api.AccessTokenFromRequest(r)
		subfolderPath, err := api.CreateSubfolder(token, dir, subfolderName)
		if err != nil {
			v.Error = err.Error()
			return pr.Respond(v, beepboop.WithError(err, http.StatusInternalServerError))
		}

		return pr.RedirectView("/x/" + subfolderPath)
	}

	token := api.AccessTokenFromRequest(r)
	flags, err := api.GetFolderFlags(token, dir)
	if err != nil {
		return HandleError(r, err)
	}

	if !flags.EditMode {
		return pr.RedirectView(
			fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()),
			beepboop.WithErrorMessage("Write access required", http.StatusUnauthorized))
	}

	return pr.Respond(v)
}

func deleteSubfolderPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	dir := path.Clean(pr.RelPath)
	parent := path.Dir(dir)

	token := api.AccessTokenFromRequest(r)
	err := api.DeleteSubfolder(token, parent, path.Base(dir))
	if err != nil {
		return HandleError(r, err)
	}

	return pr.RedirectView("/x/" + parent)
}

// CreateSubfolder returns a beepboop.Page that handles subfolder creation
func CreateSubfolder(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/create-subfolder/",
		ContentTemplate: GetContentTemplate("create-subfolder"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return createSubfolderPageHandler(api, pr)
		},
	}
}

// DeleteSubfolder returns a beepboop.Page that handles subfolder deletion
func DeleteSubfolder(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path: "/delete-subfolder/",
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return deleteSubfolderPageHandler(api, pr)
		},
	}
}
