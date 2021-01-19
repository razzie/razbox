package razbox

import (
	"archive/tar"
	"archive/zip"
	"context"
	"io"
	"path"
	"path/filepath"
	"time"

	"github.com/mholt/archiver"
	"github.com/nwaples/rardecode"
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox/internal"
)

// ArchiveFile ...
type ArchiveFile interface {
	io.Reader
	Name() string
	Size() int64
	ModTime() time.Time
}

// ArchiveWalker ...
type ArchiveWalker interface {
	Walk(func(ArchiveFile) error) error
}

// GetArchiveWalker ...
func (api *API) GetArchiveWalker(sess *beepboop.Session, filePath string) (ArchiveWalker, error) {
	filePath = path.Clean(filePath)
	dir := path.Dir(filePath)
	folder, unlock, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}
	defer unlock()

	hasViewAccess := folder.EnsureReadAccess(sess) == nil

	basename := filepath.Base(filePath)
	file, err := folder.GetFile(basename)
	if err != nil {
		if !hasViewAccess {
			return nil, &ErrNoReadAccess{Folder: dir}
		}
		return nil, &ErrNotFound{}
	}

	if !hasViewAccess && !file.Public {
		return nil, &ErrNoReadAccess{Folder: dir}
	}

	walker, err := internal.GetArchiveWalker(file.Name, file.MIME)
	if err != nil {
		return nil, err
	}

	return &archiveWalker{
		ctx:     sess.Context(),
		archive: file.GetInternalFilename(),
		walker:  walker,
	}, nil
}

type archiveFile struct {
	f archiver.File
}

func (af *archiveFile) Name() string {
	switch h := af.f.Header.(type) {
	case zip.FileHeader:
		return h.Name
	case *tar.Header:
		return h.Name
	case *rardecode.FileHeader:
		return h.Name
	default:
		return af.f.Name()
	}
}

func (af *archiveFile) Size() int64 {
	return af.f.Size()
}

func (af *archiveFile) ModTime() time.Time {
	return af.f.ModTime()
}

func (af *archiveFile) Read(p []byte) (int, error) {
	return af.f.Read(p)
}

type archiveWalker struct {
	ctx     context.Context
	archive string
	walker  archiver.Walker
}

func (aw *archiveWalker) Walk(walkFn func(ArchiveFile) error) error {
	return aw.walker.Walk(aw.archive, func(f archiver.File) error {
		if err := aw.ctx.Err(); err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		return walkFn(&archiveFile{f: f})
	})
}
