package beepboop

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/mo7zayed/reqip"
	"github.com/mssola/user_agent"
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
}

func (r *PageRequest) logRequest() {
	ip := reqip.GetClientIP(r.Request)
	if len(ip) == 0 {
		ip, _, _ = net.SplitHostPort(r.Request.RemoteAddr)
	}

	hostnames, _ := net.LookupAddr(ip)

	ua := user_agent.New(r.Request.UserAgent())
	browser, ver := ua.Browser()

	logmsg := fmt.Sprintf("[%s]: %s %s\n - IP: %s\n - hostnames: %s\n - browser: %s",
		r.RequestID,
		r.Request.Method,
		r.Request.URL.Path,
		ip,
		strings.Join(hostnames, ", "),
		fmt.Sprintf("%s %s %s", ua.OS(), browser, ver))

	loc, _ := r.Context.GeoIPClient.GetLocation(context.Background(), ip)
	if loc != nil {
		logmsg += "\n - location: " + loc.String()
	}

	session, _ := r.Request.Cookie("session")
	if session != nil {
		logmsg += "\n - sessionID: " + session.Value
	}

	r.Context.Logger.Print(logmsg)
}

// Log ...
func (r *PageRequest) Log(a ...interface{}) {
	prefix := fmt.Sprintf("[%s] ", r.RequestID)
	r.Context.Logger.Output(2, prefix+fmt.Sprint(a...))
}

// Logf ...
func (r *PageRequest) Logf(format string, a ...interface{}) {
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
