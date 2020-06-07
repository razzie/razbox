package razbox

import (
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/asaskevich/govalidator"
)

// LimitedReader is like io.LimitedReader, but returns a non-EOF error if limit is exceeded
type LimitedReader struct {
	R io.Reader
	N int64
}

// Read implements io.Reader
func (l *LimitedReader) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, &ErrSizeLimitExceeded{}
	}
	if int64(len(p)) > l.N {
		p = p[:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

// LimitedReadCloser ...
type LimitedReadCloser struct {
	R io.ReadCloser
	N int64
}

// Read implements io.Reader
func (l *LimitedReadCloser) Read(p []byte) (n int, err error) {
	if l.N <= 0 {
		return 0, &ErrSizeLimitExceeded{}
	}
	if int64(len(p)) > l.N {
		p = p[:l.N]
	}
	n, err = l.R.Read(p)
	l.N -= int64(n)
	return
}

// Close implements io.Closer
func (l *LimitedReadCloser) Close() error {
	return l.R.Close()
}

func getContentDispositionFilename(header http.Header) string {
	contentDisposition := header.Get("Content-Disposition")
	_, params, _ := mime.ParseMediaType(contentDisposition)
	return params["filename"]
}

func getSafeFilename(filenames ...string) (string, error) {
	var fails []string
	for _, filename := range filenames {
		safe := govalidator.SafeFileName(filename)
		if len(safe) > 0 && safe != "." && safe != ".." {
			return safe, nil
		}
		fails = append(fails, filename)
	}
	return "", &ErrInvalidName{Name: strings.Join(fails, " | ")}
}
