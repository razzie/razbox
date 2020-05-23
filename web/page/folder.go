package page

import (
	"net/http"
	"strings"

	"github.com/razzie/razbox/api"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type folderPageView struct {
	Folder       string
	Search       string
	Entries      []*api.FolderEntry
	EditMode     bool
	Editable     bool
	Configurable bool
	Gallery      bool
	Redirect     string
	SortRedirect string
}

func folderPageHandler(api *api.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	folderOrFilename := r.URL.Path[3:] // skip /x/
	tag := r.URL.Query().Get("tag")
	sortOrder := r.URL.Query().Get("sort")

	token := api.AccessTokenFromCookies(r.Cookies())
	entries, flags, err := api.GetFolderEntries(token, folderOrFilename)
	if err != nil {
		return internal.HandleError(r, err)
	}

	// this is a file
	if flags == nil && len(entries) == 1 {
		if strings.HasPrefix(entries[0].MIME, "text/") {
			return razlink.RedirectView(r, "/text/"+folderOrFilename)
		}

		reader, err := api.OpenFile(token, folderOrFilename)
		if err != nil {
			return internal.HandleError(r, err)
		}
		return internal.ServeFile(r, reader)
	}

	v := &folderPageView{
		Folder:       folderOrFilename,
		Search:       tag,
		EditMode:     flags.EditMode,
		Editable:     flags.Editable,
		Configurable: flags.Configurable,
		Redirect:     r.URL.RequestURI(),
		SortRedirect: r.URL.Path + "?tag=" + tag,
	}

	for _, entry := range entries {
		if len(tag) > 0 && !entry.HasTag(tag) {
			continue
		}
		if !v.Gallery && entry.HasThumbnail {
			v.Gallery = true
		}
		v.Entries = append(v.Entries, entry)
	}

	api.SortFolderEntries(v.Entries, sortOrder)
	return view(v, &folderOrFilename)
}

// Folder returns a razlink.Page that handles folders
func Folder(api *api.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/x/",
		ContentTemplate: internal.GetContentTemplate("folder"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return folderPageHandler(api, r, view)
		},
	}
}
