package api

import (
	"html/template"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/razzie/razbox/lib"
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
	Uploaded     time.Time
	UploadedStr  string
	Public       bool
	EditMode     bool
	HasThumbnail bool
}

func newSubfolderEntry(uri, subfolder string) *FolderEntry {
	return &FolderEntry{
		Folder:  true,
		Prefix:  "&#128194;",
		Name:    subfolder,
		RelPath: path.Join(uri, subfolder),
	}
}

func newFileEntry(uri string, file *lib.File) *FolderEntry {
	return &FolderEntry{
		Prefix:       lib.MIMEtoSymbol(file.MIME),
		Name:         file.Name,
		RelPath:      path.Join(uri, file.Name),
		MIME:         file.MIME,
		Tags:         file.Tags,
		Size:         file.Size,
		SizeStr:      lib.ByteCountSI(file.Size),
		Uploaded:     file.Uploaded,
		UploadedStr:  file.Uploaded.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		Public:       file.Public,
		HasThumbnail: lib.IsThumbnailSupported(file.MIME),
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
	folderOrFilename = lib.RemoveTrailingSlash(folderOrFilename)

	var filename string
	dir := folderOrFilename
	if !lib.IsFolder(api.root, folderOrFilename) {
		dir = path.Dir(folderOrFilename)
		filename = folderOrFilename
	}

	folder, cached, err := api.getFolder(dir)
	if err != nil {
		return nil, nil, err
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
			return nil, nil, err
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
	return entries, getFolderFlags(token, folder), nil
}

// SortFolderEntries ...
func (api API) SortFolderEntries(files []*FolderEntry, order string) {
	switch order {
	case "name_asc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Name > files[j].Name
		})

	case "name_desc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Name < files[j].Name
		})

	case "type_asc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].MIME > files[j].MIME
		})

	case "type_desc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].MIME < files[j].MIME
		})

	case "tags_asc":
		sort.SliceStable(files, func(i, j int) bool {
			return strings.Join(files[i].Tags, " ") > strings.Join(files[j].Tags, " ")
		})

	case "tags_desc":
		sort.SliceStable(files, func(i, j int) bool {
			return strings.Join(files[i].Tags, " ") < strings.Join(files[j].Tags, " ")
		})

	case "size_asc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Size > files[j].Size
		})

	case "size_desc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Size < files[j].Size
		})

	case "uploaded_asc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Uploaded.After(files[j].Uploaded)
		})

	case "uploaded_desc":
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Uploaded.Before(files[j].Uploaded)
		})

	default:
		sort.SliceStable(files, func(i, j int) bool {
			return files[i].Uploaded.After(files[j].Uploaded)
		})
	}

	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Folder && !files[j].Folder
	})
}
