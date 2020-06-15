package razbox

import (
	"log"

	"github.com/google/uuid"
)

// Auth ...
func (api API) Auth(token *AccessToken, folderName, accessType, password string) (*AccessToken, error) {
	sessionID := token.SessionID

	if api.db != nil {
		if len(token.IP) == 0 {
			log.Println("auth: no IP in request")
		}
		if ok, _ := api.db.IsWithinRateLimit("auth", token.IP, api.AuthsPerMin); !ok {
			return nil, &ErrRateLimitExceeded{ReqPerMin: api.AuthsPerMin}
		}

		if len(sessionID) > 0 {
			libToken := token.toLib()
			err := api.db.FillSessionToken(sessionID, libToken)
			if err != nil {
				sessionID = ""
			} else {
				token = token.fromLib(libToken)
			}
		} else {
			newSessionID, err := uuid.NewRandom()
			if err == nil {
				sessionID = newSessionID.String()
			}
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
		newToken, err := folder.GetAccessToken(accessType)
		if err != nil {
			return nil, err
		}
		if api.db != nil && len(sessionID) > 0 {
			if err := api.db.AddSessionToken(sessionID, newToken, api.CookieExpiration); err == nil {
				return &AccessToken{
					SessionID: sessionID,
				}, nil
			}
		}
		return new(AccessToken).fromLib(newToken), nil
	}

	return nil, &ErrWrongPassword{}
}
