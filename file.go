package razbox

import (
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/razzie/razbox/internal"
)

// FileReader ...
type FileReader struct {
	r    internal.FileReader
	Name string
	MIME string
}

// Read implements io.Reader
func (r FileReader) Read(p []byte) (int, error) {
	return r.r.Read(p)
}

// Seek implements io.Seeker
func (r FileReader) Seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

// Close implements io.Closer
func (r FileReader) Close() error {
	return r.r.Close()
}

// Stat returns os.FileInfo
func (r FileReader) Stat() (os.FileInfo, error) {
	return r.r.Stat()
}

func newFileReader(file *internal.File) (*FileReader, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	return &FileReader{
		r:    r,
		Name: file.Name,
		MIME: file.MIME,
	}, nil
}

// OpenFile ...
func (api API) OpenFile(token *AccessToken, filePath string) (*FileReader, error) {
	filePath = internal.RemoveTrailingSlash(filePath)
	dir := path.Dir(filePath)
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	hasViewAccess := folder.EnsureReadAccess(token.toLib()) == nil

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

	return newFileReader(file)
}

// UploadFileOptions ...
type UploadFileOptions struct {
	Folder    string
	File      multipart.File
	Header    *multipart.FileHeader
	Filename  string
	Tags      []string
	Overwrite bool
}

// UploadFile ...
func (api API) UploadFile(token *AccessToken, o *UploadFileOptions) error {
	changed := false
	folder, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return &ErrNotFound{}
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return &ErrNoReadAccess{Folder: o.Folder}
	}

	err = folder.EnsureWriteAccess(token.toLib())
	if err != nil {
		return &ErrNoWriteAccess{Folder: o.Folder}
	}

	limit := folder.MaxFileSizeMB << 20
	if o.Header.Size > limit {
		return &ErrSizeLimitExceeded{}
	}

	filename := govalidator.SafeFileName(o.Filename)
	if len(filename) == 0 || filename == "." {
		filename = govalidator.SafeFileName(o.Header.Filename)
		if len(filename) == 0 || filename == "." {
			filename = internal.Salt()
		}
	}

	mime, _ := mimetype.DetectReader(o.File)
	o.File.Seek(0, io.SeekStart)

	file := &internal.File{
		Name:     filename,
		Root:     api.root,
		RelPath:  path.Join(o.Folder, internal.FilenameToUUID(filename)),
		Tags:     o.Tags,
		MIME:     mime.String(),
		Size:     o.Header.Size,
		Uploaded: time.Now(),
	}
	err = file.Create(o.File, o.Overwrite)
	if err != nil {
		file.Delete()
		return err
	}

	folder.CacheFile(file)
	changed = true
	return nil
}

type limitedReader struct {
	r io.Reader
	n int64
}

func (r *limitedReader) Read(p []byte) (n int, err error) {
	if int64(len(p)) > r.n {
		p = p[:r.n]
	}
	n, err = r.r.Read(p)
	r.n -= int64(n)
	if r.n == 0 && err == nil {
		err = &ErrSizeLimitExceeded{}
	}
	return
}

func getResponseFilename(resp *http.Response) string {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if len(contentDisposition) > 0 {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := govalidator.SafeFileName(params["filename"])
		if len(filename) > 0 && filename != "." {
			return filename
		}
	}
	return govalidator.SafeFileName(path.Base(resp.Request.URL.Path))
}

// DownloadFileToFolderOptions ...
type DownloadFileToFolderOptions struct {
	Folder   string
	URL      string
	Filename string
	Tags     []string
}

// DownloadFileToFolder ...
func (api API) DownloadFileToFolder(token *AccessToken, o *DownloadFileToFolderOptions) error {
	changed := false
	folder, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return &ErrNotFound{}
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return &ErrNoReadAccess{Folder: o.Folder}
	}

	err = folder.EnsureWriteAccess(token.toLib())
	if err != nil {
		return &ErrNoWriteAccess{Folder: o.Folder}
	}

	req, err := http.NewRequest("GET", o.URL, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("bad response status code: %s", http.StatusText(resp.StatusCode))
	}

	limit := folder.MaxFileSizeMB << 20
	data := &limitedReader{
		r: resp.Body,
		n: limit,
	}

	filename := govalidator.SafeFileName(o.Filename)
	if len(filename) == 0 || filename == "." {
		filename = getResponseFilename(resp)
		if len(filename) == 0 || filename == "." {
			filename = internal.Salt()
		}
	}

	file := &internal.File{
		Name:     filename,
		Root:     api.root,
		RelPath:  path.Join(o.Folder, internal.FilenameToUUID(filename)),
		Tags:     o.Tags,
		Uploaded: time.Now(),
	}
	err = file.Create(data, false)
	if err != nil {
		file.Delete()
		return err
	}
	file.FixMimeAndSize()

	folder.CacheFile(file)
	changed = true
	return nil
}

// EditFileOptions ...
type EditFileOptions struct {
	Folder           string
	OriginalFilename string
	NewFilename      string
	Tags             []string
	Public           bool
}

// EditFile ...
func (api API) EditFile(token *AccessToken, o *EditFileOptions) error {
	changed := false
	folder, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return &ErrNotFound{}
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return &ErrNoReadAccess{Folder: o.Folder}
	}

	err = folder.EnsureWriteAccess(token.toLib())
	if err != nil {
		return &ErrNoWriteAccess{Folder: o.Folder}
	}

	file, err := folder.GetFile(o.OriginalFilename)
	if err != nil {
		return &ErrNotFound{}
	}

	oldTags := strings.Join(file.Tags, " ")
	newTags := strings.Join(o.Tags, " ")

	if newTags != oldTags || o.Public != file.Public {
		file.Tags = o.Tags
		file.Public = o.Public
		err := file.Save()
		if err != nil {
			return err
		}
		changed = true
	}

	newName := govalidator.SafeFileName(o.NewFilename)
	if newName == "." {
		return err
	}

	if newName != o.OriginalFilename {
		newPath := path.Join(o.Folder, newName)
		err := file.Move(newPath)
		if err != nil {
			return err
		}
		changed = true
	}

	return nil
}

// DeleteFile ...
func (api API) DeleteFile(token *AccessToken, filePath string) error {
	filePath = internal.RemoveTrailingSlash(filePath)
	dir := path.Dir(filePath)
	changed := false
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return &ErrNotFound{}
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return &ErrNoReadAccess{Folder: dir}
	}

	err = folder.EnsureWriteAccess(token.toLib())
	if err != nil {
		return &ErrNoWriteAccess{Folder: dir}
	}

	basename := filepath.Base(filePath)
	file, err := folder.GetFile(basename)
	if err != nil {
		return &ErrNotFound{}
	}

	err = file.Delete()
	if err == nil {
		folder.UncacheFile(file.Name)
		changed = true
	}
	return err
}
