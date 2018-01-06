package astibob

import (
	"fmt"

	"github.com/asticode/go-astibob/brain"
	"github.com/pkg/errors"
)

// Cmd represents a cmd
type Cmd struct {
	AbilityName string
	EventName   string
	Payload     interface{}
}

// Exec executes a cmd
func (b *Bob) Exec(cmd *Cmd) error {
	return b.ExecOnBrain(cmd, "")
}

// ExecOnBrain executes a cmd on a specific brain
func (b *Bob) ExecOnBrain(cmd *Cmd, brainName string) (err error) {
	// Fetch brain
	var brn *brain
	if brn, err = b.brainForExec(cmd, brainName); err != nil {
		err = errors.Wrapf(err, "astibob: fetching brain for cmd %+v failed", *cmd)
		return
	}

	// Write
	eventName := astibrain.WebsocketAbilityEventName(cmd.AbilityName, cmd.EventName)
	if err = brn.ws.Write(eventName, cmd.Payload); err != nil {
		err = errors.Wrapf(err, "astibob: writing event %s with payload %#v failed", eventName, cmd.Payload)
		return
	}
	return
}

// brainForExec fetches the proper brain for Exec
func (b *Bob) brainForExec(cmd *Cmd, brainName string) (brn *brain, err error) {
	// No ability name specified
	if len(cmd.AbilityName) == 0 {
		err = fmt.Errorf("astibob: no ability name specified in cmd %+v", *cmd)
		return
	}

	// Brain name is specified
	var a *ability
	var ok bool
	if len(brainName) > 0 {
		// Retrieve brain
		if brn, ok = b.brains.brain(brainName); !ok {
			err = fmt.Errorf("astibob: unknown brain %s", brainName)
			return
		}

		// Retrieve ability
		if a, ok = brn.ability(cmd.AbilityName); !ok {
			err = fmt.Errorf("astibob: unknown ability %s in brain %s", cmd.AbilityName, brainName)
			return
		}

		// Check ability status
		if !a.isOn() {
			err = fmt.Errorf("astibob: ability %s in brain %s is not on", cmd.AbilityName, brainName)
			return
		}
		return
	}

	// Loop through brains
	b.brains.brains(func(ib *brain) error {
		// Retrieve ability
		if a, ok = ib.ability(cmd.AbilityName); ok && a.isOn() {
			brn = ib
			return errors.New("astibob: dummy")
		}
		return nil
	})

	// No brain found
	if brn == nil {
		err = fmt.Errorf("astibob: no brain found for ability %s", cmd.AbilityName)
		return
	}
	return
}
