package razbox

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/razzie/razbox/internal"
)

// ErrNoReadAccess ...
type ErrNoReadAccess struct {
	Folder string
}

func (err ErrNoReadAccess) Error() string {
	return err.Folder + ": no read access"
}

// ErrNoWriteAccess ...
type ErrNoWriteAccess struct {
	Folder string
}

func (err ErrNoWriteAccess) Error() string {
	return err.Folder + ": no write access"
}

// AccessToken ...
type AccessToken struct {
	Read  map[string]string
	Write map[string]string
	//Dev   bool
}

// FromCookies ...
func (token *AccessToken) FromCookies(cookies []*http.Cookie) {
	if token.Read == nil {
		token.Read = make(map[string]string)
	}
	if token.Write == nil {
		token.Write = make(map[string]string)
	}

	for _, c := range cookies {
		if strings.HasPrefix(c.Name, "read-") {
			token.Read[c.Name[5:]] = c.Value
		}
		if strings.HasPrefix(c.Name, "write-") {
			token.Write[c.Name[6:]] = c.Value
		}
	}
}

// ToCookie ...
func (token *AccessToken) ToCookie() *http.Cookie {
	for read, value := range token.Read {
		return &http.Cookie{
			Name:    fmt.Sprintf("read-%s", read),
			Value:   value,
			Path:    "/",
			Expires: time.Now().Add(time.Hour * 24 * 7),
		}
	}
	for write, value := range token.Write {
		return &http.Cookie{
			Name:  fmt.Sprintf("write-%s", write),
			Value: value,
			Path:  "/",
		}
	}
	return nil
}

// AccessTokenFromCookies ...
func (api API) AccessTokenFromCookies(cookies []*http.Cookie) *AccessToken {
	accessToken := &AccessToken{}
	accessToken.FromCookies(cookies)
	return accessToken
}

func (token *AccessToken) toLib() *internal.AccessToken {
	return &internal.AccessToken{
		Read:  token.Read,
		Write: token.Write,
	}
}
