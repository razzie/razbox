package razbox

import (
	"html/template"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox/internal"
)

// FolderEntry ...
type FolderEntry struct {
	Folder        bool             `json:"folder,omitempty"`
	Prefix        template.HTML    `json:"prefix,omitempty"`
	Name          string           `json:"name"`
	RelPath       string           `json:"rel_path"`
	MIME          string           `json:"mime,omitempty"`
	PrimaryType   string           `json:"primary_type,omitempty"`
	SecondaryType string           `json:"secondary_type,omitempty"`
	Extension     string           `json:"extension"`
	Tags          []string         `json:"tags,omitempty"`
	Size          int64            `json:"size,omitempty"`
	Uploaded      int64            `json:"uploaded,omitempty"`
	Public        bool             `json:"public,omitempty"`
	EditMode      bool             `json:"edit_mode,omitempty"`
	HasThumbnail  bool             `json:"has_thumbnail,omitempty"`
	ThumbBounds   *ThumbnailBounds `json:"thumb_bounds,omitempty"`
	Archive       bool             `json:"archive,omitempty"`
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
	symbol, primaryType, secondaryType, extension := ExtendType(file.MIME, file.Name)
	entry := &FolderEntry{
		Prefix:        symbol,
		Name:          file.Name,
		RelPath:       path.Join(uri, file.Name),
		MIME:          file.MIME,
		PrimaryType:   primaryType,
		SecondaryType: secondaryType,
		Extension:     extension,
		Tags:          file.Tags,
		Size:          file.Size,
		Uploaded:      file.Uploaded.Unix(),
		Public:        file.Public,
		HasThumbnail:  internal.IsThumbnailSupported(file.MIME),
		Archive:       primaryType == "archive",
	}
	entry.updateThumbBounds(file, thumbnailRetryAfter)
	/*if entry.PrimaryType == "application" {
		if iface, _ := archiver.ByExtension(file.Name); iface != nil {
			entry.Prefix = "&#128230;"
			entry.Archive = true
		}
	}*/
	return entry
}

func (f *FolderEntry) updateThumbBounds(file *internal.File, thumbnailRetryAfter time.Duration) {
	bounds, _ := file.GetThumbnailBounds(thumbnailRetryAfter)
	if bounds == nil {
		if f.PrimaryType != "image" {
			f.HasThumbnail = false
		}
		return
	}

	w := bounds.Dx()
	h := bounds.Dy()
	if w > 0 && h > 0 {
		f.ThumbBounds = &ThumbnailBounds{
			Width:  w,
			Height: h,
		}
	}
}

// HasTag ...
func (f *FolderEntry) HasTag(tag string) bool {
	if tag == f.PrimaryType || tag == f.SecondaryType || tag == f.Extension {
		return true
	}
	for _, t := range f.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// GetFolderEntries ...
func (api *API) GetFolderEntries(sess *beepboop.Session, folderOrFilename string) ([]*FolderEntry, *FolderFlags, error) {
	folderOrFilename = path.Clean(folderOrFilename)

	var filename string
	dir := folderOrFilename
	if !internal.IsFolder(api.root, folderOrFilename) {
		dir = path.Dir(folderOrFilename)
		filename = folderOrFilename
	}

	folder, unlock, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, nil, err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}
	defer unlock()

	hasViewAccess := folder.EnsureReadAccess(sess) == nil
	hasEditAccess := folder.EnsureWriteAccess(sess) == nil

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
			return []*FolderEntry{entry}, nil, nil
		}
	}

	if !hasViewAccess {
		return nil, nil, &ErrNoReadAccess{Folder: dir}
	}

	var entries []*FolderEntry
	if folder.ConfigInherited {
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
	return entries, getFolderFlags(sess, folder), nil
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

// ExtendType ...
func ExtendType(mime, filename string) (symbol template.HTML, primaryType, secondaryType, extension string) {
	typ := strings.SplitN(mime, "/", 2)
	if len(typ) < 2 {
		typ = append(typ, "")
	}
	symbol = "&#128196;"
	primaryType = typ[0]
	secondaryType = typ[1]
	extension = path.Ext(filename)
	if len(extension) > 0 {
		extension = extension[1:]
	}

	switch primaryType {
	case "application":
		switch secondaryType {
		case "zip", "x-7z-compressed", "x-rar-compressed", "x-tar", "tar+gzip", "gzip", "x-bzip", "x-bzip2":
			symbol = "&#128230;"
			primaryType = "archive"
		case "vnd.microsoft.portable-executable", "x-executable", "vnd.debian.binary-package", "jar", "x-rpm":
			symbol = "&#128187;"
		case "pdf", "msword", "vnd.openxmlformats-officedocument.wordprocessingml.document", "x-mobipocket-ebook", "epub+zip":
			symbol = "&#128209;"
			primaryType = "document"
		case "x-iso9660-image", "x-cd-image", "x-raw-disk-image":
			symbol = "&#128191;"
			primaryType = "disk-image"
		case "vnd.ms-excel", "vnd.ms-powerpoint", "vnd.openxmlformats-officedocument.presentationml.presentation":
			symbol = "&#128200;"
			primaryType = "document"
		default:
			if iface, _ := archiver.ByExtension(filename); iface != nil {
				symbol = "&#128230;"
				primaryType = "archive"
			} else {
				primaryType = "other"
			}
		}
	case "audio":
		symbol = "&#127925;"
	case "font":
		symbol = "&#9000;"
	case "image":
		symbol = "&#127912;"
	case "model":
		symbol = "&#127922;"
	case "text":
		symbol = "&#128209;"
	case "video":
		symbol = "&#127916;"
	}

	return
}
