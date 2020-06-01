package page

import (
	"fmt"
	"net/http"
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type createSubfolderPageView struct {
	Error    string
	Folder   string
	Redirect string
}

func createSubfolderPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	dir := path.Clean(r.URL.Path[18:]) // skip /create-subfolder/
	title := "Create subfolder in " + dir
	v := &createSubfolderPageView{
		Folder:   dir,
		Redirect: "/x/" + dir,
	}

	if r.Method == "POST" {
		r.ParseForm()
		subfolderName := r.FormValue("subfolder")

		token := api.AccessTokenFromCookies(r.Cookies())
		subfolderPath, err := api.CreateSubfolder(token, dir, subfolderName)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		return razlink.RedirectView(r, "/x/"+subfolderPath)
	}

	token := api.AccessTokenFromCookies(r.Cookies())
	flags, err := api.GetFolderFlags(token, dir)
	if err != nil {
		return HandleError(r, err)
	}

	if !flags.EditMode {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	return view(v, &title)
}

// CreateSubfolder returns a razlink.Page that handles subfolder creation
func CreateSubfolder(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/create-subfolder/",
		ContentTemplate: GetContentTemplate("create-subfolder"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return createSubfolderPageHandler(api, r, view)
		},
	}
}
