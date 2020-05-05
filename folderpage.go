package razbox

import (
	"fmt"
	"log"
	"net/http"
	"path"

	"github.com/razzie/razlink"
)

type folderPageView struct {
	Folder  string
	Entries []*folderEntry
}

type folderEntry struct {
	Name     string
	RelPath  string
	MIME     string
	Tags     []string
	Size     string
	Uploaded string
}

var folderPageT = `
<table>
	<tr>
		<td>Name</td>
		<td>MIME type</td>
		<td>Tags</td>
		<td>Size</td>
		<td>Uploaded</td>
	</tr>
	{{$Folder := .Folder}}
	{{range .Entries}}
	<tr>
		<td><a href="/x/{{.RelPath}}">{{.Name}}</a></td>
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
</table>
`

func folderPageHandler(db *DB, r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[3:] // skip /x/

	if !IsFolder(uri) {
		return viewFile(db, r)
	}

	if len(uri) > 0 && uri[len(uri)-1] == '/' {
		uri = uri[:len(uri)-1]
	}

	var folder *Folder
	var err error
	cached := true

	if db != nil {
		folder, _ = db.GetCachedFolder(uri)
	}
	if folder == nil {
		cached = false
		folder, err = GetFolder(uri)
		if err != nil {
			log.Println(uri, "error:", err.Error())
			return razlink.ErrorView(r, "Not found", http.StatusNotFound)
		}
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		log.Println(uri, "error:", err.Error())
		return razlink.ErrorView(r, "Unauthorized", http.StatusUnauthorized)
	}

	subfolders := folder.GetSubfolders()
	files := folder.GetFiles()
	entries := make([]*folderEntry, 0, 1+len(subfolders)+len(files))

	if len(uri) > 0 {
		entry := &folderEntry{
			Name:    "<..>",
			RelPath: path.Join(uri, ".."),
		}
		entries = append(entries, entry)
	}

	for _, subfolder := range subfolders {
		entry := &folderEntry{
			Name:    fmt.Sprintf("<%s>", subfolder),
			RelPath: path.Join(uri, subfolder),
		}
		entries = append(entries, entry)
	}

	for _, file := range files {
		entry := &folderEntry{
			Name:     file.Name,
			RelPath:  path.Join(uri, file.Name),
			MIME:     file.MIME,
			Tags:     file.Tags,
			Size:     file.Size,
			Uploaded: file.Uploaded,
		}
		entries = append(entries, entry)
	}

	if db != nil && !cached {
		db.CacheFolder(folder)
	}

	v := &folderPageView{
		Folder:  uri,
		Entries: entries,
	}
	return view(v, &uri)
}

// GetFolderPage returns a razlink.Page that handles folders
func GetFolderPage(db *DB) *razlink.Page {
	return &razlink.Page{
		Path:            "/x/",
		ContentTemplate: folderPageT,
		Handler: func(r *http.Request, view razlink.ViewFunc) razlink.PageView {
			return folderPageHandler(db, r, view)
		},
	}
}
