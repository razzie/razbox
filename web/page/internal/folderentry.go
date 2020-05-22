package internal

import (
	"html/template"
	"path"
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

// NewSubfolderEntry ...
func NewSubfolderEntry(uri, subfolder string) *FolderEntry {
	return &FolderEntry{
		Folder:  true,
		Prefix:  "&#128194;",
		Name:    subfolder,
		RelPath: path.Join(uri, subfolder),
	}
}

// NewFileEntry ...
func NewFileEntry(uri string, file *lib.File) *FolderEntry {
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

// SortFolderEntries ...
func SortFolderEntries(files []*FolderEntry, order string) {
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
