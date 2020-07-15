package page

import (
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/razzie/razbox"
	"github.com/razzie/razlink"
)

// GetContentTemplate returns the content template for a page
func GetContentTemplate(page string) string {
	t, err := ioutil.ReadFile(fmt.Sprintf("web/template/%s.template", page))
	if err != nil {
		panic(err)
	}
	return string(t)
}

// HandleError ...
func HandleError(r *http.Request, err error) *razlink.View {
	switch err := err.(type) {
	case *razbox.ErrNoReadAccess:
		return razlink.RedirectView(r,
			fmt.Sprintf("/read-auth/%s?r=%s", err.Folder, r.URL.RequestURI()),
			razlink.WithError(err, http.StatusUnauthorized))
	case *razbox.ErrNoWriteAccess:
		return razlink.RedirectView(r,
			fmt.Sprintf("/write-auth/%s?r=%s", err.Folder, r.URL.RequestURI()),
			razlink.WithError(err, http.StatusUnauthorized))
	default:
		return razlink.ErrorView(r, err.Error(), http.StatusInternalServerError)
	}
}

// ServeFileAsync ...
func ServeFileAsync(r *http.Request, file *razbox.FileReader) *razlink.View {
	return razlink.HandlerView(r, func(w http.ResponseWriter, r *http.Request) {
		serveFile(w, r, file)
	})
}

// ServeFileAsAttachmentAsync ...
func ServeFileAsAttachmentAsync(r *http.Request, file *razbox.FileReader) *razlink.View {
	return razlink.HandlerView(r, func(w http.ResponseWriter, r *http.Request) {
		defer file.Close()
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Name))
		w.Header().Set("Content-Type", file.MIME)
		fi, _ := file.Stat()
		if fi != nil {
			w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		}
		io.Copy(w, file)
	})
}

// ServeThumbnail ...
func ServeThumbnail(thumb *razbox.Thumbnail) *razlink.View {
	return razlink.HandlerView(nil, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", thumb.MIME)
		w.Header().Set("Content-Length", strconv.Itoa(len(thumb.Data)))
		w.Write(thumb.Data)
	})
}

// credit: https://github.com/rb-de0/go-mp4-stream/
func serveFile(w http.ResponseWriter, r *http.Request, file *razbox.FileReader) {
	const BUFSIZE = 1024 * 8
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		w.WriteHeader(500)
		return
	}

	fileSize := int(fi.Size())

	if len(r.Header.Get("Range")) == 0 {
		contentLength := strconv.Itoa(fileSize)
		contentEnd := strconv.Itoa(fileSize - 1)

		w.Header().Set("Content-Type", file.MIME)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", contentLength)
		w.Header().Set("Content-Range", "bytes 0-"+contentEnd+"/"+contentLength)
		w.WriteHeader(200)

		buffer := make([]byte, BUFSIZE)

		for {
			n, err := file.Read(buffer)
			if n == 0 {
				break
			}
			if err != nil {
				break
			}

			data := buffer[:n]
			w.Write(data)
			w.(http.Flusher).Flush()
		}
	} else {
		rangeParam := strings.Split(r.Header.Get("Range"), "=")[1]
		splitParams := strings.Split(rangeParam, "-")

		// response values
		contentStartValue := 0
		contentStart := strconv.Itoa(contentStartValue)
		contentEndValue := fileSize - 1
		contentEnd := strconv.Itoa(contentEndValue)
		contentSize := strconv.Itoa(fileSize)

		if len(splitParams) > 0 {
			contentStartValue, err = strconv.Atoi(splitParams[0])
			if err != nil {
				contentStartValue = 0
			}
			contentStart = strconv.Itoa(contentStartValue)
		}

		if len(splitParams) > 1 {
			contentEndValue, err = strconv.Atoi(splitParams[1])
			if err != nil {
				contentEndValue = fileSize - 1
			}
			contentEnd = strconv.Itoa(contentEndValue)
		}

		contentLength := strconv.Itoa(contentEndValue - contentStartValue + 1)

		w.Header().Set("Content-Type", file.MIME)
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", contentLength)
		w.Header().Set("Content-Range", "bytes "+contentStart+"-"+contentEnd+"/"+contentSize)
		w.WriteHeader(206)

		buffer := make([]byte, BUFSIZE)

		file.Seek(int64(contentStartValue), 0)

		writeBytes := 0

		for {
			n, err := file.Read(buffer)
			writeBytes += n
			if n == 0 {
				break
			}
			if err != nil {
				break
			}

			if writeBytes >= contentEndValue {
				data := buffer[:BUFSIZE-writeBytes+contentEndValue+1]
				w.Write(data)
				w.(http.Flusher).Flush()
				break
			}

			data := buffer[:n]
			w.Write(data)
			w.(http.Flusher).Flush()
		}
	}
}

func s(x float64) string {
	if int(x) == 1 {
		return ""
	}
	return "s"
}

// TimeElapsed returns the elapsed time in human readable format (such as "5 days ago")
func TimeElapsed(now time.Time, then time.Time, full bool) string {
	var parts []string
	var text string

	year2, month2, day2 := now.Date()
	hour2, minute2, second2 := now.Clock()

	year1, month1, day1 := then.Date()
	hour1, minute1, second1 := then.Clock()

	year := math.Abs(float64(int(year2 - year1)))
	month := math.Abs(float64(int(month2 - month1)))
	day := math.Abs(float64(int(day2 - day1)))
	hour := math.Abs(float64(int(hour2 - hour1)))
	minute := math.Abs(float64(int(minute2 - minute1)))
	second := math.Abs(float64(int(second2 - second1)))

	week := math.Floor(day / 7)

	if year > 0 {
		parts = append(parts, strconv.Itoa(int(year))+" year"+s(year))
	}

	if month > 0 {
		parts = append(parts, strconv.Itoa(int(month))+" month"+s(month))
	}

	if week > 0 {
		parts = append(parts, strconv.Itoa(int(week))+" week"+s(week))
	}

	if day > 0 {
		parts = append(parts, strconv.Itoa(int(day))+" day"+s(day))
	}

	if hour > 0 {
		parts = append(parts, strconv.Itoa(int(hour))+" hour"+s(hour))
	}

	if minute > 0 {
		parts = append(parts, strconv.Itoa(int(minute))+" minute"+s(minute))
	}

	if second > 0 {
		parts = append(parts, strconv.Itoa(int(second))+" second"+s(second))
	}

	if now.After(then) {
		text = " ago"
	} else {
		text = " after"
	}

	if len(parts) == 0 {
		return "just now"
	}

	if full {
		return strings.Join(parts, ", ") + text
	}
	return parts[0] + text
}
