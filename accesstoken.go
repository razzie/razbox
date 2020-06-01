package razbox

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/razzie/razbox/internal"
)

// AccessToken ...
type AccessToken struct {
	Read  map[string]string
	Write map[string]string
}

// FromCookies ...
func (token *AccessToken) FromCookies(cookies []*http.Cookie) *AccessToken {
	if token.Read == nil {
		token.Read = make(map[string]string)
	}
	if token.Write == nil {
		token.Write = make(map[string]string)
	}

	for _, c := range cookies {
		switch {
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
	return new(AccessToken).FromCookies(cookies)
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
