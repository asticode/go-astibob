package astispeaking

import (
	"net/http"
	"sync"

	"github.com/asticode/go-astibob"
)

// Interface is the interface of the ability
type Interface struct {
	dispatchFunc astibob.DispatchFunc
	history      []string
	m            sync.Mutex
}

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return name
}

// SetDispatchFunc implements the astibob.Dispatcher interface
func (i *Interface) SetDispatchFunc(fn astibob.DispatchFunc) {
	i.dispatchFunc = fn
}

// addToHistory adds a sentence to the history while keeping it capped
func (i *Interface) addToHistory(s string) {
	// Lock
	i.m.Lock()
	defer i.m.Unlock()

	// Append
	i.history = append(i.history, s)

	// Only keep last 50 items
	if len(i.history) > 50 {
		i.history = i.history[len(i.history)-50:]
	}

	// Dispatch to clients
	if i.dispatchFunc != nil {
		i.dispatchFunc(astibob.ClientEvent{
			Name:    "history",
			Payload: s,
		})
	}
}

// Say creates a say cmd
func (i *Interface) Say(s string) *astibob.Cmd {
	i.addToHistory(s)
	return &astibob.Cmd{
		AbilityName: name,
		EventName:   websocketEventNameSay,
		Payload:     s,
	}
}

// APIHandlers implements the astibob.APIHandle interface
func (i *Interface) APIHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		"/": i.apiHandlerIndex(),
	}
}

// APIBody represents the API body
type APIBody struct {
	History []string `json:"history,omitempty"`
}

// apiHandlerIndex handles the index api request
func (i *Interface) apiHandlerIndex() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		i.m.Lock()
		defer i.m.Unlock()
		astibob.APIWrite(rw, APIBody{
			History: i.history,
		})
	})
}

// WebTemplates implements the astibob.WebTemplater interface
func (i *Interface) WebTemplates() map[string]string {
	return map[string]string{
		"/index": i.webTemplateIndex(),
	}
}

// webTemplateIndex returns the index web template
func (i *Interface) webTemplateIndex() string {
	return `{{ define "title" }}Speaking{{ end }}
{{ define "css" }}{{ end }}
{{ define "html" }}
	<div class="header">History</div>
	<div class="flex" id="history"></div>
{{ end }}
{{ define "js" }}
<script type="text/javascript">
	let speaking = {
		init: function() {
			base.init(speaking.websocketFunc, function(data) {
				// Fetch history
				base.sendHttp(base.abilityAPIPattern("/"), "GET", function(data) {
					// Display history
					if (typeof data.history !== "undefined") {
						for (let idx = 0; idx < data.history.length; idx++) {
							speaking.addHistory(data.history[idx]);
						}
					}

					// Finish
					base.finish();
				}, function() {
					asticode.loader.hide();
				});
			});
		},
		addHistory: function(history) {
			let wrapper = $("<div class='panel-wrapper'></div>");
			wrapper.appendTo($("#history"));
			let panel = $("<div class='panel'></div>");
			panel.appendTo(wrapper);
			panel.append(history);
		},
    	websocketFunc: function(event_name, payload) {
			switch (event_name) {
				case base.abilityWebsocketEventName("history"):
					speaking.addHistory(payload);
					break;
			}
		}
	}
	speaking.init();
</script>
{{ end }}
{{ template "base" . }}`
}
