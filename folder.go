package razbox

import (
	"fmt"
	"html/template"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/razzie/razbox/internal"
)

// FolderFlags ...
type FolderFlags struct {
	EditMode      bool
	Editable      bool
	Configurable  bool
	MaxFileSizeMB int64
}

func getFolderFlags(token *AccessToken, folder *internal.Folder) *FolderFlags {
	return &FolderFlags{
		EditMode:      folder.EnsureWriteAccess(token.toLib()) == nil,
		Editable:      len(folder.WritePassword) > 0,
		Configurable:  !folder.ConfigInherited,
		MaxFileSizeMB: folder.MaxFileSizeMB,
	}
}

func (api API) getFolder(folderName string) (folder *internal.Folder, cached bool, err error) {
	cached = true
	if api.db != nil {
		folder, _ = api.db.GetCachedFolder(folderName)
	}
	if folder == nil {
		cached = false
		folder, err = internal.GetFolder(api.root, folderName)
	}
	return
}

func (api API) goCacheFolder(folder *internal.Folder) {
	if api.db != nil {
		go api.db.CacheFolder(folder)
	}
}

// GetFolderFlags ...
func (api API) GetFolderFlags(token *AccessToken, folderName string) (*FolderFlags, error) {
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	return getFolderFlags(token, folder), nil
}

// ChangeFolderPassword ...
func (api API) ChangeFolderPassword(token *AccessToken, folderName, accessType, password string) (*AccessToken, error) {
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	err = folder.EnsureReadAccess(token.toLib())
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	err = folder.EnsureWriteAccess(token.toLib())
	if err != nil {
		return nil, &ErrNoWriteAccess{Folder: folderName}
	}

	if folder.ConfigInherited {
		return nil, fmt.Errorf("Cannot change password of folders that inherit parent configuration")
	}

	err = folder.SetPassword(accessType, password)
	if err != nil {
		return nil, err
	}

	newToken, err := folder.GetAccessToken(accessType)
	if err != nil {
		return nil, err
	}

	return &AccessToken{
		Read:  newToken.Read,
		Write: newToken.Write,
	}, nil
}

// FolderEntry ...
type FolderEntry struct {
	Folder       bool
	Prefix       template.HTML
	Name         string
	RelPath      string
	MIME         string
	Tags         []string
	Size         int64
	SizeStr      string
	Uploaded     int64
	UploadedStr  string
	Public       bool
	EditMode     bool
	HasThumbnail bool
	ThumbBounds  *ThumbnailBounds
}

func newSubfolderEntry(uri, subfolder string) *FolderEntry {
	return &FolderEntry{
		Folder:  true,
		Prefix:  "&#128194;",
		Name:    subfolder,
		RelPath: path.Join(uri, subfolder),
	}
}

func newFileEntry(uri string, file *internal.File, thumbnailRetryAfter time.Duration) *FolderEntry {
	entry := &FolderEntry{
		Prefix:       internal.MIMEtoSymbol(file.MIME),
		Name:         file.Name,
		RelPath:      path.Join(uri, file.Name),
		MIME:         file.MIME,
		Tags:         file.Tags,
		Size:         file.Size,
		SizeStr:      internal.ByteCountSI(file.Size),
		Uploaded:     file.Uploaded.Unix(),
		UploadedStr:  file.Uploaded.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		Public:       file.Public,
		HasThumbnail: internal.IsThumbnailSupported(file.MIME),
	}
	entry.updateThumbBounds(file, thumbnailRetryAfter)
	return entry
}

func (f *FolderEntry) updateThumbBounds(file *internal.File, thumbnailRetryAfter time.Duration) {
	thumb := file.Thumbnail
	if thumb == nil {
		return
	}

	if !strings.HasPrefix(f.MIME, "image/") &&
		len(thumb.Data) == 0 &&
		thumb.Timestamp.Add(thumbnailRetryAfter).After(time.Now()) {

		f.HasThumbnail = false
		return
	}

	w := thumb.Bounds.Dx()
	h := thumb.Bounds.Dy()
	if w > 0 && h > 0 {
		f.ThumbBounds = &ThumbnailBounds{
			Width:  w,
			Height: h,
		}
	}
}

// HasTag ...
func (f *FolderEntry) HasTag(tag string) bool {
	for _, t := range f.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetFolderEntries ...
func (api API) GetFolderEntries(token *AccessToken, folderOrFilename string) ([]*FolderEntry, *FolderFlags, error) {
	folderOrFilename = path.Clean(folderOrFilename)

	var filename string
	dir := folderOrFilename
	if !internal.IsFolder(api.root, folderOrFilename) {
		dir = path.Dir(folderOrFilename)
		filename = folderOrFilename
	}

	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	hasViewAccess := folder.EnsureReadAccess(token.toLib()) == nil
	hasEditAccess := folder.EnsureWriteAccess(token.toLib()) == nil

	if len(filename) > 0 {
		file, err := folder.GetFile(filepath.Base(filename))
		if err != nil {
			if !hasViewAccess {
				return nil, nil, &ErrNoReadAccess{Folder: dir}
			}
			return nil, nil, &ErrNotFound{}
		}

		if hasViewAccess || file.Public {
			entry := newFileEntry(folderOrFilename, file, api.ThumbnailRetryAfter)
			entry.EditMode = hasEditAccess
			return []*FolderEntry{entry}, getFolderFlags(token, folder), nil
		}
	}

	if !hasViewAccess {
		return nil, nil, &ErrNoReadAccess{Folder: dir}
	}

	var entries []*FolderEntry
	if len(folder.RelPath) > 0 {
		entries = append(entries, newSubfolderEntry(folderOrFilename, ".."))
	}
	for _, subfolder := range folder.GetSubfolders() {
		entry := newSubfolderEntry(folderOrFilename, subfolder)
		entries = append(entries, entry)
	}
	for _, file := range folder.GetFiles() {
		entry := newFileEntry(folderOrFilename, file, api.ThumbnailRetryAfter)
		entry.EditMode = hasEditAccess
		entries = append(entries, entry)
	}

	SortFolderEntries(entries)
	return entries, getFolderFlags(token, folder), nil
}

// SortFolderEntries sorts the entries by the upload date (most recent first)
func SortFolderEntries(entries []*FolderEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Uploaded > entries[j].Uploaded
	})
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Folder && !entries[j].Folder
	})
}
