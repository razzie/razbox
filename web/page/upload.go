package page

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razbox/internal"
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
	uri := r.URL.Path[8:] // skip /upload/
	uri = internal.RemoveTrailingSlash(uri)
	ajax := r.URL.Query().Get("u") == "ajax"

	token := api.AccessTokenFromCookies(r.Cookies())
	flags, err := api.GetFolderFlags(token, uri)
	if err != nil {
		return HandleError(r, err)
	}

	if !flags.EditMode {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	title := "Upload file to " + uri
	v := &uploadPageView{
		Folder:      uri,
		MaxFileSize: fmt.Sprintf("%dMB", flags.MaxFileSizeMB),
	}

	if r.Method == "POST" {
		limit := flags.MaxFileSizeMB << 20
		r.ParseMultipartForm(limit)
		data, handler, err := r.FormFile("file")
		if err != nil {
			if ajax {
				return ajaxErr(err.Error())
			}
			v.Error = err.Error()
			return view(v, &title)
		}
		defer data.Close()

		o := &razbox.UploadFileOptions{
			Folder:    uri,
			File:      data,
			Header:    handler,
			Filename:  r.FormValue("filename"),
			Tags:      strings.Fields(r.FormValue("tags")),
			Overwrite: r.FormValue("overwrite") == "overwrite",
		}
		err = api.UploadFile(token, o)
		if err != nil {
			if ajax {
				return ajaxErr(err.Error())
			}
			v.Error = err.Error()
			return view(v, &title)
		}

		return razlink.RedirectView(r, "/x/"+uri)
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