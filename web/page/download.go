package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type downloadPageView struct {
	Error       string
	Folder      string
	MaxFileSize string
}

func downloadPageHandler(api *razbox.API, pr *razlink.PageRequest) *razlink.View {
	r := pr.Request
	dir := path.Clean(pr.RelPath)
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

	pr.Title = "Download file to " + dir
	v := &uploadPageView{
		Folder:      dir,
		MaxFileSize: fmt.Sprintf("%dMB", flags.MaxUploadSizeMB),
	}

	if r.Method == "POST" {
		r.ParseForm()

		o := &razbox.DownloadFileToFolderOptions{
			Folder:    dir,
			URL:       r.FormValue("url"),
			Filename:  r.FormValue("filename"),
			Tags:      strings.Fields(r.FormValue("tags")),
			Overwrite: r.FormValue("overwrite") == "overwrite",
			Public:    r.FormValue("public") == "public",
		}
		err := api.DownloadFileToFolder(token, o)
		if err != nil {
			v.Error = err.Error()
			return pr.Respond(v, razlink.WithError(err, http.StatusInternalServerError))
		}

		return razlink.RedirectView(r, "/x/"+dir)
	}

	return pr.Respond(v)
}

// Download returns a razlink.Page that handles file downloads from an URL to a folder
func Download(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/download-to-folder/",
		ContentTemplate: GetContentTemplate("download"),
		Handler: func(pr *razlink.PageRequest) *razlink.View {
			return downloadPageHandler(api, pr)
		},
	}
}
