package razbox

import (
	"html/template"
	"path"
	"path/filepath"
	"sort"

	"github.com/razzie/razbox/internal"
)

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

func newFileEntry(uri string, file *internal.File) *FolderEntry {
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
	if file.Thumbnail != nil {
		w := file.Thumbnail.Bounds.Dx()
		h := file.Thumbnail.Bounds.Dy()
		if w > 0 && h > 0 {
			entry.ThumbBounds = &ThumbnailBounds{
				Width:  w,
				Height: h,
			}
		}
	}
	return entry
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
	folderOrFilename = internal.RemoveTrailingSlash(folderOrFilename)

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
			entry := newFileEntry(folderOrFilename, file)
			entry.EditMode = hasEditAccess
			return []*FolderEntry{entry}, nil, nil
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
		entry := newFileEntry(folderOrFilename, file)
		entry.EditMode = hasEditAccess
		entries = append(entries, entry)
	}

	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Uploaded > entries[j].Uploaded
	})
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Folder && !entries[j].Folder
	})

	return entries, getFolderFlags(token, folder), nil
}
