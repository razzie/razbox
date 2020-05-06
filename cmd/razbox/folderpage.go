package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

type folderPageView struct {
	Header         template.HTML
	Footer         template.HTML
	Folder         string
	Entries        []*folderEntry
	EditMode       bool
	ControlButtons bool
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
}

var folderPageT = `
{{.Header}}
<table>
	<tr>
		<td>Name</td>
		<td>Type</td>
		<td>Tags</td>
		<td>Size</td>
		<td>Uploaded</td>
	</tr>
	{{$Folder := .Folder}}
	{{range .Entries}}
		<tr>
			<td>
				{{.Prefix}}<a href="/x/{{.RelPath}}">{{.Name}}</a>
				{{if .EditMode}}
					[<a href="/edit/{{.RelPath}}">edit</a>]
				{{end}}
			</td>
			<td>{{.MIME}}</td>
			<td>
				{{range .Tags}}
					<a href="/search/{{$Folder}}/{{.}}">{{.}}</a>
				{{end}}
			</td>
			<td>{{.Size}}</td>
			<td>{{.Uploaded}}</td>
		</tr>
	{{end}}
	{{if not .Entries}}
		<tr><td colspan="5">No entries</td></tr>
	{{end}}
</table>
{{if .ControlButtons}}
	<div style="text-align: center">
	{{if .EditMode}}
		<form method="get">
			<button formaction="/upload/{{.Folder}}">Upload file</button>
		</form>
	{{else}}
		<form method="get">
			<button formaction="/write-auth/{{.Folder}}">Edit mode</button>
		</form>
	{{end}}
	</div>
{{end}}
{{.Footer}}
`

func folderPageHandler(db *razbox.DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[3:] // skip /x/

	if !razbox.IsFolder(uri) {
		return viewFile(db, r)
	}

	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	var folder *razbox.Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		cached = false
		folder, err = razbox.GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.RedirectView(r, fmt.Sprintf("/read-auth/%s?r=%s", uri, r.URL.Path))
	}

	editMode := folder.EnsureWriteAccess(r) == nil
	subfolders := folder.GetSubfolders()
	files := folder.GetFiles()
	entries := make([]*folderEntry, 0, 1+len(subfolders)+len(files))

	if len(uri) > 0 {
		entry := &folderEntry{
			Prefix:  "&#128194;",
			Name:    "..",
			RelPath: path.Join(uri, ".."),
		}
		entries = append(entries, entry)
	}

	for _, subfolder := range subfolders {
		entry := &folderEntry{
			Prefix:  "&#128194;",
			Name:    subfolder,
			RelPath: path.Join(uri, subfolder),
		}
		entries = append(entries, entry)
	}

	for _, file := range files {
		entry := &folderEntry{
			Prefix:   razbox.MIMEtoSymbol(file.MIME),
			Name:     file.Name,
			RelPath:  path.Join(uri, file.Name),
			MIME:     file.MIME,
			Tags:     file.Tags,
			Size:     file.Size,
			Uploaded: file.Uploaded.Format("Mon, 02 Jan 2006 15:04:05 MST"),
			EditMode: editMode,
		}
		entries = append(entries, entry)
	}

	if db != nil && !cached {
		db.CacheFolder(folder)
	}

	v := &folderPageView{
		Folder:         uri,
		Entries:        entries,
		EditMode:       editMode,
		ControlButtons: true,
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
