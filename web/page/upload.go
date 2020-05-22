package page

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
	"github.com/razzie/razbox/lib"
	"github.com/razzie/razbox/web/page/internal"
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

func uploadPageHandler(db *lib.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[8:] // skip /upload/
	uri = lib.RemoveTrailingSlash(uri)
	ajax := r.URL.Query().Get("u") == "ajax"

	var folder *lib.Folder
	var err error

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		folder, err = lib.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	err = folder.EnsureWriteAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	title := "Upload file to " + uri
	v := &uploadPageView{
		Folder:      uri,
		MaxFileSize: fmt.Sprintf("%dMB", folder.MaxFileSizeMB),
	}

	if r.Method == "POST" {
		limit := folder.MaxFileSizeMB << 20
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

		if handler.Size > limit {
			if ajax {
				return ajaxErr("limit exceeded")
			}
			v.Error = "limit exceeded"
			return view(v, &title)
		}

		filename := govalidator.SafeFileName(r.FormValue("filename"))
		if len(filename) == 0 || filename == "." {
			filename = govalidator.SafeFileName(handler.Filename)
			if len(filename) == 0 || filename == "." {
				filename = lib.Salt()
			}
		}

		mime, _ := mimetype.DetectReader(data)
		data.Seek(0, io.SeekStart)

		overwrite := r.FormValue("overwrite") == "overwrite"
		file := &lib.File{
			Name:     filename,
			RelPath:  path.Join(uri, lib.FilenameToUUID(filename)),
			Tags:     strings.Fields(r.FormValue("tags")),
			MIME:     mime.String(),
			Size:     handler.Size,
			Uploaded: time.Now(),
		}
		err = file.Create(data, overwrite)
		if err != nil {
			file.Delete()

			if ajax {
				return ajaxErr(err.Error())
			}
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

// Upload returns a razlink.Page that handles file uploads
func Upload(db *lib.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/upload/",
		ContentTemplate: internal.GetContentTemplate("upload"),
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return uploadPageHandler(db, r, view)
		},
	}
}
