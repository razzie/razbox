package razbox

import (
	"github.com/razzie/beepboop"
)

// Auth ...
func (api *API) Auth(pr *beepboop.PageRequest, folderName, accessType, password string) error {
	sess := pr.Session()

	if api.db != nil {
		if len(sess.IP()) == 0 {
			pr.Log("auth: no IP in request")
		}
		if ok, err := api.db.IsWithinRateLimit("auth", sess.IP(), api.AuthsPerMin); !ok && err == nil {
			return &ErrRateLimitExceeded{ReqPerMin: api.AuthsPerMin}
		}
	}

	folder, unlock, cached, err := api.getFolder(folderName)
	if err != nil {
		return err
	}
	if !cached {
		defer api.goCacheFolder(folder)
	}
	defer unlock()

	if folder.TestPassword(accessType, password) {
		newToken, err := folder.GetAccessToken(accessType)
		if err != nil {
			return err
		}
		return sess.MergeAccess(newToken)
	}

	return &ErrWrongPassword{}
}
