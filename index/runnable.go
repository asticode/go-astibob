package index

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"text/template"

	"github.com/asticode/go-astibob"
	"github.com/julienschmidt/httprouter"
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
		i.l.Error(fmt.Errorf("index: unescaping worker failed: %w", err))
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
		i.l.Error(fmt.Errorf("index: unescaping runnable failed: %w", err))
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
		i.l.Error(fmt.Errorf("index: creating %s request to %s failed: %w", method, u, err))
		return
	}

	// Add headers
	if header != nil {
		for k := range header {
			r.Header.Set(k, header.Get(k))
		}
	}

	// Log
	i.l.Debugf("index: sending %s request to %s", method, u)

	// Send request
	var resp *http.Response
	if resp, err = i.c.Do(r); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		i.l.Error(fmt.Errorf("index: doing %s request to %s failed: %w", method, u, err))
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
				i.l.Error(fmt.Errorf("index: copying response body of %s failed: %w", url, err))
				return
			}
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
				i.l.Error(fmt.Errorf("index: reading body of %s failed: %w", url, err))
				return
			}

			// Parse template
			var t *template.Template
			if t, err = i.t.Parse(string(b)); err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				i.l.Error(fmt.Errorf("index: parsing template %s failed: %w", url, err))
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
				i.l.Error(fmt.Errorf("index: executing template %s failed: %w", url, err))
				return
			}
		},
	)
}
