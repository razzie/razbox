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
	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type downloadPageView struct {
	Error       string
	Folder      string
	MaxFileSize string
}

var downloadPageT = `
{{if .Error}}
<strong style="color: red">{{.Error}}</strong><br /><br />
{{end}}
<div style="text-align: right; min-width: 400px">
	<small>max file size: <strong>{{.MaxFileSize}}</strong></small>
</div>
<form method="post">
	<input type="url" name="url" placeholder="URL" style="width: 400px" /><br />
	<input type="text" name="filename" placeholder="Filename (optional)" /><br />
	<input type="text" name="tags" placeholder="Tags (space separated)" /><br />
	<button id="submit">&#8681; Download</button>
</form>
<div style="float: right">
	<a href="/x/{{.Folder}}">Go back &#10548;</a>
</div>
`

type limitedReader struct {
	r io.Reader
	n int64
}

func (r *limitedReader) Read(p []byte) (n int, err error) {
	if int64(len(p)) > r.n {
		p = p[:r.n]
	}
	n, err = r.r.Read(p)
	r.n -= int64(n)
	if r.n == 0 && err == nil {
		err = fmt.Errorf("limit exceeded")
	}
	return
}

func downloadPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[20:] // skip /download-to-folder/
	uri = razbox.RemoveTrailingSlash(uri)

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
		return razlink.RedirectView(r, fmt.Sprintf("/write-auth/%s?r=%s", uri, r.URL.RequestURI()))
	}

	title := "Download file to " + uri
	v := &uploadPageView{
		Folder:      uri,
		MaxFileSize: fmt.Sprintf("%dMB", folder.MaxFileSizeMB),
	}

	if r.Method == "POST" {
		r.ParseForm()
		url := r.FormValue("url")
		filename := govalidator.SafeFileName(r.FormValue("filename"))
		if len(filename) == 0 || filename == "." {
			filename = govalidator.SafeFileName(path.Base(url))
			if len(filename) == 0 || filename == "." {
				filename = razbox.Salt()
			}
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}
		resp, err := http.DefaultClient.Do(req.WithContext(r.Context()))
		if err != nil {
			v.Error = err.Error()
			return view(v, &title)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			v.Error = "bad response status code: " + http.StatusText(resp.StatusCode)
			return view(v, &title)
		}

		limit := folder.MaxFileSizeMB << 20
		data := &limitedReader{
			r: resp.Body,
			n: limit,
		}

		file := &razbox.File{
			Name:     filename,
			RelPath:  path.Join(uri, razbox.FilenameToUUID(filename)),
			Tags:     strings.Fields(r.FormValue("tags")),
			Uploaded: time.Now(),
		}
		err = file.Create(data, false)
		if err != nil {
			file.Delete()

			v.Error = err.Error()
			return view(v, &title)
		}
		file.FixMimeAndSize()

		if db != nil {
			folder.CachedFiles = nil
			db.CacheFolder(folder)
		}
		return razlink.RedirectView(r, "/x/"+uri)
	}

	return view(v, &title)
}

// GetDownloadPage returns a razlink.Page that handles file downloads from an URL to a folder
func GetDownloadPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/download-to-folder/",
		ContentTemplate: downloadPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return downloadPageHandler(db, r, view)
		},
	}
}
