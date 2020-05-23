package razbox

// Auth ...
func (api API) Auth(folderName, accessType, password string) (*AccessToken, error) {
	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	if folder.TestPassword(accessType, password) {
		token, err := folder.GetAccessToken(accessType)
		return &AccessToken{
			Read:  token.Read,
			Write: token.Write,
		}, err
	}

	return nil, &ErrWrongPassword{}
}
