package page

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mholt/archiver"
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

type archiveEntry struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

type archivePageView struct {
	Filename string          `json:"filename,omitempty"`
	Folder   string          `json:"folder,omitempty"`
	URI      string          `json:"uri,omitempty"`
	Entries  []*archiveEntry `json:"entries,omitempty"`
}

func archiveDownloadFile(r *http.Request, archive razbox.ArchiveWalker, filename string) *beepboop.View {
	return beepboop.HandlerView(r, func(w http.ResponseWriter, r *http.Request) {
		found := false
		err := archive.Walk(func(f razbox.ArchiveFile) error {
			if f.Name() == filename {
				found = true
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", f.Name()))
				w.Header().Set("Content-Type", "application/octet-stream")
				w.Header().Set("Content-Length", strconv.FormatInt(f.Size(), 10))
				if _, err := io.Copy(w, f); err != nil {
					return err
				}
				return archiver.ErrStopWalk
			}
			return nil
		})
		if err != nil {
			beepboop.ErrorView(r, "Archive error", http.StatusInternalServerError).Render(w)
		}
		if !found {
			beepboop.ErrorView(r, "Not found", http.StatusNotFound).Render(w)
		}
	})
}

func archivePageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	archive, err := api.GetArchiveWalker(pr.Session(), filename)
	if err != nil {
		return HandleError(r, err)
	}

	_, all := r.URL.Query()["all"]
	download := r.URL.Query().Get("download")
	if len(download) > 0 {
		return archiveDownloadFile(r, archive, download)
	}

	pr.Title = filename
	v := &archivePageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
		URI:      r.RequestURI,
	}
	count := 0
	archive.Walk(func(f razbox.ArchiveFile) error {
		if count++; !all && count > 19 {
			v.Entries = append(v.Entries, nil)
			return archiver.ErrStopWalk
		}
		v.Entries = append(v.Entries, &archiveEntry{
			Name:     f.Name(),
			Size:     f.Size(),
			Modified: f.ModTime(),
		})
		return nil
	})
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
