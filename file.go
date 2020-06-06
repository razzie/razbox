package razbox

import (
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
	filePath = path.Clean(filePath)
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
	Public    bool
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

	limit := folder.Config.MaxFileSizeMB << 20
	if o.Header.Size > limit {
		return &ErrSizeLimitExceeded{}
	}

	filename := govalidator.SafeFileName(o.Filename)
	if len(filename) == 0 || filename == "." || filename == ".." {
		filename = govalidator.SafeFileName(o.Header.Filename)
		if len(filename) == 0 || filename == "." || filename == ".." {
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
		Public:   o.Public,
	}
	err = file.Create(o.File, o.Overwrite)
	if err != nil {
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

func (l *limitedReader) Read(p []byte) (n int, err error) {
	if l.n <= 0 {
		return 0, &ErrSizeLimitExceeded{}
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err = l.r.Read(p)
	l.n -= int64(n)
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
	Folder    string
	URL       string
	Filename  string
	Tags      []string
	Overwrite bool
	Public    bool
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
		return &ErrBadHTTPResponseStatus{StatusCode: resp.StatusCode}
	}

	limit := folder.Config.MaxFileSizeMB << 20
	data := &limitedReader{
		r: resp.Body,
		n: limit,
	}

	filename := govalidator.SafeFileName(o.Filename)
	if len(filename) == 0 || filename == "." || filename == ".." {
		filename = getResponseFilename(resp)
		if len(filename) == 0 || filename == "." || filename == ".." {
			filename = internal.Salt()
		}
	}

	file := &internal.File{
		Name:     filename,
		Root:     api.root,
		RelPath:  path.Join(o.Folder, internal.FilenameToUUID(filename)),
		Tags:     o.Tags,
		Uploaded: time.Now(),
		Public:   o.Public,
	}
	err = file.Create(data, o.Overwrite)
	if err != nil {
		return err
	}

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
	MoveTo           string
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
	if newName == "." || newName == ".." {
		return err
	}

	if len(newName) == 0 {
		newName = file.Name
	}

	newFolderName := o.Folder
	if len(o.MoveTo) > 0 {
		valid := false
		subfolders, _ := api.GetSubfolders(token, o.Folder)
		for _, subfolder := range subfolders {
			if subfolder == o.MoveTo {
				valid = true
				break
			}
		}
		if !valid {
			return &ErrInvalidMoveLocation{Location: o.MoveTo}
		}
		newFolderName = path.Join(o.Folder, o.MoveTo)
	}

	if newName != o.OriginalFilename || len(o.MoveTo) > 0 {
		newPath := path.Join(newFolderName, newName)
		err := file.Move(newPath)
		if err != nil {
			return err
		}
		if newFolderName != o.Folder {
			folder.UncacheFile(o.OriginalFilename)
			newFolder, _, _ := api.getFolder(newFolderName)
			if newFolder != nil {
				newFolder.CacheFile(file)
				api.goCacheFolder(newFolder)
			}
		}
		changed = true
	}

	return nil
}

// DeleteFile ...
func (api API) DeleteFile(token *AccessToken, filePath string) error {
	filePath = path.Clean(filePath)
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

// Thumbnail ...
type Thumbnail struct {
	Data   []byte
	MIME   string
	Bounds ThumbnailBounds
}

// ThumbnailBounds ...
type ThumbnailBounds struct {
	Width  int
	Height int
}

func newThumbnail(thumb *internal.Thumbnail) *Thumbnail {
	return &Thumbnail{
		Data: thumb.Data,
		MIME: thumb.MIME,
		Bounds: ThumbnailBounds{
			Width:  thumb.Bounds.Dx(),
			Height: thumb.Bounds.Dy(),
		},
	}
}

// GetFileThumbnail ...
func (api API) GetFileThumbnail(token *AccessToken, filePath string, maxWidth uint) (*Thumbnail, error) {
	filePath = path.Clean(filePath)
	dir := path.Dir(filePath)
	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, &ErrNotFound{}
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
		return nil, &ErrNotFound{}
	}

	if !internal.IsThumbnailSupported(file.MIME) {
		return nil, &ErrUnsupportedFileFormat{MIME: file.MIME}
	}

	thumb := file.Thumbnail
	if thumb == nil || (len(thumb.Data) == 0 && thumb.Timestamp.Add(api.ThumbnailRetryAfter).Before(time.Now())) {
		thumb, err = internal.GetThumbnail(path.Join(api.root, file.RelPath+".bin"), file.MIME, maxWidth)
		defer file.Save()
		if err != nil {
			file.Thumbnail = &internal.Thumbnail{Timestamp: time.Now()}
			return nil, err
		}

		file.Thumbnail = thumb
	}

	return newThumbnail(thumb), nil
}
