package page

import (
	"archive/tar"
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kjk/lzmadec"
	"github.com/mholt/archiver"
	"github.com/nwaples/rardecode"
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox"
)

var p7zipOK = exec.Command("7z").Run() == nil

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

func archiveDownloadFile(r *http.Request, archive, filename string, walker archiver.Walker) *beepboop.View {
	return beepboop.HandlerView(r, func(w http.ResponseWriter, r *http.Request) {
		walker.Walk(archive, func(f archiver.File) error {
			if err := r.Context().Err(); err != nil {
				return err
			}
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

	_, all := r.URL.Query()["all"]

	var iface interface{}
	if p7zipOK && strings.HasSuffix(filename, ".7z") {
		iface = &p7zipWalker{}
		all = true
	}

	if iface == nil {
		iface, err = archiver.ByExtension(filename)
		if err != nil {
			pr.Log("archive error:", err)
			return pr.RedirectView("/x/" + filename + "?download")
		}
	}

	w, ok := iface.(archiver.Walker)
	if !ok {
		pr.Log("archive error: walk not supported for format:", iface)
		return pr.RedirectView("/x/" + filename + "?download")
	}

	download := r.URL.Query().Get("download")
	if len(download) > 0 {
		return archiveDownloadFile(r, internalFilename, download, w)
	}

	pr.Title = filename
	v := &archivePageView{
		Filename: filepath.Base(filename),
		Folder:   dir,
		URI:      r.RequestURI,
	}
	count := 0
	walker := func(f archiver.File) error {
		if err := r.Context().Err(); err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if count++; !all && count > 19 {
			v.Entries = append(v.Entries, nil)
			return archiver.ErrStopWalk
		}
		v.Entries = append(v.Entries, &archiveEntry{
			Name:     archiveGetFilename(f),
			Size:     f.Size(),
			Modified: f.ModTime(),
		})
		return nil
	}
	w.Walk(internalFilename, walker)
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

type p7zipWalker struct{}

func (p7zipWalker) Walk(archiveFilename string, walkFn archiver.WalkFunc) error {
	archive, err := lzmadec.NewArchive(archiveFilename)
	if err != nil {
		return err
	}
	for _, entry := range archive.Entries {
		f := &p7zipFile{
			archive: archive,
			entry:   entry,
		}
		err := walkFn(archiver.File{
			FileInfo:   f,
			ReadCloser: f,
		})
		if err != nil {
			if err == archiver.ErrStopWalk {
				return nil
			}
			return err
		}
	}
	return nil
}

type p7zipFile struct {
	archive *lzmadec.Archive
	entry   lzmadec.Entry
	rc      io.ReadCloser
}

func (z *p7zipFile) Name() string {
	return z.entry.Path
}

func (z *p7zipFile) Size() int64 {
	return z.entry.Size
}

func (z *p7zipFile) Mode() os.FileMode {
	return 0
}

func (z *p7zipFile) ModTime() (t time.Time) {
	return z.entry.Modified
}

func (z *p7zipFile) IsDir() bool {
	return z.entry.Size == 0
}

func (z *p7zipFile) Sys() interface{} {
	return z.entry
}

func (z *p7zipFile) Read(p []byte) (n int, err error) {
	if z.rc == nil {
		rc, err := z.archive.GetFileReader(z.entry.Path)
		if err != nil {
			return 0, err
		}
		z.rc = rc
	}
	return z.rc.Read(p)
}

func (z *p7zipFile) Close() error {
	if z.rc != nil {
		return z.rc.Close()
	}
	return nil
}
