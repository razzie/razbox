package api

import (
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/razzie/razbox/lib"
)

// ErrSizeLimitExceeded ...
type ErrSizeLimitExceeded struct{}

func (err ErrSizeLimitExceeded) Error() string {
	return "size limit exceeded"
}

// FileReader ...
type FileReader struct {
	r    lib.FileReader
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

func newFileReader(file *lib.File) (*FileReader, error) {
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
	filePath = lib.RemoveTrailingSlash(filePath)
	dir := path.Dir(filePath)
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, err
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
		return nil, err
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
	folder, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

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
			filename = lib.Salt()
		}
	}

	mime, _ := mimetype.DetectReader(o.File)
	o.File.Seek(0, io.SeekStart)

	file := &lib.File{
		Name:     filename,
		RelPath:  path.Join(o.Folder, lib.FilenameToUUID(filename)),
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
	return nil
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
	folder, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

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
			filename = lib.Salt()
		}
	}

	file := &lib.File{
		Name:     filename,
		RelPath:  path.Join(o.Folder, lib.FilenameToUUID(filename)),
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
	folder, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

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
		return err
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
		folder.UncacheFile(o.OriginalFilename)
		folder.CacheFile(file)
	}

	return nil
}

// DeleteFile ...
func (api API) DeleteFile(token *AccessToken, filePath string) error {
	filePath = lib.RemoveTrailingSlash(filePath)
	dir := path.Dir(filePath)
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

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
		return err
	}

	err = file.Delete()
	if err == nil {
		folder.UncacheFile(file.Name)
	}
	return err
}

// Thumbnail ...
type Thumbnail struct {
	Data   []byte
	MIME   string
	Bounds image.Rectangle
}

// GetFileThumbnail ...
func (api API) GetFileThumbnail(token *AccessToken, filePath string) (*Thumbnail, error) {
	filePath = lib.RemoveTrailingSlash(filePath)
	dir := path.Dir(filePath)
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: dir}
	}

	basename := filepath.Base(filePath)
	file, err := folder.GetFile(basename)
	if err != nil {
		return nil, err
	}

	if !lib.IsThumbnailSupported(file.MIME) {
		return nil, fmt.Errorf("unsupported format: %s", file.MIME)
	}

	return api.getThumbnail(filePath, file)
}
