package razlink

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

var errViewRenderer, _ = BindLayout("<strong>{{.}}</strong>")

// ErrorView returns a PageView that represents an error
func ErrorView(r *http.Request, errmsg string, errcode int) PageView {
	return func(w http.ResponseWriter) {
		//http.Error(w, errmsg, errcode)
		w.WriteHeader(errcode)
		errViewRenderer(w, r, errmsg, errmsg)
	}
}

// EmbedView returns a PageView that embeds the given website
func EmbedView(url string) PageView {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<iframe src="%s" style="position:fixed; top:0; left:0; bottom:0; right:0; width:100%%; height:100%%; border:none; margin:0; padding:0; overflow:hidden; z-index:999999;"></iframe>`, url)
	}
}

// RedirectView ...
func RedirectView(r *http.Request, url string) PageView {
	return func(w http.ResponseWriter) {
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

// CookieAndRedirectView ...
func CookieAndRedirectView(r *http.Request, cookie *http.Cookie, url string) PageView {
	return func(w http.ResponseWriter) {
		http.SetCookie(w, cookie)
		http.Redirect(w, r, url, http.StatusSeeOther)
	}
}

// CopyView ...
func CopyView(resp *http.Response) PageView {
	bytes, _ := ioutil.ReadAll(resp.Body)

	return func(w http.ResponseWriter) {
		for k, v := range resp.Header {
			w.Header().Set(k, v[0])
		}
		w.WriteHeader(resp.StatusCode)
		//io.Copy(w, resp.Body)
		w.Write(bytes)
	}
}
