package razbox

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/mo7zayed/reqip"
	"github.com/razzie/razbox/internal"
)

// AccessToken ...
type AccessToken struct {
	SessionID string
	IP        string
	Read      map[string]string
	Write     map[string]string
}

func (token *AccessToken) fromCookies(cookies []*http.Cookie) *AccessToken {
	if token.Read == nil {
		token.Read = make(map[string]string)
	}
	if token.Write == nil {
		token.Write = make(map[string]string)
	}

	for _, c := range cookies {
		switch {
		case c.Name == "session":
			token.SessionID = c.Value
		case strings.HasPrefix(c.Name, "read-"):
			token.Read[c.Name[5:]] = c.Value
		case strings.HasPrefix(c.Name, "write-"):
			token.Write[c.Name[6:]] = c.Value
		}
	}

	return token
}

// ToCookie ...
func (token *AccessToken) ToCookie(expiration time.Duration) *http.Cookie {
	if len(token.SessionID) > 0 {
		return &http.Cookie{
			Name:    "session",
			Value:   token.SessionID,
			Path:    "/",
			Expires: time.Now().Add(expiration),
		}
	}
	for read, value := range token.Read {
		return &http.Cookie{
			Name:    fmt.Sprintf("read-%s", read),
			Value:   value,
			Path:    "/",
			Expires: time.Now().Add(expiration),
		}
	}
	for write, value := range token.Write {
		return &http.Cookie{
			Name:    fmt.Sprintf("write-%s", write),
			Value:   value,
			Path:    "/",
			Expires: time.Now().Add(expiration),
		}
	}
	return nil
}

// AccessTokenFromCookies ...
func (api API) AccessTokenFromCookies(cookies []*http.Cookie) *AccessToken {
	token := new(AccessToken).fromCookies(cookies)
	if api.db != nil && len(token.SessionID) > 0 {
		libToken := token.toLib()
		api.db.FillSessionToken(token.SessionID, libToken)
		token.fromLib(libToken)
	}
	return token
}

// AccessTokenFromRequest ...
func (api API) AccessTokenFromRequest(r *http.Request) *AccessToken {
	token := api.AccessTokenFromCookies(r.Cookies())
	token.IP = reqip.GetClientIP(r)
	return token
}

func (token *AccessToken) toLib() *internal.AccessToken {
	return &internal.AccessToken{
		Read:  token.Read,
		Write: token.Write,
	}
}

func (token *AccessToken) fromLib(libtoken *internal.AccessToken) *AccessToken {
	if libtoken == nil {
		return token
	}
	token.Read = libtoken.Read
	token.Write = libtoken.Write
	return token
}
