package razbox

import (
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mholt/archiver"
	"github.com/razzie/beepboop"
	"github.com/razzie/razbox/internal"
)

// FolderFlags ...
type FolderFlags struct {
	EditMode        bool
	Editable        bool
	Deletable       bool
	Configurable    bool
	Subfolders      bool
	MaxUploadSizeMB int64
}

func getFolderFlags(sess *beepboop.Session, f *internal.Folder) *FolderFlags {
	gotWriteAccess := f.EnsureWriteAccess(sess) == nil
	deletable := false
	if gotWriteAccess && f.ConfigInherited {
		entries, err := ioutil.ReadDir(path.Join(f.Root, f.RelPath))
		deletable = (err == nil) && len(entries) == 0
	}

	return &FolderFlags{
		EditMode:        gotWriteAccess,
		Editable:        len(f.Config.WritePassword) > 0,
		Deletable:       deletable,
		Configurable:    !f.ConfigInherited,
		Subfolders:      f.Config.Subfolders,
		MaxUploadSizeMB: f.GetMaxUploadSizeMB(),
	}
}

func (api *API) lockFolder(folder *internal.Folder) (unlock func(), err error) {
	if _, locked := api.folderLock.LoadOrStore(folder.ConfigRootFolder, nil); locked {
		return nil, &ErrFolderBusy{}
	}
	return func() {
		api.folderLock.Delete(folder.ConfigRootFolder)
	}, nil
}

func (api *API) getFolder(folderName string) (folder *internal.Folder, unlock func(), cached bool, err error) {
	folder, cached, err = api.getFolderNoLock(folderName)
	if folder != nil {
		unlock, err = api.lockFolder(folder)
		if err != nil {
			folder = nil
		}
	}
	return
}

func (api *API) getFolderNoLock(folderName string) (folder *internal.Folder, cached bool, err error) {
	cached = true
	if api.db != nil {
		folder, _ = internal.GetCachedFolder(api.db, folderName)
	}
	if folder == nil {
		cached = false
		folder, err = internal.GetFolder(api.root, folderName)
		if err != nil {
			err = &ErrNotFound{}
		}
	}
	return
}

func (api *API) goCacheFolder(folder *internal.Folder) {
	if api.db != nil {
		go internal.CacheFolder(api.db, folder)
	}
}

// GetFolderFlags ...
func (api *API) GetFolderFlags(sess *beepboop.Session, folderName string) (*FolderFlags, error) {
	folder, unlock, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}
	defer unlock()

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	return getFolderFlags(sess, folder), nil
}

// ChangeFolderPassword ...
func (api *API) ChangeFolderPassword(sess *beepboop.Session, folderName, accessType, password string) error {
	changed := false
	folder, unlock, cached, err := api.getFolder(folderName)
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
		return &ErrNoReadAccess{Folder: folderName}
	}

	err = folder.EnsureWriteAccess(sess)
	if err != nil {
		return &ErrNoWriteAccess{Folder: folderName}
	}

	err = folder.SetPassword(accessType, password)
	if err != nil {
		return err
	}
	changed = true

	newToken, err := folder.GetAccessToken(accessType)
	if err != nil {
		return err
	}

	sess.RemoveAccess(accessType, folderName)
	return sess.MergeAccess(newToken)
}

// GetSubfolders ...
func (api *API) GetSubfolders(sess *beepboop.Session, folderName string) ([]string, error) {
	subfolders, err := api.getSubfoldersRecursive(sess, folderName, true, false)
	if err != nil {
		return nil, err
	}

	relSubfolders := make([]string, 0, len(subfolders))
	for _, subfolder := range subfolders {
		relSubfolder, _ := filepath.Rel(folderName, subfolder)
		if len(relSubfolder) == 0 || relSubfolder == "." {
			continue
		}
		relSubfolders = append(relSubfolders, relSubfolder)
	}
	return relSubfolders, nil
}

