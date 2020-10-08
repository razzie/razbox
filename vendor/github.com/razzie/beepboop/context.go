package beepboop

import (
	"context"
	"log"

	"github.com/razzie/geoip-server/geoip"
)

// Context ...
type Context struct {
	Context     context.Context
	DB          *DB
	Logger      *log.Logger
	GeoIPClient geoip.Client
}

// ContextGetter ...
type ContextGetter func(context.Context) *Context
