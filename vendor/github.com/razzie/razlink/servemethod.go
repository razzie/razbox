package razlink

import (
	"context"
	"net/http"
	"time"
)

// ServeMethod defines how to serve a request
type ServeMethod int

// Available serve methods
const (
	Proxy ServeMethod = iota
	Embed
	Redirect
	Track
)

// GetServeMethodForURL tries to determine the best possible serve method for an url
func GetServeMethodForURL(ctx context.Context, url string, timeout time.Duration) (ServeMethod, error) {
	if url == "." {
		return Track, nil
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, _ := http.NewRequest("GET", url, nil)
	resp, err := http.DefaultClient.Do(req.WithContext(timeoutCtx))
	if err != nil {
		return Redirect, err
	}

	defer resp.Body.Close()

	return GetServeMethodFromHeader(resp.Header), nil
}

// GetServeMethodFromHeader tries to determine the best possible serve method from a http response header
func GetServeMethodFromHeader(header http.Header) ServeMethod {
	if HasContentType(header, "text/html") {
		if len(header.Get("X-Frame-Options")) > 0 {
			return Redirect
		}

		return Embed
	}

	return Proxy
}
