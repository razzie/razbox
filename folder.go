package razbox

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

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

func getFolderFlags(token *beepboop.AccessToken, f *internal.Folder) *FolderFlags {
	gotWriteAccess := f.EnsureWriteAccess(token.AccessMap) == nil
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

func (api API) getFolder(folderName string) (folder *internal.Folder, cached bool, err error) {
	cached = true
	if api.db != nil {
		folder, _ = internal.GetCachedFolder(api.db, folderName)
	}
	if folder == nil {
		cached = false
		folder, err = internal.GetFolder(api.root, folderName)
	}
	return
}

func (api API) goCacheFolder(folder *internal.Folder) {
	if api.db != nil {
		go internal.CacheFolder(api.db, folder)
	}
}

// GetFolderFlags ...
func (api API) GetFolderFlags(token *beepboop.AccessToken, folderName string) (*FolderFlags, error) {
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	err = folder.EnsureReadAccess(token.AccessMap)
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	return getFolderFlags(token, folder), nil
}

// ChangeFolderPassword ...
func (api API) ChangeFolderPassword(token *beepboop.AccessToken, folderName, accessType, password string) (*beepboop.AccessToken, error) {
	changed := false
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()

	err = folder.EnsureReadAccess(token.AccessMap)
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	err = folder.EnsureWriteAccess(token.AccessMap)
	if err != nil {
		return nil, &ErrNoWriteAccess{Folder: folderName}
	}

	err = folder.SetPassword(accessType, password)
	if err != nil {
		return nil, err
	}
	changed = true

	newToken, err := folder.GetAccessToken(accessType)
	if err != nil {
		return nil, err
	}

	if api.db != nil && len(token.SessionID) > 0 {
		tokenToRemove := make(beepboop.AccessMap)
		tokenToRemove.Add(accessType, folderName, "")
		if err := api.db.RemoveSessionAccess(token.SessionID, token.IP, tokenToRemove); err != nil {
			fmt.Println("session token error:", err)
		}

		if err := api.db.AddSessionAccess(token.SessionID, token.IP, newToken); err == nil {
			return &beepboop.AccessToken{
				SessionID: token.SessionID,
			}, nil
		}
		fmt.Println("session token error:", err)
	}

	return &beepboop.AccessToken{
		AccessMap: newToken,
	}, nil
}

// GetSubfolders ...
func (api *API) GetSubfolders(token *beepboop.AccessToken, folderName string) ([]string, error) {
	subfolders, err := api.getSubfoldersRecursive(token, folderName, true, false)
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

func (api *API) getSubfoldersRecursive(token *beepboop.AccessToken, folderName string, fromConfigRoot, inheritedOnly bool) ([]string, error) {
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		tmpFolder := folder
		defer api.goCacheFolder(tmpFolder)
	}

	err = folder.EnsureReadAccess(token.AccessMap)
	if err != nil {
		return nil, &ErrNoReadAccess{Folder: folderName}
	}

	if inheritedOnly && !folder.ConfigInherited {
		return nil, nil
	}

	if fromConfigRoot && folder.ConfigInherited {
		folder, cached, err = api.getFolder(folder.ConfigRootFolder)
		if err != nil {
			return nil, &ErrNotFound{}
		}
		if !cached {
			defer api.goCacheFolder(folder)
		}
	}

	var subfolders []string
	subfolders = append(subfolders, folder.RelPath)
	for _, subfolder := range folder.GetSubfolders() {
		subsubfolders, _ := api.getSubfoldersRecursive(token, path.Join(folder.RelPath, subfolder), false, true)
		subfolders = append(subfolders, subsubfolders...)
	}
	return subfolders, nil
}

// CreateSubfolder ...
func (api *API) CreateSubfolder(token *beepboop.AccessToken, folderName, subfolder string) (string, error) {
	changed := false
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return "", &ErrNotFound{}
	}
	defer func() {
		if !cached || changed {
			api.goCacheFolder(folder)
		}
	}()

	err = folder.EnsureReadAccess(token.AccessMap)
	if err != nil {
		return "", &ErrNoReadAccess{Folder: folderName}
	}

	err = folder.EnsureWriteAccess(token.AccessMap)
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
func (api *API) DeleteSubfolder(token *beepboop.AccessToken, folderName, subfolder string) error {
	flags, err := api.GetFolderFlags(token, path.Join(folderName, subfolder))
	if err != nil {
		return err
	}

	if !flags.EditMode {
		return &ErrNoWriteAccess{Folder: path.Join(folderName, subfolder)}
	}

	if !flags.Deletable {
		return &ErrNotDeletable{Name: subfolder}
	}

	parent, _, err := api.getFolder(folderName)
	if err != nil {
		return &ErrNotDeletable{Name: subfolder}
	}

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
	SizeStr       string           `json:"size_str,omitempty"`
	Uploaded      int64            `json:"uploaded,omitempty"`
	UploadedStr   string           `json:"uploaded_str,omitempty"`
	Public        bool             `json:"public,omitempty"`
	EditMode      bool             `json:"edit_mode,omitempty"`
	HasThumbnail  bool             `json:"has_thumbnail,omitempty"`
	ThumbBounds   *ThumbnailBounds `json:"thumb_bounds,omitempty"`
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
		SizeStr:       internal.ByteCountSI(file.Size),
		Uploaded:      file.Uploaded.Unix(),
		UploadedStr:   file.Uploaded.Format("Mon, 02 Jan 2006 15:04:05 MST"),
		Public:        file.Public,
		HasThumbnail:  internal.IsThumbnailSupported(file.MIME),
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
func (api API) GetFolderEntries(token *beepboop.AccessToken, folderOrFilename string) ([]*FolderEntry, *FolderFlags, error) {
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

	hasViewAccess := folder.EnsureReadAccess(token.AccessMap) == nil
	hasEditAccess := folder.EnsureWriteAccess(token.AccessMap) == nil

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
