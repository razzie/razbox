package page

import (
	"fmt"
	"net/http"
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type createSubfolderPageView struct {
	Error    string `json:"error,omitempty"`
	Folder   string `json:"folder,omitempty"`
	Redirect string `json:"redirect,omitempty"`
}

func createSubfolderPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
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
			return pr.Respond(v, razlink.WithError(err, http.StatusInternalServerError))
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
			razlink.WithErrorMessage("Write access required", http.StatusUnauthorized))
	}

	return pr.Respond(v)
}

func deleteSubfolderPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
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

// CreateSubfolder returns a razlink.Page that handles subfolder creation
func CreateSubfolder(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/create-subfolder/",
		ContentTemplate: GetContentTemplate("create-subfolder"),
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return createSubfolderPageHandler(api, pr)
		},
	}
}

// DeleteSubfolder returns a razlink.Page that handles subfolder deletion
func DeleteSubfolder(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path: "/delete-subfolder/",
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return deleteSubfolderPageHandler(api, pr)
		},
	}
}
