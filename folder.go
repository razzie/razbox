package razbox

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

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
