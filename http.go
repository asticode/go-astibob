package astibob

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/asticode/go-astikit"
	"github.com/julienschmidt/httprouter"
)

func WriteHTTPError(l astikit.SeverityLogger, rw http.ResponseWriter, code int, err error) {
	rw.WriteHeader(code)
	l.Error(err)
	if err := json.NewEncoder(rw).Encode(Error{Message: err.Error()}); err != nil {
		l.Error(fmt.Errorf("astibob: marshaling failed: %w", err))
	}
}

func WriteHTTPData(l astikit.SeverityLogger, rw http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(rw).Encode(data); err != nil {
		WriteHTTPError(l, rw, http.StatusInternalServerError, fmt.Errorf("astibob: json encoding failed: %w", err))
		return
	}
}

func ContentHandle(path string, c []byte, l astikit.SeverityLogger) httprouter.Handle {
	// Get mime type
	t := mime.TypeByExtension(filepath.Ext(path))
	if t == "" {
		t = "binary"
	}
	return func(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
		// Set content type
		rw.Header().Set("Content-Type", t)

		// Write
		if _, err := rw.Write(c); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			l.Error(fmt.Errorf("astibob: writing %s failed: %w", r.URL.Path, err))
			return
		}
	}
}

func DirHandle(path string) httprouter.Handle {
	return func(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
		req.URL.Path = ps.ByName("path")
		http.FileServer(http.Dir(path)).ServeHTTP(w, req)
	}
}
