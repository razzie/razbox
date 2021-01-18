package internal

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/kjk/lzmadec"
	"github.com/mholt/archiver"
)

//var p7zipOK = exec.Command("7z").Run() == nil

// GetArchiveWalker ...
func GetArchiveWalker(filename, mime string) (archiver.Walker, error) {
	switch mime {
	case "application/x-rar-compressed":
		return archiver.NewRar(), nil
	case "application/x-tar":
		return archiver.NewTar(), nil
	case "application/zip":
		return archiver.NewZip(), nil
	case "application/x-7z-compressed":
		return &p7zipWalker{}, nil
	}
	iface, err := archiver.ByExtension(filename)
	if err != nil {
		return nil, err
	}
	walker, ok := iface.(archiver.Walker)
	if !ok {
		return nil, &ErrUnsupportedFileFormat{
			MIME: fmt.Sprintf("%s (%s)", mime, path.Ext(filename)),
		}
	}
	return walker, nil
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
