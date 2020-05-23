package page

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/razzie/razbox/api"
	"github.com/razzie/razbox/web/page/internal"
	"github.com/razzie/razlink"
)

type downloadPageView struct {
	Error       string
	Folder      string
	MaxFileSize string
}

func downloadPageHandler(a *api.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[20:] // skip /download-to-folder/
	uri = internal.RemoveTrailingSlash(uri)

	token := a.AccessTokenFromCookies(r.Cookies())
	flags, err := a.GetFolderFlags(token, uri)
	if err != nil {
		return internal.HandleError(r, err)
	}

	if !flags.EditMode {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	title := "Download file to " + uri
	v := &uploadPageView{
		Folder:      uri,
		MaxFileSize: fmt.Sprintf("%dMB", flags.MaxFileSizeMB),
	}

	if r.Method == "POST" {
		r.ParseForm()

		o := &api.DownloadFileToFolderOptions{
			Folder:   uri,
			URL:      r.FormValue("url"),
			Filename: r.FormValue("filename"),
			Tags:     strings.Fields(r.FormValue("tags")),
		}
		err := a.DownloadFileToFolder(token, o)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		return razlink.RedirectView(r, "/x/"+uri)
	}

	return view(v, &title)
}

// Download returns a razlink.Page that handles file downloads from an URL to a folder
func Download(api *api.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/download-to-folder/",
		ContentTemplate: internal.GetContentTemplate("download"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return downloadPageHandler(api, r, view)
		},
	}
}
