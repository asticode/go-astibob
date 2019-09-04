package index

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"text/template"

	"path/filepath"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

func (i *Index) updateRunnableStatus(m *astibob.Message) (err error) {
	// Check worker
	if m.From.Worker == nil {
		err = errors.New("index: no from worker")
		return
	}

	// Get worker
	i.mw.Lock()
	w, ok := i.ws[*m.From.Worker]
	i.mw.Unlock()

	// No worker
	if !ok {
		err = fmt.Errorf("index: worker %s doesn't exist", *m.From.Worker)
		return
	}

	// Check runnable
	if m.From.Name == nil {
		err = errors.New("index: no from name")
		return
	}

	// Get runnable
	w.mr.Lock()
	r, ok := w.rs[*m.From.Name]
	w.mr.Unlock()

	// No runnable
	if !ok {
		err = fmt.Errorf("index: runnable %s doesn't exist", *m.From.Name)
		return
	}

	// Update status
	if m.Name == astibob.RunnableStartedMessage {
		r.Status = astibob.RunningStatus
	} else {
		r.Status = astibob.StoppedStatus
	}

	// Update runnable
	w.rs[*m.From.Name] = r
	return
}

func (i *Index) sendRequestToRunnable(rw http.ResponseWriter, p httprouter.Params, method, path string, body io.Reader, header http.Header, fn func(worker, runnable, url string, resp *http.Response)) {
	// Unescape worker
	worker, err := url.QueryUnescape(p.ByName("worker"))
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		astilog.Error(errors.Wrap(err, "index: unescaping worker failed"))
		return
	}

	// Get worker
	i.mw.Lock()
	w, ok := i.ws[worker]
	i.mw.Unlock()

	// No worker
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Unescape runnable
	runnable, err := url.QueryUnescape(p.ByName("runnable"))
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		astilog.Error(errors.Wrap(err, "index: unescaping runnable failed"))
		return
	}

	// Get runnable
	w.mr.Lock()
	_, ok = w.rs[runnable]
	w.mr.Unlock()

	// No runnable
	if !ok {
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	// Create url
	u := w.addr + "/" + filepath.Join("runnables", runnable, path)

	// Create request
	r, err := http.NewRequest(method, u, body)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		astilog.Error(errors.Wrapf(err, "index: creating %s request to %s failed", method, u))
		return
	}

	// Add headers
	if header != nil {
		for k := range header {
			r.Header.Set(k, header.Get(k))
		}
	}

	// Log
	astilog.Debugf("index: sending %s request to %s", method, u)

	// Send request
	var resp *http.Response
	if resp, err = i.c.Do(r); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		astilog.Error(errors.Wrapf(err, "index: doing %s request to %s failed", method, u))
		return
	}
	defer resp.Body.Close()

	// Custom
	fn(worker, runnable, u, resp)
}

func (i *Index) runnableRoutes(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	i.sendRequestToRunnable(
		rw,
		p,
		r.Method,
		"/routes"+p.ByName("path"),
		r.Body,
		r.Header,
		func(_, _, url string, resp *http.Response) {
			// Copy header
			for k := range resp.Header {
				rw.Header().Set(k, resp.Header.Get(k))
			}

			// Copy status code
			rw.WriteHeader(resp.StatusCode)

			// Copy body
			if _, err := io.Copy(rw, resp.Body); err != nil {
				astilog.Error(errors.Wrapf(err, "index: copying response body of %s failed", url))
				return
			}
			return
		},
	)
}

type TemplateData struct {
	Runnable string
	Worker   string
}

func (i *Index) runnableWeb(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	// We need the template from the runnable, not the executed result itself.
	// Indeed in order to execute the template we need the layouts, which the workers don't have.
	i.sendRequestToRunnable(
		rw,
		p,
		http.MethodGet,
		"/templates"+p.ByName("path"),
		nil,
		nil,
		func(worker, runnable, url string, resp *http.Response) {
			// Read body
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				astilog.Error(errors.Wrapf(err, "index: reading body of %s failed", url))
				return
			}

			// Parse template
			var t *template.Template
			if t, err = i.t.Parse(string(b)); err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				astilog.Error(errors.Wrapf(err, "index: parsing template %s failed", url))
				return
			}

			// Set content type
			rw.Header().Set("Content-Type", "text/html; charset=UTF-8")

			// Execute template
			if err = t.Execute(rw, TemplateData{
				Runnable: runnable,
				Worker:   worker,
			}); err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				astilog.Error(errors.Wrapf(err, "index: executing template %s failed", url))
				return
			}
		},
	)
}
