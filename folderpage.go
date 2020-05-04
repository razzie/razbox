package razbox

import (
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/razzie/razlink"
)

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
	{{range .}}
	<tr>
		<td><a href="/x/{{.RelPath}}">{{.Name}}</a></td>
		<td>{{.MIME}}</td>
		<td>{{range .Tags}} {{.}}{{end}}</td>
		<td>{{.Size}}</td>
		<td>{{.Uploaded}}</td>
	</tr>
	{{end}}
</table>
`

func isFolder(dir string) bool {
	fi, err := os.Stat(path.Join(Root, dir))
	if err != nil {
		return false
	}

	return fi.IsDir()
}

func viewFile(r *http.Request) razlink.PageView {
	return razlink.ErrorView(r, "Not implemented", http.StatusInternalServerError)
}

func pageHandler(r *http.Request, view razlink.ViewFunc) razlink.PageView {
	uri := r.URL.Path[3:] // skip /x/

	if !isFolder(uri) {
		return viewFile(r)
	}

	folder, err := GetFolder(filepath.Dir(uri))
	if err != nil {
		return razlink.ErrorView(r, err.Error(), http.StatusInternalServerError)
	}

	err = folder.EnsureReadAccess(r)
	if err != nil {
		return razlink.ErrorView(r, err.Error(), http.StatusInternalServerError)
	}

	subfolders := folder.GetSubfolders()
	files := folder.GetFiles()
	entries := make([]*folderEntry, 0, len(subfolders)+len(files))
	for _, subfolder := range subfolders {
		entry := &folderEntry{
			Name:    subfolder,
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

	return view(entries, &uri)
}

// FolderPage ...
var FolderPage = razlink.Page{
	Path:            "/x/",
	ContentTemplate: folderPageT,
	Handler:         pageHandler,
}
