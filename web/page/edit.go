package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/razbox/api"
	"github.com/razzie/razbox/web/page/internal"
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

func editPageHandler(a *api.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	filename := r.URL.Path[6:] // skip /edit/
	filename = internal.RemoveTrailingSlash(filename)
	dir := path.Dir(filename)
	redirect := r.URL.Query().Get("r")
	if len(redirect) == 0 {
		redirect = "/x/" + dir
	}

	token := a.AccessTokenFromCookies(r.Cookies())
	entry, _, err := a.GetFolderEntries(token, filename)
	if err != nil {
		return internal.HandleError(r, err)
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
		Thumb:    strings.HasPrefix(entry[0].MIME, "image/"),
	}
	title := "Edit " + filename

	if r.Method == "POST" {
		r.ParseForm()

		o := &api.EditFileOptions{
			Folder:           dir,
			OriginalFilename: entry[0].Name,
			NewFilename:      r.FormValue("filename"),
			Tags:             strings.Fields(r.FormValue("tags")),
			Public:           r.FormValue("public") == "public",
		}
		err := a.EditFile(token, o)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		return razlink.RedirectView(r, redirect)
	}

	return view(v, &title)
}

// Edit returns a razlink.Page that handles edits
func Edit(api *api.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/edit/",
		ContentTemplate: internal.GetContentTemplate("edit"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return editPageHandler(api, r, view)
		},
	}
}
