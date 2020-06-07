package razbox

import (
	"io"
	"mime"
	"net/http"
	"path"

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

func getResponseFilename(resp *http.Response) string {
	contentDisposition := resp.Header.Get("Content-Disposition")
	if len(contentDisposition) > 0 {
		_, params, _ := mime.ParseMediaType(contentDisposition)
		filename := govalidator.SafeFileName(params["filename"])
		if len(filename) > 0 && filename != "." {
			return filename
		}
	}
	return govalidator.SafeFileName(path.Base(resp.Request.URL.Path))
}
