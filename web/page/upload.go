package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type uploadPageView struct {
	Error       string `json:"error,omitempty"`
	Folder      string `json:"folder,omitempty"`
	MaxFileSize string `json:"max_file_size,omitempty"`
}

func uploadPageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
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

	pr.Title = "Upload file to " + dir
	v := &uploadPageView{
		Folder:      dir,
		MaxFileSize: fmt.Sprintf("%dMB", flags.MaxUploadSizeMB),
	}
	handleError := func(err error) *beepboop.View {
		v.Error = err.Error()
		return pr.Respond(v, beepboop.WithError(err, http.StatusInternalServerError))
	}

	if r.Method == "POST" {
		limit := (flags.MaxUploadSizeMB + 10) << 20
		if r.ContentLength > limit {
			return handleError(&razbox.ErrSizeLimitExceeded{})
		}

		r.Body = &razbox.LimitedReadCloser{
			R: r.Body,
			N: limit,
		}
		r.ParseMultipartForm(1 << 20)
		defer r.MultipartForm.RemoveAll()

		o := &razbox.UploadFileOptions{
			Folder:    dir,
			Files:     r.MultipartForm.File["files"],
			Filename:  r.FormValue("filename"),
			Tags:      strings.Fields(r.FormValue("tags")),
			Overwrite: r.FormValue("overwrite") == "overwrite",
			Public:    r.FormValue("public") == "public",
		}
		err = api.UploadFile(token, o)
		if err != nil {
			return handleError(err)
		}

		return pr.RedirectView("/x/" + dir)
	}

	return pr.Respond(v)
}

// Upload returns a beepboop.Page that handles file uploads
func Upload(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/upload/",
		ContentTemplate: GetContentTemplate("upload"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return uploadPageHandler(api, pr)
		},
	}
}
