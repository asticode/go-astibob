package astibob

import (
	"encoding/json"
	"net/http"

	"github.com/asticode/go-astilog"
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
