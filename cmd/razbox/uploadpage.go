package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type uploadPageView struct {
	Error       string
	Folder      string
	MaxFileSize string
}

var uploadPageT = `
{{if .Error}}
<strong style="color: red">{{.Error}}</strong><br /><br />
{{end}}
<form enctype="multipart/form-data" action="/upload/{{.Folder}}" method="post">
	<input type="file" name="file" /> max file size: <strong>{{.MaxFileSize}}</strong><br />
	<input type="text" name="filename" placeholder="Filename (optional)" /><br />
	<input type="text" name="tags" placeholder="Tags (space separated)" /><br />
	<button>Upload &#10548;</button>
</form>
`

func uploadPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[8:] // skip /upload/

	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	var folder *razbox.Folder
	var err error

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		folder, err = razbox.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.Path))
	}

	title := "Upload file to " + uri
	v := &uploadPageView{
		Folder:      uri,
		MaxFileSize: fmt.Sprintf("%dMB", folder.MaxFileSizeMB),
	}

	if r.Method == "POST" {
		r.ParseMultipartForm(folder.MaxFileSizeMB << 20)
		data, handler, err := r.FormFile("file")
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}
		defer data.Close()

		filename := govalidator.SafeFileName(r.FormValue("filename"))
		if len(filename) == 0 || filename == "." {
			filename = govalidator.SafeFileName(handler.Filename)
			if len(filename) == 0 || filename == "." {
				filename = razbox.Salt()
			}
		}

		mime, _ := mimetype.DetectReader(data)
		data.Seek(0, io.SeekStart)

		file := &razbox.File{
			Name:     filename,
			RelPath:  path.Join(uri, razbox.FilenameToUUID(filename)),
			Tags:     strings.Fields(r.FormValue("tags")),
			MIME:     mime.String(),
			Size:     handler.Size,
			Uploaded: time.Now(),
		}
		err = file.Create(data)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}

		if db != nil {
			folder.CachedFiles = nil
			db.CacheFolder(folder)
		}
		return razlink.RedirectView(r, "/x/"+uri)
	}

	return view(v, &title)
}

// GetUploadPage returns a razlink.Page that handles file uploads
func GetUploadPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/upload/",
		ContentTemplate: uploadPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return uploadPageHandler(db, r, view)
		},
	}
}
