package razbox

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/razzie/beepboop"
	"github.com/razzie/razbox/internal"
)

// MaxThumbnailWidth ...
const MaxThumbnailWidth = internal.MaxThumbnailWidth

// FileReader ...
type FileReader interface {
	http.File
	os.FileInfo
	MimeType() string
}

// OpenFile ...
func (api *API) OpenFile(sess *beepboop.Session, filePath string) (FileReader, error) {
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

	return file.Open()
}

// GetInternalFilename ...
func (api *API) GetInternalFilename(sess *beepboop.Session, filePath string) (string, error) {
	filePath = path.Clean(filePath)
	dir := path.Dir(filePath)
	folder, unlock, cached, err := api.getFolder(dir)
	if err != nil {
		return "", err
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
			return "", &ErrNoReadAccess{Folder: dir}
		}
		return "", &ErrNotFound{}
	}

	if !hasViewAccess && !file.Public {
		return "", &ErrNoReadAccess{Folder: dir}
	}

	return file.GetInternalFilename(), nil
}

// UploadFileOptions ...
type UploadFileOptions struct {
	Folder    string
	Files     []*multipart.FileHeader
	Filename  string
	Tags      []string
	Overwrite bool
	Public    bool
}

// UploadFile ...
func (api *API) UploadFile(sess *beepboop.Session, o *UploadFileOptions) error {
	changed := false
	folder, unlock, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return err
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()
	defer unlock()

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return &ErrNoReadAccess{Folder: o.Folder}
	}

	err = folder.EnsureWriteAccess(sess)
	if err != nil {
		return &ErrNoWriteAccess{Folder: o.Folder}
	}

	if len(o.Files) == 0 {
		return &ErrNoFiles{}
	}

	nthFilename := func(n int) string {
		if n == 0 || len(o.Filename) == 0 {
			return o.Filename
		}
		ext := path.Ext(o.Filename)
		return fmt.Sprintf("%s-%d%s", strings.TrimSuffix(o.Filename, ext), n+1, ext)
	}

	limit := folder.GetMaxUploadSizeMB() << 20
	for i, header := range o.Files {
		if header.Size > limit {
			return &ErrSizeLimitExceeded{}
		}

		limit -= header.Size
		filename, _ := getSafeFilename(nthFilename(i), header.Filename, internal.Salt())

		data, err := header.Open()
		if err != nil {
			return err
		}
		defer data.Close()

		mime, _ := internal.DetectContentType(data)
		data.Seek(0, io.SeekStart)

		file := &internal.File{
			Name:     filename,
			Root:     api.root,
			RelPath:  path.Join(o.Folder, internal.FilenameToUUID(filename)),
			Tags:     o.Tags,
			MIME:     mime,
			Size:     header.Size,
			Uploaded: time.Now(),
			Public:   o.Public,
		}
		err = file.Create(data, o.Overwrite)
		if err != nil {
			return err
		}

		folder.CacheFile(file)
		changed = true
	}

	return nil
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
func (api *API) DownloadFileToFolder(sess *beepboop.Session, o *DownloadFileToFolderOptions) error {
	changed := false
	folder, unlock, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return err
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()
	defer unlock()

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return &ErrNoReadAccess{Folder: o.Folder}
	}

	err = folder.EnsureWriteAccess(sess)
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

	limit := folder.GetMaxUploadSizeMB() << 20
	data := &LimitedReader{
		R: resp.Body,
		N: limit,
	}

	filename, _ := getSafeFilename(
		o.Filename,
		getContentDispositionFilename(resp.Header),
		path.Base(resp.Request.URL.Path),
		internal.Salt())

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
func (api *API) EditFile(sess *beepboop.Session, o *EditFileOptions) error {
	changed := false
	folder, unlock, cached, err := api.getFolder(o.Folder)
	if err != nil {
		return err
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()
	defer unlock()

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return &ErrNoReadAccess{Folder: o.Folder}
	}

	err = folder.EnsureWriteAccess(sess)
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

	newName := file.Name
	if len(o.NewFilename) > 0 {
		newName, err = getSafeFilename(o.NewFilename)
		if err != nil {
			return err
		}
	}

	newFolderName := o.Folder
	if len(o.MoveTo) > 0 {
		valid := false
		subfolders, _ := api.GetSubfolders(sess, o.Folder)
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
			newFolder, _, _ := api.getFolderNoLock(newFolderName)
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
func (api *API) DeleteFile(sess *beepboop.Session, filePath string) error {
	filePath = path.Clean(filePath)
	dir := path.Dir(filePath)
	changed := false
	folder, unlock, cached, err := api.getFolder(dir)
	if err != nil {
		return err
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()
	defer unlock()

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return &ErrNoReadAccess{Folder: dir}
	}

	err = folder.EnsureWriteAccess(sess)
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
	Width  int `json:"width"`
	Height int `json:"height"`
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
func (api *API) GetFileThumbnail(sess *beepboop.Session, filePath string) (*Thumbnail, error) {
	filePath = path.Clean(filePath)
	dir := path.Dir(filePath)
	folder, _, err := api.getFolderNoLock(dir)
	if err != nil {
		return nil, err
	}

	err = folder.EnsureReadAccess(sess)
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

	thumb, err := file.GetThumbnail(api.ThumbnailRetryAfter)
	if err != nil {
		return nil, err
	}
	return newThumbnail(thumb), nil
}
