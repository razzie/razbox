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
<script>
function _(el) {
	return document.getElementById(el);
}
function uploadFile() {
	_("submit").disabled = true;
	var form = _("upload_form")
	var formdata = new FormData();
	formdata.append("file", form.elements["file"].files[0]);
	formdata.append("filename", form.elements["filename"].value);
	formdata.append("tags", form.elements["tags"].value);
	formdata.append("overwrite", form.elements["overwrite"].checked ? "overwrite" : "");
	var ajax = new XMLHttpRequest();
	ajax.upload.addEventListener("progress", progressHandler, false);
	ajax.addEventListener("load", completeHandler, false);
	ajax.addEventListener("error", errorHandler, false);
	ajax.addEventListener("abort", abortHandler, false);
	ajax.open("POST", "/upload/{{.Folder}}?u=ajax");
	ajax.send(formdata);
}
function progressHandler(event) {
	_("loaded_n_total").innerHTML = "Uploaded " + event.loaded + " bytes of " + event.total;
	var percent = (event.loaded / event.total) * 100;
	_("progress").value = Math.round(percent);
	_("status").innerHTML = Math.round(percent) + "% uploaded... please wait";
}
function completeHandler(event) {
	_("submit").disabled = false;
	if (event.target.statusText === 'OK')
		window.location.replace("/x/{{.Folder}}");
	else
		_("error").innerHTML = event.target.responseText;
}
function errorHandler(event) {
	_("status").innerHTML = "Upload Failed";
}
function abortHandler(event) {
	_("status").innerHTML = "Upload Aborted";
}
</script>
<strong style="color: red" id="error">{{.Error}}</strong><br /><br />
<div style="text-align: right; min-width: 400px">
	<small>max file size: <strong>{{.MaxFileSize}}</strong></small>
</div>
<form
	enctype="multipart/form-data"
	action="/upload/{{.Folder}}"
	method="post"
	onsubmit="uploadFile(); return false;"
	id="upload_form"
>
	<input type="file" name="file" /><br />
	<input type="text" name="filename" placeholder="Filename (optional)" /><br />
	<input type="text" name="tags" placeholder="Tags (space separated)" /><br />
	<input type="checkbox" name="overwrite" value="overwrite" />
	<label for="overwrite">Overwrite if exists</label><br />
	<button id="submit">&#8686; Upload</button>
</form>
<div style="float: right">
	<a href="/x/{{.Folder}}">Go back &#10548;</a>
</div>
<div style="clear: both">
	<progress id="progress" value="0" max="100" style="width: 100%"></progress>
	<p id="status"></p>
	<p id="loaded_n_total"></p>
</div>
`

func ajaxErr(err string) razlink.PageView {
	return func(w http.ResponseWriter) {
		http.Error(w, err, http.StatusInternalServerError)
	}
}

func uploadPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[8:] // skip /upload/
	uri = razbox.RemoveTrailingSlash(uri)
	ajax := r.URL.Query().Get("u") == "ajax"

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

	title := "Upload file to " + uri
	v := &uploadPageView{
		Folder:      uri,
		MaxFileSize: fmt.Sprintf("%dMB", folder.MaxFileSizeMB),
	}

	if r.Method == "POST" {
		r.ParseMultipartForm(folder.MaxFileSizeMB << 20)
		data, handler, err := r.FormFile("file")
		if err != nil {
			if ajax {
				return ajaxErr(err.Error())
			}
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

		overwrite := r.FormValue("overwrite") == "overwrite"
		file := &razbox.File{
			Name:     filename,
			RelPath:  path.Join(uri, razbox.FilenameToUUID(filename)),
			Tags:     strings.Fields(r.FormValue("tags")),
			MIME:     mime.String(),
			Size:     handler.Size,
			Uploaded: time.Now(),
		}
		err = file.Create(data, overwrite)
		if err != nil {
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
