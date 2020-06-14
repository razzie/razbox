package page

import (
	"net/http"
	"path"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type folderPageView struct {
	Folder       string
	Search       string
	Entries      []*razbox.FolderEntry
	EditMode     bool
	Editable     bool
	Deletable    bool
	Configurable bool
	Gallery      bool
	Redirect     string
}

func folderPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	folderOrFilename := path.Clean(r.URL.Path[3:]) // skip /x/
	tag := r.URL.Query().Get("tag")
	token := api.AccessTokenFromCookies(r.Cookies())
	entries, flags, err := api.GetFolderEntries(token, folderOrFilename)
	if err != nil {
		return HandleError(r, err)
	}

	// this is a file
	if len(entries) == 1 && !entries[0].Folder {
		if strings.HasPrefix(entries[0].MIME, "text/") {
			return razlink.RedirectView(r, "/text/"+folderOrFilename)
		}

		reader, err := api.OpenFile(token, folderOrFilename)
		if err != nil {
			return HandleError(r, err)
		}
		_, download := r.URL.Query()["download"]
		if download {
			return ServeFileAttachment(r, reader)
		}
		return ServeFile(r, reader)
	}

	v := &folderPageView{
		Folder:       folderOrFilename,
		Search:       tag,
		EditMode:     flags.EditMode,
		Editable:     flags.Editable,
		Deletable:    flags.Deletable,
		Configurable: flags.Configurable,
		Redirect:     r.URL.RequestURI(),
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

	return view(v, &folderOrFilename)
}

// Folder returns a razlink.Page that handles folders
func Folder(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/x/",
		ContentTemplate: GetContentTemplate("folder"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return folderPageHandler(api, r, view)
		},
	}
}
