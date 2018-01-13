package astibob

// Event names
const (
	EventNameAbilityStarted    = "ability.started"
	EventNameAbilityStopped    = "ability.stopped"
	EventNameBrainDisconnected = "brain.disconnected"
	EventNameBrainRegistered   = "brain.registered"
	EventNameReady             = "ready"
)

// Event represents an event
type Event struct {
	Ability *EventAbility
	Brain   *EventBrain
	Name    string
}

// EventBob represents a Bob event.
type EventBob struct {
	Brains []*EventBrain `json:"brains,omitempty"`
}

// newEventBob create a new Bob event
func newEventBob(brains *brains) (e EventBob) {
	// Init
	e = EventBob{}

	// Loop through brains
	brains.brains(func(b *brain) error {
		e.Brains = append(e.Brains, newEventBrain(b))
		return nil
	})
	return
}

// EventBrain represents a brain event.
type EventBrain struct {
	Abilities []*EventAbility `json:"abilities,omitempty"`
	Name      string          `json:"name"`
}

// newEventBrain creates a new brain event
func newEventBrain(b *brain) (o *EventBrain) {
	// Create Event brain
	o = &EventBrain{
		Name: b.name,
	}

	// Loop through abilities
	b.abilities(func(a *ability) error {
		o.Abilities = append(o.Abilities, newEventAbility(a))
		return nil
	})
	return
}

// EventAbility represents an ability event.
type EventAbility struct {
	BrainName   string `json:"brain_name,omitempty"`
	Description string `json:"description"`
	IsOn        bool   `json:"is_on"`
	Name        string `json:"name"`
	WebHomepage    string `json:"web_homepage,omitempty"`
}

// newEventAbility creates a new ability event
func newEventAbility(a *ability) *EventAbility {
	return &EventAbility{
		Description: a.description,
		IsOn:        a.isOn(),
		Name:        a.name,
		WebHomepage:    a.webHomepage,
	}
}
