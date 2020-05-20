package razlink

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// InstanceID is a unique ID for this instance (to be prepended to entry IDs)
var InstanceID = newInstanceID()

func newInstanceID() string {
	i := uint16(time.Now().UnixNano())
	return strconv.FormatInt(int64(i), 36)
}

// NewID returns a new (hopefully unique) ID for entries
func NewID() string {
	return InstanceID + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}

// Hash returns the SHA1 hash of a string
func Hash(s string) string {
	algorithm := sha1.New()
	algorithm.Write([]byte(s))
	return hex.EncodeToString(algorithm.Sum(nil))
}

func getPrivateIPBlocks() (blocks []*net.IPNet) {
	// https://stackoverflow.com/a/50825191
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		blocks = append(blocks, block)
	}
	return
}

var privateIPBlocks = getPrivateIPBlocks()

// IsPrivateIP returns true if the given IP address belongs to private network space
func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// HasContentType determines whether the request `content-type` includes a
// server-acceptable mime-type
func HasContentType(header http.Header, mimetype string) bool {
	contentType := header.Get("Content-type")
	if contentType == "" {
		return mimetype == "application/octet-stream"
	}

	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			break
		}
		if t == mimetype {
			return true
		}
	}
	return false
}

var pixel, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=")

// WritePixel writes a transparent pixel to a http.ResponseWriter
func WritePixel(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(pixel)))
	_, _ = w.Write(pixel)
}

var favicon, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAACAAAAAjCAYAAAD17ghaAAAABHNCSVQICAgIfAhkiAAAAAlwSFlzAAAa" +
	"PwAAGj8BlYhsxgAAABl0RVh0U29mdHdhcmUAd3d3Lmlua3NjYXBlLm9yZ5vuPBoAAAIBSURBVFiFvddN" +
	"iE5xFMfxz4yMtyjvY/KShYXkrZGdl7L1slCKZCNLWShkIUoWZCcLJRpKlI0dpfEyLO3EGCShKDVjmqkZ" +
	"Mz0Wf0+e7nNn/Kfn/59TZ3Pv6Xx/955zz/9c8to8XEAPhvEdd7AhMxeswidUSnwYh3LCp+L1GPBaERtz" +
	"Cdj3H3jV7zZnErA9Ni6XgFmRcXNyCJiJuZGx71KCm3AAn8XVv4KTqeCb8GIC4ApeYUaj4CW4gdFC8p84" +
	"KjzhQAn8AeY3Ap6GU/hVSPwbVwrJF+EwzuO4BFNwF96rf6rHWNdI4hYcxHXcwyVsrrm/9i+kCO7BnkbA" +
	"sBrdJckrQo2vYqRwvQ8nhHI0ZAvxZQx4mY8Kb6m1UXDVLk8A/hztqcBV+xAJ7xCGTVJrxtIJxFdyCPgR" +
	"GdufGl61a+Kb7ybaUgtYKXxSsY3Yj9OYnlLENmF+lwGf4FHJ9Y/Ym1LEApxBp3BS3cd+oU9gtzD1ikI6" +
	"TdKWS1g2j6FXfX90YPFkCWkVpmHxGO4VTryWQvwybMEa/95oEmtHl/qydGMn1uNZ4d5XHEkpoknolbJV" +
	"rHiA1frFlCIIy+g5DI4DLfrW1CJgOR5GCrg9JYOAPqHpdsQE5/oxGYmMG8ol4GVkXFcmPnhq/PoPYEVO" +
	"AW14OwZ8UIJFNsZm4yzeYAjfcEtYgsEfmmcOvhtYrlIAAAAASUVORK5CYII=")

// WriteFavicon writes Razlink favicon to a http.ResponseWriter
func WriteFavicon(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(favicon)))
	_, _ = w.Write(favicon)
}

// GetBase returns the base target for relative URLs
func GetBase(r *http.Request) string {
	slashes := strings.Count(r.URL.Path, "/")
	if slashes > 1 {
		return strings.Repeat("../", slashes-1)
	}
	return "/"
}

// GetShorthandPath returns a shorthand version of a path in case it's too long
func GetShorthandPath(path string) string {
	if len(path) > 32 {
		return path[:15] + ".." + path[len(path)-15:]
	}
	return path
}
