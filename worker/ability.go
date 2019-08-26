package worker

import (
	"fmt"

	"github.com/asticode/go-astibob"
	"github.com/asticode/go-astilog"
	"github.com/pkg/errors"
	"net/http"
	"encoding/json"
	"bytes"
)

func (w *Worker) sendMessageToAbility(m *astibob.Message) (err error) {
	// No to worker
	if m.To == nil || m.To.Worker == nil {
		err = errors.Wrap(err, "worker: no to worker")
		return
	}

	// Destination is itself
	if *m.To.Worker == w.name {
		return
	}

	// Log
	astilog.Debugf("worker: sending %s message to ability", m.Name)

	// Get worker
	w.mw.Lock()
	mw, ok := w.ws[*m.To.Worker]
	w.mw.Unlock()

	// No worker
	if !ok {
		err = fmt.Errorf("worker: worker %s doesn't exist", *m.To.Worker)
		return
	}

	// Marshal
	var b []byte
	if b, err = json.Marshal(m); err != nil {
		err = errors.Wrap(err, "worker: marshaling failed")
		return
	}

	// Create request
	var req *http.Request
	if req, err = http.NewRequest(http.MethodPost, fmt.Sprintf("%s/messages", mw.addr), bytes.NewReader(b)); err != nil {
		err = errors.Wrap(err, "worker: creating request failed")
		return
	}

	// Send request
	var resp *http.Response
	if resp, err = w.ch.Do(req); err != nil {
		err = errors.Wrap(err, "worker: sending request failed")
		return
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		// Unmarshal
		// We silence the error since there may not be an error message in the response
		var e astibob.Error
		json.NewDecoder(resp.Body).Decode(&e)

		// Log
		if e.Message != "" {
			err = fmt.Errorf("worker: response error message is %s", e.Message)
		} else {
			err = fmt.Errorf("worker: response status code is %d", resp.StatusCode)
		}
	}
	return
}
