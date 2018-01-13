package astispeaking

import "github.com/asticode/go-astibob"

// Interface is the interface of the ability
type Interface struct{}

// NewInterface creates a new interface
func NewInterface() *Interface {
	return &Interface{}
}

// Name implements the astibob.Interface interface
func (i *Interface) Name() string {
	return Name
}

// UI implements the astibob.UIDisplayer interface
func (i *Interface) UI() *astibob.UI {
	return &astibob.UI{
		Description: "Says words to your audio output using speech synthesis",
		Homepage:    "/index",
		Title:       "Speaking",
		WebTemplates: map[string]string{
			"/index": i.webTemplateIndex(),
		},
	}
}

// Say creates a say cmd
func (i *Interface) Say(s string) *astibob.Cmd {
	return &astibob.Cmd{
		AbilityName: Name,
		EventName:   websocketEventNameSay,
		Payload:     s,
	}
}

// webTemplateIndex returns the index web template
func (i *Interface) webTemplateIndex() string {
	return `{{ define "title" }}Speaking{{ end }}
{{ define "css" }}{{ end }}
{{ define "html" }}
    caca
{{ end }}
{{ define "js" }}
<script type="text/javascript">
	let speaking = {
		init: function() {
			base.init(null, function(data) {
				// Finish
				base.finish();
			});
		}
	}
	speaking.init();
</script>
{{ end }}
{{ template "base" . }}`
}
