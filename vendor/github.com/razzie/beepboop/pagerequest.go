package beepboop

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/mssola/user_agent"
	"github.com/razzie/reqip"
)

// PageRequest ...
type PageRequest struct {
	Context   *Context
	Request   *http.Request
	RequestID string
	RelPath   string
	RelURI    string
	IsAPI     bool
	Title     string
	renderer  LayoutRenderer
	logged    bool
}

func (r *PageRequest) logRequest() {
	ip := reqip.GetClientIP(r.Request)
	ua := user_agent.New(r.Request.UserAgent())
	browser, ver := ua.Browser()

	logmsg := fmt.Sprintf("[%s]: %s %s\n • %s, %s %s %s",
		r.RequestID, r.Request.Method, r.Request.RequestURI,
		ip, ua.OS(), browser, ver)

	var hasLocation bool
	if r.Context.GeoIPClient != nil {
		loc, _ := r.Context.GeoIPClient.GetLocation(context.Background(), ip)
		if loc != nil {
			hasLocation = true
			logmsg += ", " + loc.String()
		}
	}
	if !hasLocation {
		hostnames, _ := net.LookupAddr(ip)
		logmsg += ", " + strings.Join(hostnames, ", ")
	}

	session, _ := r.Request.Cookie("session")
	if session != nil {
		logmsg += ", session: " + session.Value
	}

	r.Context.Logger.Print(logmsg)
	r.logged = true
}

func (r *PageRequest) logRequestNonblocking() {
	r.logged = true
	go r.logRequest()
}

// Log ...
func (r *PageRequest) Log(a ...interface{}) {
	if !r.logged {
		r.logRequestNonblocking()
	}
	prefix := fmt.Sprintf("[%s] ", r.RequestID)
	r.Context.Logger.Output(2, prefix+fmt.Sprint(a...))
}

// Logf ...
func (r *PageRequest) Logf(format string, a ...interface{}) {
	if !r.logged {
		r.logRequestNonblocking()
	}
	prefix := fmt.Sprintf("[%s] ", r.RequestID)
	r.Context.Logger.Output(2, prefix+fmt.Sprintf(format, a...))
}

// Respond returns the default page response View
func (r *PageRequest) Respond(data interface{}, opts ...ViewOption) *View {
	v := &View{
		StatusCode: http.StatusOK,
		Data:       data,
	}
	for _, opt := range opts {
		opt(v)
	}
	v.renderer = func(w http.ResponseWriter) {
		r.renderer(w, r.Request, r.Title, data, v.StatusCode)
	}
	return v
}

// AccessToken returns an AccessToken from this page request
func (r *PageRequest) AccessToken() *AccessToken {
	return NewAccessTokenFromRequest(r)
}