func (api *API) getSubfoldersRecursive(sess *beepboop.Session, folderName string, fromConfigRoot, inheritedOnly bool) ([]string, error) {
	folder, cached, err := api.getFolderNoLock(folderName)
	if err != nil {
		return nil, err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	if inheritedOnly && !folder.ConfigInherited {
		return nil, nil
	}

	if fromConfigRoot && folder.ConfigInherited {
		folder, cached, err = api.getFolderNoLock(folder.ConfigRootFolder)
		if err != nil {
			return nil, err
		}
		if !cached {
			defer api.goCacheFolder(folder)
		}
	}

	var subfolders []string
	subfolders = append(subfolders, folder.RelPath)
	for _, subfolder := range folder.GetSubfolders() {
		subsubfolders, _ := api.getSubfoldersRecursive(sess, path.Join(folder.RelPath, subfolder), false, true)
		subfolders = append(subfolders, subsubfolders...)
	}
	return subfolders, nil
}

// CreateSubfolder ...
func (api *API) CreateSubfolder(sess *beepboop.Session, folderName, subfolder string) (string, error) {
	changed := false
	folder, unlock, cached, err := api.getFolder(folderName)
	if err != nil {
		return "", err
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()
	defer unlock()

	err = folder.EnsureReadAccess(sess)
	if err != nil {
		return "", &ErrNoReadAccess{Folder: folderName}
	}

	err = folder.EnsureWriteAccess(sess)
	if err != nil {
		return "", &ErrNoWriteAccess{Folder: folderName}
	}

	if !folder.Config.Subfolders {
		return "", &ErrSubfoldersDisabled{Folder: folderName}
	}

	safeName, err := getSafeFilename(subfolder)
	if err != nil {
		return "", err
	}

	err = os.Mkdir(path.Join(api.root, folder.RelPath, safeName), 0755)
	if err != nil {
		return "", err
	}

	folder.CacheSubfolder(safeName)
	changed = true
	return path.Join(folder.RelPath, safeName), nil
}

// DeleteSubfolder ...
func (api *API) DeleteSubfolder(sess *beepboop.Session, folderName, subfolder string) error {
	flags, err := api.GetFolderFlags(sess, path.Join(folderName, subfolder))
	if err != nil {
		return err
	}

	if !flags.EditMode {
		return &ErrNoWriteAccess{Folder: path.Join(folderName, subfolder)}
	}

	if !flags.Deletable {
		return &ErrNotDeletable{Name: subfolder}
	}

	parent, unlock, _, err := api.getFolder(folderName)
	if err != nil {
		return &ErrNotDeletable{Name: subfolder}
	}
	defer unlock()

	err = os.Remove(path.Join(api.root, folderName, subfolder))
	if err != nil {
		return err
	}

	parent.UncacheSubfolder(subfolder)
	api.goCacheFolder(parent)
	return nil
}

// FolderEntry ...
type FolderEntry struct {
	Folder        bool             `json:"folder,omitempty"`
	Prefix        template.HTML    `json:"prefix,omitempty"`
	Name          string           `json:"name"`
	RelPath       string           `json:"rel_path"`
	MIME          string           `json:"mime,omitempty"`
	PrimaryType   string           `json:"primary_type,omitempty"`
	SecondaryType string           `json:"secondary_type,omitempty"`
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
	typ := strings.SplitN(file.MIME, "/", 2)
	if len(typ) < 2 {
		typ = append(typ, "")
	}
	entry := &FolderEntry{
		Prefix:        internal.MIMEtoSymbol(file.MIME),
		Name:          file.Name,
		RelPath:       path.Join(uri, file.Name),
		MIME:          file.MIME,
		PrimaryType:   typ[0],
		SecondaryType: typ[1],
		Tags:          file.Tags,
		Size:          file.Size,
		Uploaded:      file.Uploaded.Unix(),
		Public:        file.Public,
		HasThumbnail:  internal.IsThumbnailSupported(file.MIME),
	}
	entry.updateThumbBounds(file, thumbnailRetryAfter)
	if entry.PrimaryType == "application" {
		if iface, _ := archiver.ByExtension(file.Name); iface != nil {
			entry.Prefix = "&#128230;"
			entry.Archive = true
		}
	}
	return entry
}

func (f *FolderEntry) updateThumbBounds(file *internal.File, thumbnailRetryAfter time.Duration) {
	thumb := file.Thumbnail
	if thumb == nil {
		return
	}

	if f.PrimaryType != "image" &&
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
	if tag == f.PrimaryType {
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
