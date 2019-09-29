package astibob

import (
	"encoding/json"
	"net/http"

	"mime"
	"path/filepath"

	"github.com/asticode/go-astilog"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

func WriteHTTPError(rw http.ResponseWriter, code int, err error) {
	rw.WriteHeader(code)
	astilog.Error(err)
	if err := json.NewEncoder(rw).Encode(Error{Message: err.Error()}); err != nil {
		astilog.Error(errors.Wrap(err, "astibob: marshaling failed"))
	}
}

func WriteHTTPData(rw http.ResponseWriter, data interface{}) {
	if err := json.NewEncoder(rw).Encode(data); err != nil {
		WriteHTTPError(rw, http.StatusInternalServerError, errors.Wrap(err, "astibob: json encoding failed"))
		return
	}
}

func ContentHandle(path string, c []byte) httprouter.Handle {
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
			astilog.Error(errors.Wrapf(err, "astibob: writing %s failed", r.URL.Path))
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
