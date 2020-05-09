package razbox

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Salt returns a random salt
func Salt() string {
	return strconv.FormatInt(rand.Int63(), 36)
}

// Hash returns the SHA1 hash of a string
func Hash(s string) string {
	algorithm := sha1.New()
	algorithm.Write([]byte(s))
	return hex.EncodeToString(algorithm.Sum(nil))
}

// FilenameToUUID returns an UUID from a filename
func FilenameToUUID(filename string) string {
	algorithm := md5.New()
	algorithm.Write([]byte(filename))
	bytes := algorithm.Sum(nil)
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant is 10
	return uuid.Must(uuid.FromBytes(bytes)).String()
}

// ByteCountSI ...
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// ByteCountIEC ...
func ByteCountIEC(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB",
		float64(b)/float64(div), "KMGTPE"[exp])
}

// IsFolder returns whether a relative path is a folder
func IsFolder(dir string) bool {
	fi, err := os.Stat(path.Join(Root, dir))
	if err != nil {
		return false
	}

	return fi.IsDir()
}

// RemoveTrailingSlash removes the trailing slash from a path
func RemoveTrailingSlash(path string) string {
	if len(path) > 0 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	return path
}

// MIMEtoSymbol returns a symbol that represents the MIME type
func MIMEtoSymbol(mime string) template.HTML {
	t := strings.SplitN(mime, "/", 2)
	switch t[0] {
	case "application":
		if len(t) < 2 {
			break
		}
		switch t[1] {
		case "zip", "x-7z-compressed", "x-rar-compressed", "x-tar", "tar+gzip", "gzip", "x-bzip", "x-bzip2":
			return "&#128230;"
		case "vnd.microsoft.portable-executable", "vnd.debian.binary-package", "jar", "x-rpm":
			return "&#128187;"
		case "pdf", "msword", "vnd.openxmlformats-officedocument.wordprocessingml.document", "x-mobipocket-ebook", "epub+zip":
			return "&#128209;"
		case "x-iso9660-image", "x-cd-image", "x-raw-disk-image":
			return "&#128191;"
		case "vnd.ms-excel", "vnd.ms-powerpoint", "vnd.openxmlformats-officedocument.presentationml.presentation":
			return "&#128200;"
		}
	case "audio":
		return "&#127925;"
	case "font":
		return "&#9000;"
	case "image":
		return "&#127912;"
	case "model":
		return "&#127922;"
	case "text":
		return "&#128209;"
	case "video":
		return "&#127916;"
	}

	return "&#128196;"
}
