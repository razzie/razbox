package internal

import (
	"path"

	"github.com/razzie/beepboop"
)

// GetCachedFolder returns a cached Folder
func GetCachedFolder(db *beepboop.DB, folderName string) (*Folder, error) {
	var folder Folder
	err := db.GetCachedValue("folder:"+path.Clean(folderName), &folder)
	if err != nil {
		return nil, err
	}

	if len(folder.CachedFiles) == 0 || len(folder.CachedSubfolders) == 0 {
		return nil, &ErrFolderMissingCache{Folder: folder.RelPath}
	}

	return &folder, nil
}

// CacheFolder caches a Folder
func CacheFolder(db *beepboop.DB, folder *Folder) error {
	if len(folder.CachedFiles) == 0 {
		folder.GetFiles()
	}
	if len(folder.CachedSubfolders) == 0 {
		folder.GetSubfolders()
	}

	return db.CacheValue("folder:"+path.Clean(folder.RelPath), folder, true)
}

// UncacheFolder uncaches a Folder
func UncacheFolder(db *beepboop.DB, folderName string) error {
	return db.UncacheValue("folder:" + path.Clean(folderName))
}
