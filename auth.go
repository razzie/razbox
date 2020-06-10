package razbox

import (
	"log"
)

// Auth ...
func (api API) Auth(token *AccessToken, folderName, accessType, password string) (*AccessToken, error) {
	if api.db != nil {
		if len(token.IP) == 0 {
			log.Println("auth: no IP in request")
		}
		if ok, _ := api.db.IsWithinRateLimit("auth", token.IP, api.AuthsPerMin); !ok {
			return nil, &ErrRateLimitExceeded{ReqPerMin: api.AuthsPerMin}
		}
	}

	folder, cached, err := api.getFolder(folderName)
	if err != nil {
		return nil, &ErrNotFound{}
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}

	if folder.TestPassword(accessType, password) {
		token, err := folder.GetAccessToken(accessType)
		return new(AccessToken).fromLib(token), err
	}

	return nil, &ErrWrongPassword{}
}
