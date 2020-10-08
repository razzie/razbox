package beepboop

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/mo7zayed/reqip"
)

// AccessToken ...
type AccessToken struct {
	SessionID string
	IP        string
	AccessMap AccessMap
}

// NewAccessToken returns a new AccessToken
func NewAccessToken() *AccessToken {
	return &AccessToken{
		AccessMap: make(AccessMap),
	}
}

// NewAccessTokenFromRequest returns a new AccessToken from a page request
func NewAccessTokenFromRequest(r *PageRequest) *AccessToken {
	token := new(AccessToken).fromCookies(r.Request.Cookies())
	token.IP = reqip.GetClientIP(r.Request)
	if len(token.IP) == 0 {
		token.IP, _, _ = net.SplitHostPort(r.Request.RemoteAddr)
	}
	db := r.Context.DB
	if db != nil && len(token.SessionID) > 0 {
		dbToken, err := db.GetAccessToken(token.SessionID, token.IP)
		if err == nil {
			token.AccessMap.Merge(dbToken.AccessMap)
		} else {
			r.Log(err)
		}
	}
	return token
}

func (token *AccessToken) fromCookies(cookies []*http.Cookie) *AccessToken {
	if token.AccessMap == nil {
		token.AccessMap = make(AccessMap)
	}
	for _, c := range cookies {
		if c.Name == "session" {
			token.SessionID = c.Value
			continue
		}
		access := strings.SplitN(c.Name, "-", 2)
		if len(access) < 2 {
			continue
		}
		token.AccessMap.Add(access[0], access[1], c.Value)
	}
	return token
}

// ToCookie returns either a SessionID cookie or a cookie containing the access to a single resource
func (token *AccessToken) ToCookie(expiration time.Duration) *http.Cookie {
	if len(token.SessionID) > 0 {
		return &http.Cookie{
			Name:    "session",
			Value:   token.SessionID,
			Path:    "/",
			Expires: time.Now().Add(expiration),
		}
	}
	for typ, res := range token.AccessMap {
		for resname, code := range res {
			return &http.Cookie{
				Name:    fmt.Sprintf("%s-%s", typ, resname),
				Value:   string(code),
				Path:    "/",
				Expires: time.Now().Add(expiration),
			}
		}
	}
	return nil
}
