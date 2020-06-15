package page

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type uploadPageView struct {
	Error       string
	Folder      string
	MaxFileSize string
}

func ajaxErr(err string) razlink.PageView {
	return func(w http.ResponseWriter) {
		http.Error(w, err, http.StatusInternalServerError)
	}
}

func uploadPageHandler(api *razbox.API, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	dir := path.Clean(r.URL.Path[8:]) // skip /upload/
	ajax := r.URL.Query().Get("u") == "ajax"

	token := api.AccessTokenFromRequest(r)
	flags, err := api.GetFolderFlags(token, dir)
	if err != nil {
		return HandleError(r, err)
	}

	if !flags.EditMode {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	title := "Upload file to " + dir
	v := &uploadPageView{
		Folder:      dir,
		MaxFileSize: fmt.Sprintf("%dMB", flags.MaxUploadSizeMB),
	}
	handleError := func(err error) razlink.PageView {
		if ajax {
			return ajaxErr(err.Error())
		}
		v.Error = err.Error()
		return view(v, &title)
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

		return razlink.RedirectView(r, "/x/"+dir)
	}

	return view(v, &title)
}

// Upload returns a razlink.Page that handles file uploads
func Upload(api *razbox.API) *razlink.Page {
	return &razlink.Page{
		Path:            "/upload/",
		ContentTemplate: GetContentTemplate("upload"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return uploadPageHandler(api, r, view)
		},
	}
}
