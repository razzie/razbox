package page

import (
	"archive/tar"
	"archive/zip"
	"path"
	"path/filepath"
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
	Entries  []*archiveEntry `json:"entries,omitempty"`
}

func (v *archivePageView) addEntry(f archiver.File) error {
	entry := &archiveEntry{
		Name:     f.Name(),
		Size:     f.Size(),
		Modified: f.ModTime(),
	}

	switch h := f.Header.(type) {
	case zip.FileHeader:
		entry.Name = h.Name
	case *tar.Header:
		entry.Name = h.Name
	case *rardecode.FileHeader:
		entry.Name = h.Name
	}

	v.Entries = append(v.Entries, entry)
	return nil
}

func archivePageHandler(api *razbox.API, pr *beepboop.PageRequest) *beepboop.View {
	r := pr.Request
	filename := path.Clean(pr.RelPath)
	dir := path.Dir(filename)
	token := beepboop.NewAccessTokenFromRequest(pr)
	internalFilename, err := api.GetInternalFilename(token, filename)
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

	pr.Title = filename
	v := &archivePageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
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
