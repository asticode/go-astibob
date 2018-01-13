package astispeaking

import (
	"net/http"
	"sync"

	"github.com/asticode/go-astibob"
)

// Interface is the interface of the ability
type Interface struct {
	history []string
	m       sync.Mutex
}

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return Name
}

// addToHistory adds a sentence to the history while keeping it capped
func (i *Interface) addToHistory(s string) {
	i.m.Lock()
	defer i.m.Unlock()
	i.history = append(i.history, s)
	if len(i.history) > 50 {
		i.history = i.history[len(i.history)-50:]
	}
}

// Say creates a say cmd
func (i *Interface) Say(s string) *astibob.Cmd {
	i.addToHistory(s)
	return &astibob.Cmd{
		AbilityName: Name,
		EventName:   websocketEventNameSay,
		Payload:     s,
	}
}

// APIHandlers implements the astibob.APIHandle interface
func (i *Interface) APIHandlers() map[string]http.Handler {
	return map[string]http.Handler{
		"/history": i.apiHandlerHistory(),
	}
}

// APIBody represents the API body
type APIBody struct {
	History []string `json:"history,omitempty"`
}

// apiHandlerHistory handles the history api request
func (i *Interface) apiHandlerHistory() http.Handler {
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
{{ define "html" }}{{ end }}
{{ define "js" }}
<script type="text/javascript">
	let speaking = {
		init: function() {
			base.init(null, function(data) {
				// Fetch history
				base.sendHttp(base.apiPattern("/history"), "GET", function(data) {
					// Display history
					$("#content").append("<div class='header'>History</div>");
					speaking.flex = $("<div class='flex'></div>");
					speaking.flex.appendTo($("#content"));
					if (typeof data.history !== "undefined" && data.history.length > 0) {
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
			wrapper.appendTo(speaking.flex);
			let panel = $("<div class='panel'></div>");
			panel.appendTo(wrapper);
			panel.append(history);
		}
	}
	speaking.init();
</script>
{{ end }}
{{ template "base" . }}`
}
