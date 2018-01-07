package astibrain

// dispatch dispatches an event to Bob
func (b *Brain) dispatch(e Event) {
	b.d.Do(func() {
		// Send
		b.ws.send(WebsocketAbilityEventName(e.AbilityName, e.Name), e.Payload)
	})
}
