package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type folderPageView struct {
	Folder   string
	Search   string
	Entries  []*folderEntry
	EditMode bool
	Gallery  bool
	Redirect string
}

type folderEntry struct {
	Prefix   template.HTML
	Name     string
	RelPath  string
	MIME     string
	Tags     []string
	Size     string
	Uploaded string
	EditMode bool
	IsImage  bool
}

func newSubfolderEntry(uri, subfolder string) *folderEntry {
	return &folderEntry{
		Prefix:  "&#128194;",
		Name:    subfolder,
		RelPath: path.Join(uri, subfolder),
	}
}

func newFileEntry(uri string, file *razbox.File) *folderEntry {
	return &folderEntry{
		Prefix:   razbox.MIMEtoSymbol(file.MIME),
		Name:     file.Name,
		RelPath:  path.Join(uri, file.Name),
		MIME:     file.MIME,
		Tags:     file.Tags,
		Size:     razbox.ByteCountSI(file.Size),
		Uploaded: file.Uploaded.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		IsImage:  strings.HasPrefix(file.MIME, "image/"),
	}
}

var folderPageT = `
{{if .Search}}
	<div>
		<span style="float: left">&#128269; Search results for tag: <strong>{{.Search}}</strong></span>
		<span style="float: right">&#128194; <a href="/x/{{.Folder}}">View folder content</a></span>
	</div>
	<div style="clear: both; margin-bottom: 1rem"></div>
{{end}}
<table>
	<tr>
		<td>Name</td>
		<td>Type</td>
		<td>Tags</td>
		<td>Size</td>
		<td>Uploaded</td>
		<td></td>
	</tr>
	{{$Folder := .Folder}}
	{{$Redirect := .Redirect}}
	{{range .Entries}}
		<tr>
			<td>{{.Prefix}}<a href="/x/{{.RelPath}}">{{.Name}}</a></td>
			<td>{{.MIME}}</td>
			<td>
				{{range .Tags}}
					<a href="/search/{{$Folder}}/{{.}}">{{.}}</a>
				{{end}}
			</td>
			<td>{{.Size}}</td>
			<td>{{.Uploaded}}</td>
			<td>
				{{if .EditMode}}
					<a href="/edit/{{.RelPath}}/?r={{$Redirect}}">&#9998;</a>
					<a href="/delete/{{.RelPath}}/?r={{$Redirect}}" onclick="return confirm('Are you sure?')">&#10008;</a>
				{{end}}
			</td>
		</tr>
	{{end}}
	{{if not .Entries}}
		<tr><td colspan="6">No entries</td></tr>
	{{end}}
</table>
<div style="text-align: center">
	<form method="get">
		{{if .Search}}
			<input type="hidden" name="tag" value="{{.Search}}" />
			<button formaction="/gallery/{{.Folder}}/"{{if not .Gallery}} disabled{{end}}>Gallery</button>
		{{else}}
			{{if .EditMode}}
				<button formaction="/upload/{{.Folder}}">Upload file</button>
				<button formaction="/change-password/{{.Folder}}">Change password</button>
			{{else}}
				<button formaction="/write-auth/{{.Folder}}">Edit mode</button>
			{{end}}
			<button formaction="/gallery/{{.Folder}}"{{if not .Gallery}} disabled{{end}}>Gallery</button>
		{{end}}
	</form>
</div>
`

func folderPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[3:] // skip /x/
	uri = razbox.RemoveTrailingSlash(uri)

	var filename string
	dir := uri
	if !razbox.IsFolder(uri) {
		dir = path.Dir(uri)
		filename = filepath.Base(uri)
	}

	var folder *razbox.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(dir)
	}
	if folder == nil {
		cached = false
		folder, err = razbox.GetFolder(dir)
		if err != nil {
			log.Println(dir, "error:", err.Error())
			return razlink.ErrorView(r, "Folder not found", http.StatusNotFound)
		}
	}

	if db != nil && !cached {
		defer db.CacheFolder(folder)
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", dir, r.URL.RequestURI()))
	}

	if len(filename) > 0 {
		file, err := folder.GetFile(filepath.Base(filename))
		if err != nil {
			log.Println(filename, "error:", err.Error())
			return razlink.ErrorView(r, "File not found", http.StatusNotFound)
		}

		data, err := file.Open()
		if err != nil {
			log.Println(filename, "error:", err.Error())
			return razlink.ErrorView(r, "Could not open file", http.StatusInternalServerError)
		}

		return func(w http.ResponseWriter) {
			defer data.Close()
			w.Header().Set("Content-Type", file.MIME)
			_, err := io.Copy(w, data)
			if err != nil {
				log.Println(filename, "error:", err.Error())
			}
		}
	}

	editMode := folder.EnsureWriteAccess(r) == nil
	subfolders := folder.GetSubfolders()
	files := folder.GetFiles()
	entries := make([]*folderEntry, 0, 1+len(subfolders)+len(files))

	if len(uri) > 0 {
		entry := newSubfolderEntry(uri, "..")
		entries = append(entries, entry)
	}

	for _, subfolder := range subfolders {
		entry := newSubfolderEntry(uri, subfolder)
		entries = append(entries, entry)
	}

	var enableGallery bool
	for _, file := range files {
		entry := newFileEntry(uri, file)
		entry.EditMode = editMode
		if !enableGallery && entry.IsImage {
			enableGallery = true
		}
		entries = append(entries, entry)
	}

	v := &folderPageView{
		Folder:   uri,
		Entries:  entries,
		EditMode: editMode,
		Gallery:  enableGallery,
		Redirect: r.URL.Path,
	}
	return view(v, &uri)
}

// GetFolderPage returns a razlink.Page that handles folders
func GetFolderPage(db *razbox.DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/x/",
		ContentTemplate: folderPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return folderPageHandler(db, r, view)
		},
	}
}
