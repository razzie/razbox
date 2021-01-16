package internal

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/gabriel-vasile/mimetype"
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
	algorithm.Write([]byte(path.Clean(filename)))
	bytes := algorithm.Sum(nil)
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant is 10
	return uuid.Must(uuid.FromBytes(bytes)).String()
}

// IsFolder returns whether a relative path is a folder
func IsFolder(root, relPath string) bool {
	fi, err := os.Stat(path.Join(root, relPath))
	if err != nil {
		return false
	}

	return fi.IsDir()
}

// DetectContentType determines the MIME type of a given file
func DetectContentType(r io.ReadSeeker) (string, error) {
	r.Seek(0, io.SeekStart)
	mime, err := mimetype.DetectReader(r)
	if err != nil {
		return mime.String(), err
	}

	if mime.String() == "application/octet-stream" {
		var header [512]byte
		r.Seek(0, io.SeekStart)
		_, _ = r.Read(header[:])
		return http.DetectContentType(header[:]), nil
	}

	return mime.String(), nil
}
