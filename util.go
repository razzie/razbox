package razbox

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"

	"github.com/asaskevich/govalidator"
)

type limitedReader struct {
	r io.Reader
	n int64
}

func (r *limitedReader) Read(p []byte) (n int, err error) {
	if int64(len(p)) > r.n {
		p = p[:r.n]
	}
	n, err = r.r.Read(p)
	r.n -= int64(n)
	if r.n == 0 && err == nil {
		err = fmt.Errorf("limit exceeded")
	}
	return
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
