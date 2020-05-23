package razbox

import (
	"fmt"

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

// GetFolderFlags ...
func (api API) GetFolderFlags(token *AccessToken, folderName string) (*FolderFlags, error) {
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, err
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
		return nil, err
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
