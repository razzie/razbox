package page

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mholt/archiver"
	"github.com/nwaples/rardecode"
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type archiveEntry struct {
	Name     string
	Size     int64
	Modified time.Time
}

type archivePageView struct {
	Filename string          `json:"filename,omitempty"`
	Folder   string          `json:"folder,omitempty"`
	URI      string          `json:"uri,omitempty"`
	Entries  []*archiveEntry `json:"entries,omitempty"`
}

func (v *archivePageView) addEntry(f archiver.File) error {
	if f.IsDir() {
		return nil
	}

	v.Entries = append(v.Entries, &archiveEntry{
		Name:     archiveGetFilename(f),
		Size:     f.Size(),
		Modified: f.ModTime(),
	})
	return nil
}

func archiveGetFilename(f archiver.File) string {
	switch h := f.Header.(type) {
	case zip.FileHeader:
		return h.Name
	case *tar.Header:
		return h.Name
	case *rardecode.FileHeader:
		return h.Name
	default:
		return f.Name()
	}
}

func archiveDownloadFile(archive, filename string, walker archiver.Walker) *beepboop.View {
	return beepboop.HandlerView(nil, func(w http.ResponseWriter, r *http.Request) {
		walker.Walk(archive, func(f archiver.File) error {
			if archiveGetFilename(f) == filename {
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", f.Name()))
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Length", strconv.FormatInt(f.Size(), 10))
				io.Copy(w, f)
				return archiver.ErrStopWalk
			}
			return nil
		})
	})
}

func archivePageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	internalFilename, err := api.GetInternalFilename(pr.Session(), filename)
	if err != nil {
		return HandleError(r, err)
	}

	iface, err := archiver.ByExtension(filename)
	if err != nil {
		pr.Log("archive error:", err)
		return pr.RedirectView("/x/" + filename)
	}

	w, ok := iface.(archiver.Walker)
	if !ok {
		pr.Log("archive error: walk not supported for format:", iface)
		return pr.RedirectView("/x/" + filename)
	}

	download := r.URL.Query().Get("download")
	if len(download) > 0 {
		return archiveDownloadFile(internalFilename, download, w)
	}

	pr.Title = filename
	v := &archivePageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
		URI:      r.RequestURI,
	}
	w.Walk(internalFilename, v.addEntry)
	return pr.Respond(v)
}

// Archive returns a beepboop.Page that visualizes archive files
func Archive(api *razbox.API) *beepboop.Page {
	return &beepboop.Page{
		Path:            "/archive/",
		ContentTemplate: GetContentTemplate("archive"),
		Handler: func(pr *beepboop.PageRequest) *beepboop.View {
			return archivePageHandler(api, pr)
		},
	}
}
