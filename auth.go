package razbox

import (
	"github.com/google/uuid"
	"github.com/razzie/beepboop"
)

// Auth ...
func (api API) Auth(pr *beepboop.PageRequest, folderName, accessType, password string) (*beepboop.AccessToken, error) {
	token := beepboop.NewAccessTokenFromRequest(pr)
	sessionID := token.SessionID

	if api.db != nil {
		if len(token.IP) == 0 {
			pr.Log("auth: no IP in request")
		}
		if ok, err := api.db.IsWithinRateLimit("auth", token.IP, api.AuthsPerMin); !ok && err == nil {
			return nil, &ErrRateLimitExceeded{ReqPerMin: api.AuthsPerMin}
		}

		if len(sessionID) > 0 {
			sessToken, err := api.db.GetAccessToken(sessionID, token.IP)
			if err != nil {
				pr.Log("session token error:", err)
				sessionID = ""
			} else {
				token.AccessMap.Merge(sessToken.AccessMap)
			}
		} else {
			newSessionID, err := uuid.NewRandom()
			if err != nil {
				pr.Log("session ID gen err:", err)
			} else {
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
			if err = api.db.AddSessionAccess(sessionID, token.IP, newToken); err == nil {
				return &beepboop.AccessToken{
					SessionID: sessionID,
				}, nil
			}
			pr.Log("session token error:", err)
		}
		return &beepboop.AccessToken{
			SessionID: sessionID,
			AccessMap: newToken,
		}, nil
	}

	return nil, &ErrWrongPassword{}
}
