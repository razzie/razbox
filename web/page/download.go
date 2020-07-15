package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type downloadPageView struct {
	Error       string `json:"error,omitempty"`
	Folder      string `json:"folder,omitempty"`
	MaxFileSize string `json:"max_file_size,omitempty"`
}

func downloadPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
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
			beepboop.WithErrorMessage("Write access required", http.StatusUnauthorized))
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
			return pr.Respond(v, beepboop.WithError(err, http.StatusInternalServerError))
		}

		return pr.RedirectView("/x/" + dir)
	}

	return pr.Respond(v)
}

// Download returns a beepboop.Page that handles file downloads from an URL to a folder
func Download(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/download-to-folder/",
		ContentTemplate: GetContentTemplate("download"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return downloadPageHandler(api, pr)
		},
	}
}
