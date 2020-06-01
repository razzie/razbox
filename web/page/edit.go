package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type editPageView struct {
	Error    string
	Folder   string
	Filename string
	Tags     string
	Public   bool
	Redirect string
	Thumb    bool
}

func editPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := path.Clean(r.URL.Path[6:]) // skip /edit/
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	token := api.AccessTokenFromCookies(r.Cookies())
	entry, _, err := api.GetFolderEntries(token, filename)
	if err != nil {
		return HandleError(r, err)
	}

	if !entry[0].EditMode {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	v := &editPageView{
		Folder:   dir,
		Filename: entry[0].Name,
		Tags:     strings.Join(entry[0].Tags, " "),
		Public:   entry[0].Public,
		Redirect: redirect,
		Thumb:    entry[0].HasThumbnail,
	}
	title := "Edit " + filename

	if r.Method == "POST" {
		r.ParseForm()

		o := &razbox.EditFileOptions{
			Folder:           dir,
			OriginalFilename: entry[0].Name,
			NewFilename:      r.FormValue("filename"),
			Tags:             strings.Fields(r.FormValue("tags")),
			Public:           r.FormValue("public") == "public",
		}
		err := api.EditFile(token, o)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		return razlink.RedirectView(r, redirect)
	}

	return view(v, &title)
}

// Edit returns a razlink.Page that handles edits
func Edit(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/edit/",
		ContentTemplate: GetContentTemplate("edit"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return editPageHandler(api, r, view)
		},
	}
}
